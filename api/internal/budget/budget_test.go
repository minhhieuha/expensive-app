package budget_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"expense-app/api/internal/auth"
	"expense-app/api/internal/budget"
	"expense-app/api/internal/category"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
	"expense-app/api/internal/transaction"
)

type budgetCatAdapter struct {
	repo *category.Repo
}

func (a *budgetCatAdapter) GetCategory(ctx context.Context, userID int64, categoryID int64) (*budget.CategoryInfo, error) {
	c, err := a.repo.GetByID(ctx, userID, categoryID)
	if err != nil {
		return nil, err
	}
	return &budget.CategoryInfo{ID: c.ID, Kind: c.Kind}, nil
}

type catValidatorAdapter struct {
	repo *category.Repo
}

func (a *catValidatorAdapter) GetCategory(ctx context.Context, userID int64, categoryID int64) (*transaction.CategoryInfo, error) {
	c, err := a.repo.GetByID(ctx, userID, categoryID)
	if err != nil {
		return nil, err
	}
	return &transaction.CategoryInfo{ID: c.ID, Kind: c.Kind}, nil
}

type testEnv struct {
	db          *sql.DB
	mux         *http.ServeMux
	tokenA      string
	userIDA     int64
	tokenB      string
	userIDB     int64
	catExpenseA int64
	catIncomeA  int64
	catExpenseB int64
	txSvc       *transaction.Service
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	testDB := db.OpenTestDB(t)

	authRepo := auth.NewRepo(testDB)
	authSvc := auth.NewService(authRepo, 720)

	resA, err := authSvc.Register(t.Context(), "budget_a@example.com", "password123")
	if err != nil {
		t.Fatalf("register A: %v", err)
	}

	resB, err := authSvc.Register(t.Context(), "budget_b@example.com", "password123")
	if err != nil {
		t.Fatalf("register B: %v", err)
	}

	catRepo := category.NewRepo(testDB)
	catsA, _ := catRepo.List(t.Context(), resA.User.ID, "")
	var expA, incA int64
	for _, c := range catsA {
		if c.Kind == "expense" && expA == 0 {
			expA = c.ID
		}
		if c.Kind == "income" && incA == 0 {
			incA = c.ID
		}
	}

	catsB, _ := catRepo.List(t.Context(), resB.User.ID, "expense")
	var expB int64
	if len(catsB) > 0 {
		expB = catsB[0].ID
	}

	authValidator := func(ctx context.Context, token string) (int64, error) {
		u, err := authSvc.GetCurrentUser(ctx, token)
		if err != nil {
			return 0, err
		}
		return u.ID, nil
	}
	authMid := httpx.RequireAuth(authValidator)

	mux := http.NewServeMux()
	budgetRepo := budget.NewRepo(testDB)
	budgetSvc := budget.NewService(budgetRepo, &budgetCatAdapter{repo: catRepo})
	budgetHandler := budget.NewHandler(budgetSvc, authMid)
	budgetHandler.RegisterRoutes(mux)

	txRepo := transaction.NewRepo(testDB)
	txSvc := transaction.NewService(txRepo, &catValidatorAdapter{repo: catRepo})

	return &testEnv{
		db:          testDB,
		mux:         mux,
		tokenA:      resA.SessionToken,
		userIDA:     resA.User.ID,
		tokenB:      resB.SessionToken,
		userIDB:     resB.User.ID,
		catExpenseA: expA,
		catIncomeA:  incA,
		catExpenseB: expB,
		txSvc:       txSvc,
	}
}

func doReq(t *testing.T, mux *http.ServeMux, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyBytes = b
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "expense-app")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestBudget_UpsertAndGet(t *testing.T) {
	env := setupTestEnv(t)

	// 1. Tạo budget cho tháng 2026-11
	rec := doReq(t, env.mux, "PUT", "/api/budgets", env.tokenA, map[string]any{
		"category_id": env.catExpenseA,
		"month":       "2026-11",
		"amount":      1000000,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var b budget.Budget
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("unmarshal budget: %v", err)
	}
	if b.Amount != 1000000 || b.Month != "2026-11" || b.Status != "ok" || b.Spent != 0 {
		t.Fatalf("unexpected budget response: %+v", b)
	}

	// 2. Lấy danh sách budgets tháng 2026-11
	recGet := doReq(t, env.mux, "GET", "/api/budgets?month=2026-11", env.tokenA, nil)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recGet.Code, recGet.Body.String())
	}
	var list []budget.Budget
	if err := json.Unmarshal(recGet.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 || list[0].Amount != 1000000 {
		t.Fatalf("expected 1 budget, got %+v", list)
	}

	// 3. Upsert lại để đổi amount thành 2,500,000
	recUpdate := doReq(t, env.mux, "PUT", "/api/budgets", env.tokenA, map[string]any{
		"category_id": env.catExpenseA,
		"month":       "2026-11",
		"amount":      2500000,
	})
	if recUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}
	var b2 budget.Budget
	_ = json.Unmarshal(recUpdate.Body.Bytes(), &b2)
	if b2.Amount != 2500000 || b2.ID != b.ID {
		t.Fatalf("expected updated amount 2500000 and same ID, got %+v", b2)
	}
}

func TestBudget_SpentAndStatusCalculations(t *testing.T) {
	env := setupTestEnv(t)

	// Đặt ngân sách 1,000,000 VND
	doReq(t, env.mux, "PUT", "/api/budgets", env.tokenA, map[string]any{
		"category_id": env.catExpenseA,
		"month":       "2026-11",
		"amount":      1000000,
	})

	// Case 1: Chi 500,000 (50%) -> spent=500000, status="ok"
	_, err := env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExpenseA,
		Type:       "expense",
		Amount:     500000,
		OccurredOn: "2026-11-05",
	})
	if err != nil {
		t.Fatalf("create tx 1: %v", err)
	}

	rec := doReq(t, env.mux, "GET", "/api/budgets?month=2026-11", env.tokenA, nil)
	var list []budget.Budget
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list[0].Spent != 500000 || list[0].Status != "ok" {
		t.Fatalf("expected spent 500000, status 'ok', got %+v", list[0])
	}

	// Case 2: Chi thêm 350,000 (tổng 850,000 = 85% >= 80%) -> status="warning"
	_, err = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExpenseA,
		Type:       "expense",
		Amount:     350000,
		OccurredOn: "2026-11-15",
	})
	if err != nil {
		t.Fatalf("create tx 2: %v", err)
	}

	rec = doReq(t, env.mux, "GET", "/api/budgets?month=2026-11", env.tokenA, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list[0].Spent != 850000 || list[0].Status != "warning" {
		t.Fatalf("expected spent 850000, status 'warning', got %+v", list[0])
	}

	// Case 3: Chi thêm 200,000 (tổng 1,050,000 > 1,000,000) -> status="exceeded"
	_, err = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExpenseA,
		Type:       "expense",
		Amount:     200000,
		OccurredOn: "2026-11-20",
	})
	if err != nil {
		t.Fatalf("create tx 3: %v", err)
	}

	rec = doReq(t, env.mux, "GET", "/api/budgets?month=2026-11", env.tokenA, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list[0].Spent != 1050000 || list[0].Status != "exceeded" {
		t.Fatalf("expected spent 1050000, status 'exceeded', got %+v", list[0])
	}

	// Case 4: Giao dịch tháng khác (2026-12) không làm tăng spent của tháng 2026-11
	_, err = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExpenseA,
		Type:       "expense",
		Amount:     999000,
		OccurredOn: "2026-12-01",
	})
	if err != nil {
		t.Fatalf("create tx in Dec: %v", err)
	}

	rec = doReq(t, env.mux, "GET", "/api/budgets?month=2026-11", env.tokenA, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list[0].Spent != 1050000 {
		t.Fatalf("December transaction must not affect November spent: got %d", list[0].Spent)
	}
}

func TestBudget_ValidationErrors(t *testing.T) {
	env := setupTestEnv(t)

	cases := []struct {
		name       string
		categoryID int64
		month      string
		amount     int64
		wantStatus int
	}{
		{"amount <= 0", env.catExpenseA, "2026-11", 0, http.StatusUnprocessableEntity},
		{"negative amount", env.catExpenseA, "2026-11", -5000, http.StatusUnprocessableEntity},
		{"amount > 10^12", env.catExpenseA, "2026-11", 1000000000001, http.StatusUnprocessableEntity},
		{"invalid month format", env.catExpenseA, "2026-1", 100000, http.StatusUnprocessableEntity},
		{"invalid month slash", env.catExpenseA, "2026/11", 100000, http.StatusUnprocessableEntity},
		{"income category rejected", env.catIncomeA, "2026-11", 100000, http.StatusUnprocessableEntity},
		{"non-existent category", 999999, "2026-11", 100000, http.StatusUnprocessableEntity},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doReq(t, env.mux, "PUT", "/api/budgets", env.tokenA, map[string]any{
				"category_id": tc.categoryID,
				"month":       tc.month,
				"amount":      tc.amount,
			})
			if rec.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBudget_UserIsolation(t *testing.T) {
	env := setupTestEnv(t)

	// User A tạo ngân sách
	recA := doReq(t, env.mux, "PUT", "/api/budgets", env.tokenA, map[string]any{
		"category_id": env.catExpenseA,
		"month":       "2026-11",
		"amount":      5000000,
	})
	if recA.Code != http.StatusOK {
		t.Fatalf("user A put budget: %s", recA.Body.String())
	}
	var bA budget.Budget
	_ = json.Unmarshal(recA.Body.Bytes(), &bA)

	// 1. User B gọi GET /api/budgets?month=2026-11 -> không thấy budget của User A
	recB := doReq(t, env.mux, "GET", "/api/budgets?month=2026-11", env.tokenB, nil)
	var listB []budget.Budget
	_ = json.Unmarshal(recB.Body.Bytes(), &listB)
	if len(listB) != 0 {
		t.Fatalf("User B must see empty budgets, got %+v", listB)
	}

	// 2. User B cố gắng xóa budget của User A -> 404
	recDel := doReq(t, env.mux, "DELETE", "/api/budgets/"+strconv.FormatInt(bA.ID, 10), env.tokenB, nil)
	if recDel.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when deleting other's budget, got %d", recDel.Code)
	}

	// 3. User B cố gắng đặt ngân sách cho category của User A -> 422 (không tìm thấy category của user B)
	recSteal := doReq(t, env.mux, "PUT", "/api/budgets", env.tokenB, map[string]any{
		"category_id": env.catExpenseA,
		"month":       "2026-11",
		"amount":      100000,
	})
	if recSteal.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 when using other's category, got %d", recSteal.Code)
	}
}

func TestBudget_Delete(t *testing.T) {
	env := setupTestEnv(t)

	// Tạo budget
	rec := doReq(t, env.mux, "PUT", "/api/budgets", env.tokenA, map[string]any{
		"category_id": env.catExpenseA,
		"month":       "2026-11",
		"amount":      500000,
	})
	var b budget.Budget
	_ = json.Unmarshal(rec.Body.Bytes(), &b)

	// Xóa thành công
	delRec := doReq(t, env.mux, "DELETE", "/api/budgets/"+strconv.FormatInt(b.ID, 10), env.tokenA, nil)
	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", delRec.Code)
	}

	// Xóa lại lần nữa -> 404
	delRec2 := doReq(t, env.mux, "DELETE", "/api/budgets/"+strconv.FormatInt(b.ID, 10), env.tokenA, nil)
	if delRec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on already deleted budget, got %d", delRec2.Code)
	}
}