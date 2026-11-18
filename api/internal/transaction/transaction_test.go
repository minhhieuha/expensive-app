package transaction_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"expense-app/api/internal/auth"
	"expense-app/api/internal/category"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
	"expense-app/api/internal/transaction"
)

type catAdapter struct {
	repo *category.Repo
}

func (a *catAdapter) GetCategory(ctx context.Context, userID int64, categoryID int64) (*transaction.CategoryInfo, error) {
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
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	testDB := db.OpenTestDB(t)

	authRepo := auth.NewRepo(testDB)
	authSvc := auth.NewService(authRepo, 720)

	resA, err := authSvc.Register(t.Context(), "tx_a@example.com", "password123")
	if err != nil {
		t.Fatalf("register A: %v", err)
	}

	resB, err := authSvc.Register(t.Context(), "tx_b@example.com", "password123")
	if err != nil {
		t.Fatalf("register B: %v", err)
	}

	authValidator := func(ctx context.Context, token string) (int64, error) {
		user, err := authSvc.GetCurrentUser(ctx, token)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}
	requireAuthMid := httpx.RequireAuth(authValidator)

	catRepo := category.NewRepo(testDB)

	// Lấy danh mục mặc định của A
	catsA, err := catRepo.List(t.Context(), resA.User.ID, "")
	if err != nil || len(catsA) == 0 {
		t.Fatalf("list default cats A: %v", err)
	}
	var catExpA, catIncA int64
	for _, c := range catsA {
		if c.Kind == "expense" && catExpA == 0 {
			catExpA = c.ID
		}
		if c.Kind == "income" && catIncA == 0 {
			catIncA = c.ID
		}
	}

	// Lấy danh mục mặc định của B
	catsB, err := catRepo.List(t.Context(), resB.User.ID, "expense")
	if err != nil || len(catsB) == 0 {
		t.Fatalf("list default cats B: %v", err)
	}
	catExpB := catsB[0].ID

	txRepo := transaction.NewRepo(testDB)
	txService := transaction.NewService(txRepo, &catAdapter{repo: catRepo})
	txHandler := transaction.NewHandler(txService, requireAuthMid)

	mux := http.NewServeMux()
	txHandler.RegisterRoutes(mux)

	return &testEnv{
		db:          testDB,
		mux:         mux,
		tokenA:      resA.SessionToken,
		userIDA:     resA.User.ID,
		tokenB:      resB.SessionToken,
		userIDB:     resB.User.ID,
		catExpenseA: catExpA,
		catIncomeA:  catIncA,
		catExpenseB: catExpB,
	}
}

func doRequest(mux *http.ServeMux, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "expense-app")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestTransaction_Unauthenticated(t *testing.T) {
	env := setupTestEnv(t)
	rec := doRequest(env.mux, http.MethodGet, "/api/transactions", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestTransaction_CRUD(t *testing.T) {
	env := setupTestEnv(t)

	// 1. Create transaction
	createBody := map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      150000,
		"occurred_on": "2026-11-18",
		"note":        "Ăn trưa cơm tấm",
	}
	rec := doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created transaction.Transaction
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if created.Amount != 150000 || created.OccurredOn != "2026-11-18" || created.Note != "Ăn trưa cơm tấm" {
		t.Errorf("unexpected transaction created: %+v", created)
	}

	txIDStr := strconv.FormatInt(created.ID, 10)

	// 2. Get transaction detail
	recGet := doRequest(env.mux, http.MethodGet, "/api/transactions/"+txIDStr, env.tokenA, nil)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recGet.Code)
	}
	var fetched transaction.Transaction
	_ = json.Unmarshal(recGet.Body.Bytes(), &fetched)
	if fetched.ID != created.ID || fetched.Amount != 150000 {
		t.Errorf("unexpected fetched transaction: %+v", fetched)
	}

	// 3. Update transaction
	updateBody := map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      180000,
		"occurred_on": "2026-11-18",
		"note":        "Ăn trưa + cafe",
	}
	recPatch := doRequest(env.mux, http.MethodPatch, "/api/transactions/"+txIDStr, env.tokenA, updateBody)
	if recPatch.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recPatch.Code, recPatch.Body.String())
	}
	var updated transaction.Transaction
	_ = json.Unmarshal(recPatch.Body.Bytes(), &updated)
	if updated.Amount != 180000 || updated.Note != "Ăn trưa + cafe" {
		t.Errorf("unexpected updated transaction: %+v", updated)
	}

	// 4. Delete transaction
	recDel := doRequest(env.mux, http.MethodDelete, "/api/transactions/"+txIDStr, env.tokenA, nil)
	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recDel.Code, recDel.Body.String())
	}

	// 5. Get after delete -> 404
	recCheck := doRequest(env.mux, http.MethodGet, "/api/transactions/"+txIDStr, env.tokenA, nil)
	if recCheck.Code != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", recCheck.Code)
	}
}

func TestTransaction_Validation(t *testing.T) {
	env := setupTestEnv(t)

	// Case 1: Amount <= 0 -> 422
	rec := doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      0,
		"occurred_on": "2026-11-18",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("amount <= 0: expected 422, got %d", rec.Code)
	}

	// Case 2: Amount > 10^12 -> 422
	rec = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      1000000000001,
		"occurred_on": "2026-11-18",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("amount > 1e12: expected 422, got %d", rec.Code)
	}

	// Case 3: Ngày sai format hoặc không tồn tại (2026-02-31) -> 422
	rec = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      50000,
		"occurred_on": "2026-02-31",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid date 2026-02-31: expected 422, got %d", rec.Code)
	}

	// Case 4: Category không tồn tại hoặc của user khác -> 422
	rec = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catExpenseB, // của User B
		"type":        "expense",
		"amount":      50000,
		"occurred_on": "2026-11-18",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("other user category: expected 422, got %d", rec.Code)
	}

	// Case 5: Category kind không khớp transaction type -> 422
	rec = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catIncomeA, // Income category nhưng type là expense
		"type":        "expense",
		"amount":      50000,
		"occurred_on": "2026-11-18",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("kind mismatch: expected 422, got %d", rec.Code)
	}
}

func TestTransaction_UserIsolation(t *testing.T) {
	env := setupTestEnv(t)

	// User A tạo giao dịch
	createBody := map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      200000,
		"occurred_on": "2026-11-18",
		"note":        "Giao dịch bí mật A",
	}
	recCreate := doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, createBody)
	var created transaction.Transaction
	_ = json.Unmarshal(recCreate.Body.Bytes(), &created)
	txIDStr := strconv.FormatInt(created.ID, 10)

	// 1. User B gọi GET list -> không thấy giao dịch của A
	recListB := doRequest(env.mux, http.MethodGet, "/api/transactions", env.tokenB, nil)
	var pageB transaction.PaginatedResponse
	_ = json.Unmarshal(recListB.Body.Bytes(), &pageB)
	for _, item := range pageB.Items {
		if item.ID == created.ID || item.Note == "Giao dịch bí mật A" {
			t.Errorf("user B should NOT see user A's transaction: %+v", item)
		}
	}

	// 2. User B gọi GET detail -> 404
	recGetB := doRequest(env.mux, http.MethodGet, "/api/transactions/"+txIDStr, env.tokenB, nil)
	if recGetB.Code != http.StatusNotFound {
		t.Errorf("user B get user A tx: expected 404, got %d", recGetB.Code)
	}

	// 3. User B gọi PATCH -> 404
	recPatchB := doRequest(env.mux, http.MethodPatch, "/api/transactions/"+txIDStr, env.tokenB, map[string]interface{}{
		"category_id": env.catExpenseB,
		"type":        "expense",
		"amount":      999999,
		"occurred_on": "2026-11-18",
	})
	if recPatchB.Code != http.StatusNotFound {
		t.Errorf("user B patch user A tx: expected 404, got %d", recPatchB.Code)
	}

	// 4. User B gọi DELETE -> 404
	recDelB := doRequest(env.mux, http.MethodDelete, "/api/transactions/"+txIDStr, env.tokenB, nil)
	if recDelB.Code != http.StatusNotFound {
		t.Errorf("user B delete user A tx: expected 404, got %d", recDelB.Code)
	}
}

func TestTransaction_QueryWithPercentAndUnderscore(t *testing.T) {
	env := setupTestEnv(t)

	// Tx 1: có ký tự % và _
	_ = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      10000,
		"occurred_on": "2026-11-18",
		"note":        "Mua 100% cafe_sua ngon",
	})

	// Tx 2: có chữ tương tự nhưng không có % và _
	_ = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
		"category_id": env.catExpenseA,
		"type":        "expense",
		"amount":      20000,
		"occurred_on": "2026-11-18",
		"note":        "Mua 100 cafe asua ngon",
	})

	// Tìm q=100% -> Chỉ trả về Tx 1
	recPercent := doRequest(env.mux, http.MethodGet, "/api/transactions?q="+url.QueryEscape("100%"), env.tokenA, nil)
	var pagePercent transaction.PaginatedResponse
	_ = json.Unmarshal(recPercent.Body.Bytes(), &pagePercent)
	if pagePercent.Total != 1 {
		t.Errorf("search 100%%: expected 1 item, got %d", pagePercent.Total)
	}
	if len(pagePercent.Items) > 0 && pagePercent.Items[0].Note != "Mua 100% cafe_sua ngon" {
		t.Errorf("search 100%%: unexpected item note: %s", pagePercent.Items[0].Note)
	}

	// Tìm q=cafe_sua -> Chỉ trả về Tx 1
	recUnderscore := doRequest(env.mux, http.MethodGet, "/api/transactions?q=cafe_sua", env.tokenA, nil)
	var pageUnderscore transaction.PaginatedResponse
	_ = json.Unmarshal(recUnderscore.Body.Bytes(), &pageUnderscore)
	if pageUnderscore.Total != 1 {
		t.Errorf("search cafe_sua: expected 1 item, got %d", pageUnderscore.Total)
	}
}

func TestTransaction_PaginationAndFilter(t *testing.T) {
	env := setupTestEnv(t)

	dates := []string{"2026-11-10", "2026-11-12", "2026-11-14", "2026-11-16", "2026-11-18"}
	for i, d := range dates {
		_ = doRequest(env.mux, http.MethodPost, "/api/transactions", env.tokenA, map[string]interface{}{
			"category_id": env.catExpenseA,
			"type":        "expense",
			"amount":      int64(10000 * (i + 1)),
			"occurred_on": d,
			"note":        "Giao dịch số " + strconv.Itoa(i+1),
		})
	}

	// 1. Phân trang: page=1, page_size=2
	recPage1 := doRequest(env.mux, http.MethodGet, "/api/transactions?page=1&page_size=2", env.tokenA, nil)
	var resPage1 transaction.PaginatedResponse
	_ = json.Unmarshal(recPage1.Body.Bytes(), &resPage1)
	if resPage1.Total != 5 || len(resPage1.Items) != 2 {
		t.Errorf("expected total=5, len=2; got total=%d, len=%d", resPage1.Total, len(resPage1.Items))
	}
	// Kiểm tra sắp xếp occurred_on DESC
	if resPage1.Items[0].OccurredOn != "2026-11-18" || resPage1.Items[1].OccurredOn != "2026-11-16" {
		t.Errorf("unexpected sort order: [0]=%s, [1]=%s", resPage1.Items[0].OccurredOn, resPage1.Items[1].OccurredOn)
	}

	// 2. Lọc theo from và to
	recFilter := doRequest(env.mux, http.MethodGet, "/api/transactions?from=2026-11-12&to=2026-11-15", env.tokenA, nil)
	var resFilter transaction.PaginatedResponse
	_ = json.Unmarshal(recFilter.Body.Bytes(), &resFilter)
	if resFilter.Total != 2 {
		t.Errorf("expected total=2 (ngay 12 va 14), got %d", resFilter.Total)
	}
}