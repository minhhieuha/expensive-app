package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-app/api/internal/db"
)

func setupTestServer(t *testing.T) (*http.ServeMux, *Service) {
	t.Helper()
	testDB := db.OpenTestDB(t)
	repo := NewRepo(testDB)
	service := NewService(repo, 720)
	handler := NewHandler(service, false)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux, service
}

func TestRegisterSuccessAndDefaultCategories(t *testing.T) {
	mux, service := setupTestServer(t)

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	cookie := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookie {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected session cookie to be set")
	}

	var user User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}

	// Verify categories were seeded in DB
	var catCount int
	err := service.repo.db.QueryRow("SELECT COUNT(*) FROM categories WHERE user_id = ?", user.ID).Scan(&catCount)
	if err != nil {
		t.Fatalf("failed to query categories: %v", err)
	}
	if catCount != 11 { // 8 expense + 3 income
		t.Errorf("expected 11 default categories seeded, got %d", catCount)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	mux, _ := setupTestServer(t)

	body := `{"email":"dup@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first registration failed: %d", rec.Code)
	}

	// Second registration with same email (different casing)
	body2 := `{"email":"DUP@example.com","password":"password456"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body2))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 conflict, got %d", rec2.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	mux, _ := setupTestServer(t)

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"invalid email", "not-an-email", "password123"},
		{"short password", "valid@example.com", "short"},
		{"long password", "valid@example.com", strings.Repeat("p", 73)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(map[string]string{"email": tt.email, "password": tt.password})
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(b))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected status 422, got %d", rec.Code)
			}
		})
	}
}

func TestLoginSuccessAndFailureMessages(t *testing.T) {
	mux, _ := setupTestServer(t)

	// Register user
	b := `{"email":"login@example.com","password":"mypassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(b))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Valid login
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(b))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRec.Code)
	}

	// Wrong password
	wrongPw := `{"email":"login@example.com","password":"wrongpassword"}`
	wrongReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(wrongPw))
	wrongRec := httptest.NewRecorder()
	mux.ServeHTTP(wrongRec, wrongReq)

	if wrongRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for wrong password, got %d", wrongRec.Code)
	}

	// Nonexistent email
	noUser := `{"email":"nonexistent@example.com","password":"wrongpassword"}`
	noUserReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(noUser))
	noUserRec := httptest.NewRecorder()
	mux.ServeHTTP(noUserRec, noUserReq)

	if noUserRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for nonexistent email, got %d", noUserRec.Code)
	}

	// Verify both 401 responses return the same error code and message
	if wrongRec.Body.String() != noUserRec.Body.String() {
		t.Errorf("error responses differ: wrong password (%s) vs nonexistent email (%s)",
			wrongRec.Body.String(), noUserRec.Body.String())
	}
}

func TestRateLimitOnLogin(t *testing.T) {
	mux, _ := setupTestServer(t)

	body := `{"email":"ratelimit@example.com","password":"wrongpassword"}`
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected status 401, got %d", i, rec.Code)
		}
	}

	// 6th attempt must be rate limited (429)
	req6 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req6.RemoteAddr = "192.168.1.100:12345"
	rec6 := httptest.NewRecorder()
	mux.ServeHTTP(rec6, req6)

	if rec6.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 on 6th attempt, got %d", rec6.Code)
	}
}

func TestLogoutAndMe(t *testing.T) {
	mux, _ := setupTestServer(t)

	// Register
	regBody := `{"email":"me@example.com","password":"password123"}`
	regReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(regBody))
	regRec := httptest.NewRecorder()
	mux.ServeHTTP(regRec, regReq)

	var sessionToken string
	for _, c := range regRec.Result().Cookies() {
		if c.Name == "session" {
			sessionToken = c.Value
			break
		}
	}

	// GET /api/me with session cookie
	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected GET /api/me status 200, got %d", meRec.Code)
	}

	// Logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d", logoutRec.Code)
	}

	// Try GET /api/me with old token -> must be 401
	meReq2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq2.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	meRec2 := httptest.NewRecorder()
	mux.ServeHTTP(meRec2, meReq2)

	if meRec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", meRec2.Code)
	}

	// GET /api/me without cookie -> must be 401
	meReqNoCookie := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meRecNoCookie := httptest.NewRecorder()
	mux.ServeHTTP(meRecNoCookie, meReqNoCookie)

	if meRecNoCookie.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cookie, got %d", meRecNoCookie.Code)
	}
}

func TestExpiredSession(t *testing.T) {
	mux, service := setupTestServer(t)

	// Register user
	res, err := service.Register(t.Context(), "expire@example.com", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Manually expire session in DB
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	tokenHash := HashToken(res.SessionToken)
	_, err = service.repo.db.Exec("UPDATE sessions SET expires_at = ? WHERE token_hash = ?", past, tokenHash)
	if err != nil {
		t.Fatalf("update session expiration failed: %v", err)
	}

	// GET /api/me with expired token
	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq.AddCookie(&http.Cookie{Name: "session", Value: res.SessionToken})
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired session, got %d", meRec.Code)
	}
}
func TestPasswordExactBoundaries(t *testing.T) {
	mux, _ := setupTestServer(t)

	// 7 bytes -> fail
	b7 := `{"email":"b7@example.com","password":"1234567"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(b7))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for 7 byte password, got %d", rec.Code)
	}

	// 8 bytes -> pass
	b8 := `{"email":"b8@example.com","password":"12345678"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(b8))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for 8 byte password, got %d", rec.Code)
	}

	// 72 bytes -> pass
	pw72 := strings.Repeat("a", 72)
	b72, _ := json.Marshal(map[string]string{"email": "b72@example.com", "password": pw72})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(b72))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for 72 byte password, got %d", rec.Code)
	}

	// 73 bytes -> fail
	pw73 := strings.Repeat("a", 73)
	b73, _ := json.Marshal(map[string]string{"email": "b73@example.com", "password": pw73})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(b73))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for 73 byte password, got %d", rec.Code)
	}
}

func TestDistinctSessionTokensPerLogin(t *testing.T) {
	mux, _ := setupTestServer(t)

	regBody := `{"email":"multisession@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(regBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register failed: %d", rec.Code)
	}

	token1 := getCookieValue(rec, "session")

	// Login again
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(regBody))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login failed: %d", loginRec.Code)
	}

	token2 := getCookieValue(loginRec, "session")

	if token1 == token2 {
		t.Error("expected different session tokens for consecutive logins")
	}

	// Both tokens should be valid for GET /api/me
	meReq1 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq1.AddCookie(&http.Cookie{Name: "session", Value: token1})
	meRec1 := httptest.NewRecorder()
	mux.ServeHTTP(meRec1, meReq1)
	if meRec1.Code != http.StatusOK {
		t.Errorf("expected token1 to be valid, got %d", meRec1.Code)
	}

	meReq2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq2.AddCookie(&http.Cookie{Name: "session", Value: token2})
	meRec2 := httptest.NewRecorder()
	mux.ServeHTTP(meRec2, meReq2)
	if meRec2.Code != http.StatusOK {
		t.Errorf("expected token2 to be valid, got %d", meRec2.Code)
	}
}

func getCookieValue(rec *httptest.ResponseRecorder, name string) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
