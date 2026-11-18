package category_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"expense-app/api/internal/auth"
	"expense-app/api/internal/category"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
)

type testEnv struct {
	db         *sql.DB
	mux        *http.ServeMux
	svcA       *auth.Service
	tokenA     string
	userIDA    int64
	tokenB     string
	userIDB    int64
	catService *category.Service
	catRepo    *category.Repo
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	testDB := db.OpenTestDB(t)

	authRepo := auth.NewRepo(testDB)
	authSvc := auth.NewService(authRepo, 720)

	resA, err := authSvc.Register(t.Context(), "usera@example.com", "password123")
	if err != nil {
		t.Fatalf("register user A failed: %v", err)
	}

	resB, err := authSvc.Register(t.Context(), "userb@example.com", "password123")
	if err != nil {
		t.Fatalf("register user B failed: %v", err)
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
	catService := category.NewService(catRepo)
	catHandler := category.NewHandler(catService, requireAuthMid)

	mux := http.NewServeMux()
	catHandler.RegisterRoutes(mux)

	return &testEnv{
		db:         testDB,
		mux:        mux,
		svcA:       authSvc,
		tokenA:     resA.SessionToken,
		userIDA:    resA.User.ID,
		tokenB:     resB.SessionToken,
		userIDB:    resB.User.ID,
		catService: catService,
		catRepo:    catRepo,
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

func TestCategory_Unauthenticated(t *testing.T) {
	env := setupTestEnv(t)

	rec := doRequest(env.mux, http.MethodGet, "/api/categories", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCategory_CRUD(t *testing.T) {
	env := setupTestEnv(t)

	// 1. Create category
	createBody := map[string]string{
		"name": "Du lịch hè",
		"kind": "expense",
	}
	rec := doRequest(env.mux, http.MethodPost, "/api/categories", env.tokenA, createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created category.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if created.Name != "Du lịch hè" || created.Kind != "expense" {
		t.Errorf("unexpected category: %+v", created)
	}

	idStr := strconv.FormatInt(created.ID, 10)

	// 2. List categories (filter by kind=expense)
	recList := doRequest(env.mux, http.MethodGet, "/api/categories?kind=expense", env.tokenA, nil)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recList.Code)
	}
	var list []category.Category
	_ = json.Unmarshal(recList.Body.Bytes(), &list)
	found := false
	for _, c := range list {
		if c.ID == created.ID && c.Name == "Du lịch hè" {
			found = true
			break
		}
	}
	if !found {
		t.Error("created category not found in list")
	}

	// 3. Update category name
	updateBody := map[string]string{"name": "Du lịch thu"}
	recUpdate := doRequest(env.mux, http.MethodPatch, "/api/categories/"+idStr, env.tokenA, updateBody)
	if recUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}
	var updated category.Category
	_ = json.Unmarshal(recUpdate.Body.Bytes(), &updated)
	if updated.Name != "Du lịch thu" {
		t.Errorf("expected name 'Du lịch thu', got '%s'", updated.Name)
	}

	// 4. Delete category (no transactions)
	recDelete := doRequest(env.mux, http.MethodDelete, "/api/categories/"+idStr, env.tokenA, nil)
	if recDelete.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", recDelete.Code, recDelete.Body.String())
	}

	// 5. Verify deleted -> 404
	recCheck := doRequest(env.mux, http.MethodPatch, "/api/categories/"+idStr, env.tokenA, updateBody)
	if recCheck.Code != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", recCheck.Code)
	}
}

func TestCategory_Validation(t *testing.T) {
	env := setupTestEnv(t)

	// Tên rỗng -> 422
	rec := doRequest(env.mux, http.MethodPost, "/api/categories", env.tokenA, map[string]string{
		"name": "   ",
		"kind": "expense",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("empty name: expected 422, got %d", rec.Code)
	}

	// Tên quá 50 ký tự -> 422
	longName := "123456789012345678901234567890123456789012345678901"
	rec = doRequest(env.mux, http.MethodPost, "/api/categories", env.tokenA, map[string]string{
		"name": longName,
		"kind": "expense",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("long name: expected 422, got %d", rec.Code)
	}

	// Kind không hợp lệ -> 422
	rec = doRequest(env.mux, http.MethodPost, "/api/categories", env.tokenA, map[string]string{
		"name": "Test",
		"kind": "invalid",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid kind: expected 422, got %d", rec.Code)
	}

	// Trùng tên trong cùng user và cùng kind -> 409
	rec = doRequest(env.mux, http.MethodPost, "/api/categories", env.tokenA, map[string]string{
		"name": "Ăn uống",
		"kind": "expense",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate name: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCategory_UserIsolation(t *testing.T) {
	env := setupTestEnv(t)

	// User A tạo danh mục riêng
	created, err := env.catService.Create(t.Context(), env.userIDA, "Bí mật User A", "expense")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	idStr := strconv.FormatInt(created.ID, 10)

	// User B lấy danh sách -> không có danh mục của A
	recList := doRequest(env.mux, http.MethodGet, "/api/categories", env.tokenB, nil)
	var listB []category.Category
	_ = json.Unmarshal(recList.Body.Bytes(), &listB)
	for _, c := range listB {
		if c.ID == created.ID || c.Name == "Bí mật User A" {
			t.Errorf("user B should NOT see user A's category: %+v", c)
		}
	}

	// User B cố sửa danh mục của A -> 404
	recPatch := doRequest(env.mux, http.MethodPatch, "/api/categories/"+idStr, env.tokenB, map[string]string{"name": "Hacked"})
	if recPatch.Code != http.StatusNotFound {
		t.Errorf("user B patch user A category: expected 404, got %d", recPatch.Code)
	}

	// User B cố xóa danh mục của A -> 404
	recDelete := doRequest(env.mux, http.MethodDelete, "/api/categories/"+idStr, env.tokenB, nil)
	if recDelete.Code != http.StatusNotFound {
		t.Errorf("user B delete user A category: expected 404, got %d", recDelete.Code)
	}
}

func TestCategory_DeleteWithReplace(t *testing.T) {
	env := setupTestEnv(t)

	// Tạo 2 danh mục expense: C1 và C2
	c1, err := env.catService.Create(t.Context(), env.userIDA, "C1 Chi", "expense")
	if err != nil {
		t.Fatalf("create c1: %v", err)
	}
	c2, err := env.catService.Create(t.Context(), env.userIDA, "C2 Chi", "expense")
	if err != nil {
		t.Fatalf("create c2: %v", err)
	}

	// Tạo 1 danh mục income: C_inc
	cInc, err := env.catService.Create(t.Context(), env.userIDA, "C Thu", "income")
	if err != nil {
		t.Fatalf("create cInc: %v", err)
	}

	// Danh mục của user B
	cB, err := env.catService.Create(t.Context(), env.userIDB, "C của B", "expense")
	if err != nil {
		t.Fatalf("create cB: %v", err)
	}

	c1Str := strconv.FormatInt(c1.ID, 10)
	c2Str := strconv.FormatInt(c2.ID, 10)
	cIncStr := strconv.FormatInt(cInc.ID, 10)
	cBStr := strconv.FormatInt(cB.ID, 10)

	// Chèn 1 giao dịch trỏ tới C1 của User A
	now := time.Now().UTC().Format(time.RFC3339)
	insertTxQuery := `INSERT INTO transactions (user_id, category_id, type, amount, occurred_on, note, created_at, updated_at) VALUES (?, ?, 'expense', 50000, '2026-11-16', 'Test tx', ?, ?)`
	if _, err := env.db.Exec(insertTxQuery, env.userIDA, c1.ID, now, now); err != nil {
		t.Fatalf("insert test transaction failed: %v", err)
	}

	// 1. Xóa C1 không kèm replace_with -> 422 (vì có giao dịch)
	recNoRep := doRequest(env.mux, http.MethodDelete, "/api/categories/"+c1Str, env.tokenA, nil)
	if recNoRep.Code != http.StatusUnprocessableEntity {
		t.Errorf("delete with transactions and no replace_with: expected 422, got %d: %s", recNoRep.Code, recNoRep.Body.String())
	}

	// 2. Xóa C1 kèm replace_with=c1 (chính nó) -> 422
	recSelf := doRequest(env.mux, http.MethodDelete, "/api/categories/"+c1Str+"?replace_with="+c1Str, env.tokenA, nil)
	if recSelf.Code != http.StatusUnprocessableEntity {
		t.Errorf("self replacement: expected 422, got %d: %s", recSelf.Code, recSelf.Body.String())
	}

	// 3. Xóa C1 kèm replace_with=cInc (khác loại: expense vs income) -> 422
	recMismatch := doRequest(env.mux, http.MethodDelete, "/api/categories/"+c1Str+"?replace_with="+cIncStr, env.tokenA, nil)
	if recMismatch.Code != http.StatusUnprocessableEntity {
		t.Errorf("kind mismatch replacement: expected 422, got %d: %s", recMismatch.Code, recMismatch.Body.String())
	}

	// 4. Xóa C1 kèm replace_with=cB (danh mục của user B) -> 404
	recOtherUser := doRequest(env.mux, http.MethodDelete, "/api/categories/"+c1Str+"?replace_with="+cBStr, env.tokenA, nil)
	if recOtherUser.Code != http.StatusNotFound {
		t.Errorf("other user replacement: expected 404, got %d: %s", recOtherUser.Code, recOtherUser.Body.String())
	}

	// 5. Xóa C1 kèm replace_with=c2 (hợp lệ) -> 200
	recSuccess := doRequest(env.mux, http.MethodDelete, "/api/categories/"+c1Str+"?replace_with="+c2Str, env.tokenA, nil)
	if recSuccess.Code != http.StatusOK {
		t.Fatalf("valid replace: expected 200, got %d: %s", recSuccess.Code, recSuccess.Body.String())
	}

	// Xác minh C1 đã bị xóa
	recCheckC1 := doRequest(env.mux, http.MethodPatch, "/api/categories/"+c1Str, env.tokenA, map[string]string{"name": "New"})
	if recCheckC1.Code != http.StatusNotFound {
		t.Errorf("c1 should be deleted: expected 404, got %d", recCheckC1.Code)
	}

	// Xác minh giao dịch cũ đã chuyển sang C2
	var txCatID int64
	if err := env.db.QueryRow(`SELECT category_id FROM transactions WHERE user_id = ?`, env.userIDA).Scan(&txCatID); err != nil {
		t.Fatalf("query reassigned transaction failed: %v", err)
	}
	if txCatID != c2.ID {
		t.Errorf("expected transaction category_id = %d, got %d", c2.ID, txCatID)
	}
}