package report_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-app/api/internal/auth"
	"expense-app/api/internal/category"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
	"expense-app/api/internal/report"
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
	catExp1     int64
	catExp2     int64
	catInc      int64
	txSvc       *transaction.Service
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	testDB := db.OpenTestDB(t)

	authRepo := auth.NewRepo(testDB)
	authSvc := auth.NewService(authRepo, 720)

	resA, err := authSvc.Register(t.Context(), "report_a@example.com", "password123")
	if err != nil {
		t.Fatalf("register A: %v", err)
	}

	resB, err := authSvc.Register(t.Context(), "report_b@example.com", "password123")
	if err != nil {
		t.Fatalf("register B: %v", err)
	}

	catRepo := category.NewRepo(testDB)
	catsA, _ := catRepo.List(t.Context(), resA.User.ID, "")
	var exp1, exp2, inc int64
	for _, c := range catsA {
		if c.Kind == "expense" {
			if exp1 == 0 {
				exp1 = c.ID
			} else if exp2 == 0 {
				exp2 = c.ID
			}
		} else if c.Kind == "income" && inc == 0 {
			inc = c.ID
		}
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
	reportRepo := report.NewRepo(testDB)
	reportSvc := report.NewService(reportRepo)
	reportHandler := report.NewHandler(reportSvc, authMid)
	reportHandler.RegisterRoutes(mux)

	txRepo := transaction.NewRepo(testDB)
	txSvc := transaction.NewService(txRepo, &catAdapter{repo: catRepo})

	return &testEnv{
		db:      testDB,
		mux:     mux,
		tokenA:  resA.SessionToken,
		userIDA: resA.User.ID,
		tokenB:  resB.SessionToken,
		userIDB: resB.User.ID,
		catExp1: exp1,
		catExp2: exp2,
		catInc:  inc,
		txSvc:   txSvc,
	}
}

func doReq(t *testing.T, mux *http.ServeMux, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(nil))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "expense-app")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestReport_Summary(t *testing.T) {
	env := setupTestEnv(t)

	// 1. Tháng rỗng không có dữ liệu -> 0
	rec := doReq(t, env.mux, "GET", "/api/reports/summary?month=2026-11", env.tokenA)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var s report.Summary
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if s.TotalExpense != 0 || s.TotalIncome != 0 || s.Balance != 0 {
		t.Fatalf("expected all zeros, got %+v", s)
	}

	// 2. Thêm giao dịch ngày biên: 01, 15, 30 (tháng 11 có 30 ngày)
	// Thu: 10,000,000 ngày 2026-11-01
	_, err := env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catInc,
		Type:       "income",
		Amount:     10000000,
		OccurredOn: "2026-11-01",
	})
	if err != nil {
		t.Fatalf("create income: %v", err)
	}

	// Chi 1: 2,000,000 ngày 2026-11-15
	_, err = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp1,
		Type:       "expense",
		Amount:     2000000,
		OccurredOn: "2026-11-15",
	})
	if err != nil {
		t.Fatalf("create expense 1: %v", err)
	}

	// Chi 2: 1,000,000 ngày 2026-11-30 (ngày cuối tháng)
	_, err = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp2,
		Type:       "expense",
		Amount:     1000000,
		OccurredOn: "2026-11-30",
	})
	if err != nil {
		t.Fatalf("create expense 2: %v", err)
	}

	// Chi 3: 5,000,000 ngày 2026-12-01 (tháng sau - không được tính)
	_, err = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp1,
		Type:       "expense",
		Amount:     5000000,
		OccurredOn: "2026-12-01",
	})
	if err != nil {
		t.Fatalf("create expense next month: %v", err)
	}

	// Kiểm tra summary tháng 2026-11
	rec = doReq(t, env.mux, "GET", "/api/reports/summary?month=2026-11", env.tokenA)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if s.TotalExpense != 3000000 {
		t.Errorf("expected expense 3000000, got %d", s.TotalExpense)
	}
	if s.TotalIncome != 10000000 {
		t.Errorf("expected income 10000000, got %d", s.TotalIncome)
	}
	if s.Balance != 7000000 {
		t.Errorf("expected balance 7000000, got %d", s.Balance)
	}
}

func TestReport_ByCategory(t *testing.T) {
	env := setupTestEnv(t)

	// Tạo 2 khoản chi: Cat1: 3,000,000 (75%), Cat2: 1,000,000 (25%) -> Tổng 4,000,000
	_, _ = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp1,
		Type:       "expense",
		Amount:     3000000,
		OccurredOn: "2026-11-10",
	})
	_, _ = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp2,
		Type:       "expense",
		Amount:     1000000,
		OccurredOn: "2026-11-20",
	})

	rec := doReq(t, env.mux, "GET", "/api/reports/by-category?month=2026-11", env.tokenA)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var items []report.CategoryReportItem
	_ = json.Unmarshal(rec.Body.Bytes(), &items)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// Item đầu tiên phải có amount lớn nhất
	if items[0].Amount != 3000000 || items[0].Percentage != 75 {
		t.Errorf("expected 3000000 and 75%%, got %+v", items[0])
	}
	if items[1].Amount != 1000000 || items[1].Percentage != 25 {
		t.Errorf("expected 1000000 and 25%%, got %+v", items[1])
	}
}

func TestReport_Trend(t *testing.T) {
	env := setupTestEnv(t)

	// Tạo giao dịch chi tháng 2026-09 và 2026-11
	_, _ = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp1,
		Type:       "expense",
		Amount:     1500000,
		OccurredOn: "2026-09-15",
	})
	_, _ = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp1,
		Type:       "expense",
		Amount:     2500000,
		OccurredOn: "2026-11-05",
	})

	rec := doReq(t, env.mux, "GET", "/api/reports/trend?month=2026-11&months=6", env.tokenA)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var trend []report.TrendItem
	_ = json.Unmarshal(rec.Body.Bytes(), &trend)

	if len(trend) != 6 {
		t.Fatalf("expected 6 months, got %d", len(trend))
	}

	// Tháng đầu tiên là 2026-06, tháng cuối cùng là 2026-11
	if trend[0].Month != "2026-06" || trend[5].Month != "2026-11" {
		t.Errorf("unexpected trend range: start=%s, end=%s", trend[0].Month, trend[5].Month)
	}

	// 2026-09 có expense 1500000
	if trend[3].Month != "2026-09" || trend[3].TotalExpense != 1500000 {
		t.Errorf("expected 2026-09 expense 1500000, got %+v", trend[3])
	}

	// 2026-11 có expense 2500000
	if trend[5].Month != "2026-11" || trend[5].TotalExpense != 2500000 {
		t.Errorf("expected 2026-11 expense 2500000, got %+v", trend[5])
	}
}

func TestReport_UserIsolation(t *testing.T) {
	env := setupTestEnv(t)

	// User A tạo giao dịch
	_, _ = env.txSvc.Create(t.Context(), env.userIDA, transaction.CreateInput{
		CategoryID: env.catExp1,
		Type:       "expense",
		Amount:     5000000,
		OccurredOn: "2026-11-10",
	})

	// User B truy vấn summary -> phải là 0
	recB := doReq(t, env.mux, "GET", "/api/reports/summary?month=2026-11", env.tokenB)
	var sB report.Summary
	_ = json.Unmarshal(recB.Body.Bytes(), &sB)
	if sB.TotalExpense != 0 {
		t.Errorf("User B saw User A's expense: %d", sB.TotalExpense)
	}

	// User B truy vấn by-category -> rỗng
	recCatsB := doReq(t, env.mux, "GET", "/api/reports/by-category?month=2026-11", env.tokenB)
	var catsB []report.CategoryReportItem
	_ = json.Unmarshal(recCatsB.Body.Bytes(), &catsB)
	if len(catsB) != 0 {
		t.Errorf("User B saw User A's categories: %+v", catsB)
	}
}