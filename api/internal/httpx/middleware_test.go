package httpx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-app/api/internal/auth"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
)

// SetupTestAuthService dựng một auth.Service thật trên DB in-memory dùng lại được.
func SetupTestAuthService(t *testing.T) *auth.Service {
	t.Helper()
	testDB := db.OpenTestDB(t)
	repo := auth.NewRepo(testDB)
	return auth.NewService(repo, 720)
}

// makeAuthValidator bọc auth.Service thành httpx.AuthSessionFunc
func makeAuthValidator(svc *auth.Service) httpx.AuthSessionFunc {
	return func(ctx context.Context, token string) (int64, error) {
		user, err := svc.GetCurrentUser(ctx, token)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}
}

// RegisterAndLogin đăng ký user và trả về session token dùng cho các test tiếp theo.
func RegisterAndLogin(t *testing.T, service *auth.Service, email, password string) string {
	t.Helper()
	result, err := service.Register(t.Context(), email, password)
	if err != nil {
		t.Fatalf("RegisterAndLogin: register failed: %v", err)
	}
	return result.SessionToken
}

func TestRequireAuth_NoCookie(t *testing.T) {
	svc := SetupTestAuthService(t)
	validator := makeAuthValidator(svc)

	protected := httpx.RequireAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_FakeCookie(t *testing.T) {
	svc := SetupTestAuthService(t)
	validator := makeAuthValidator(svc)

	protected := httpx.RequireAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "totallyfaketoken00000000000000000000000000000000000000000000000000"})
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for fake cookie, got %d", rec.Code)
	}
}

func TestRequireAuth_ValidCookie(t *testing.T) {
	svc := SetupTestAuthService(t)
	validator := makeAuthValidator(svc)
	token := RegisterAndLogin(t, svc, "auth-valid@example.com", "password123")

	var capturedUserID int64
	protected := httpx.RequireAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = httpx.UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedUserID == 0 {
		t.Error("expected userID to be set in context, got 0")
	}
}

func TestRequireCSRF_AllowsGetHead(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := httpx.RequireCSRF(inner)

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		req := httptest.NewRequest(method, "/", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("method %s: expected 200, got %d", method, rec.Code)
		}
	}
}

func TestRequireCSRF_BlocksWriteWithoutHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := httpx.RequireCSRF(inner)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := httptest.NewRequest(method, "/", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("method %s without header: expected 403, got %d", method, rec.Code)
		}
	}
}

func TestRequireCSRF_AllowsWriteWithHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := httpx.RequireCSRF(inner)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := httptest.NewRequest(method, "/", nil)
		req.Header.Set("X-Requested-With", "expense-app")
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("method %s with header: expected 200, got %d", method, rec.Code)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := httpx.SecurityHeaders(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "same-origin"},
		{"Content-Security-Policy", "default-src 'self'"},
	}
	for _, tt := range tests {
		if got := rec.Header().Get(tt.header); got != tt.expected {
			t.Errorf("header %s: expected %q, got %q", tt.header, tt.expected, got)
		}
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	wrapped := httpx.Recovery(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on panic, got %d", rec.Code)
	}
}

// ---- Bộ test cô lập User A vs User B ----
// UserIsolation là helper tái sử dụng ở các task sau (T05, T06...).

type UserIsolation struct {
	ServiceA *auth.Service
	TokenA   string
	UserIDA  int64
	ServiceB *auth.Service
	TokenB   string
	UserIDB  int64
}

func SetupUserIsolation(t *testing.T) *UserIsolation {
	t.Helper()
	testDB := db.OpenTestDB(t)
	repoA := auth.NewRepo(testDB)
	svcA := auth.NewService(repoA, 720)

	resA, err := svcA.Register(t.Context(), "user-a@example.com", "passwordA123")
	if err != nil {
		t.Fatalf("register user A failed: %v", err)
	}

	resB, err := svcA.Register(t.Context(), "user-b@example.com", "passwordB123")
	if err != nil {
		t.Fatalf("register user B failed: %v", err)
	}

	return &UserIsolation{
		ServiceA: svcA,
		TokenA:   resA.SessionToken,
		UserIDA:  resA.User.ID,
		ServiceB: svcA,
		TokenB:   resB.SessionToken,
		UserIDB:  resB.User.ID,
	}
}

func TestUserIsolation_MeEndpointReturnsOwnData(t *testing.T) {
	iso := SetupUserIsolation(t)
	mux := http.NewServeMux()

	authHandler := auth.NewHandler(iso.ServiceA, false)
	authHandler.RegisterRoutes(mux)

	// User A hits /api/me -> should see user-a@example.com
	meReqA := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReqA.AddCookie(&http.Cookie{Name: "session", Value: iso.TokenA})
	meRecA := httptest.NewRecorder()
	mux.ServeHTTP(meRecA, meReqA)
	if meRecA.Code != http.StatusOK {
		t.Fatalf("user A: expected 200, got %d", meRecA.Code)
	}
	if body := meRecA.Body.String(); !strings.Contains(body, "user-a@example.com") {
		t.Errorf("user A: expected user-a@example.com in response, got: %s", body)
	}

	// User B hits /api/me -> should see user-b@example.com, NOT user-a
	meReqB := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReqB.AddCookie(&http.Cookie{Name: "session", Value: iso.TokenB})
	meRecB := httptest.NewRecorder()
	mux.ServeHTTP(meRecB, meReqB)
	if meRecB.Code != http.StatusOK {
		t.Fatalf("user B: expected 200, got %d", meRecB.Code)
	}
	if body := meRecB.Body.String(); !strings.Contains(body, "user-b@example.com") {
		t.Errorf("user B: expected user-b@example.com in response, got: %s", body)
	}
	if body := meRecB.Body.String(); strings.Contains(body, "user-a@example.com") {
		t.Errorf("user B should NOT see user-a data, got: %s", body)
	}
}
