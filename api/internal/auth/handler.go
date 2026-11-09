package auth

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"expense-app/api/internal/httpx"
)

type Handler struct {
	service      *Service
	cookieSecure bool
}

func NewHandler(service *Service, cookieSecure bool) *Handler {
	return &Handler{
		service:      service,
		cookieSecure: cookieSecure,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", h.handleRegister)
	mux.HandleFunc("POST /api/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", h.handleLogout)
	mux.HandleFunc("GET /api/me", h.handleMe)
}

type credentialsReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req credentialsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	result, err := h.service.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		if errors.Is(err, ErrDuplicateEmail) {
			httpx.WriteError(w, http.StatusConflict, "conflict", "Email đã được sử dụng", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	h.setSessionCookie(w, result.SessionToken, result.ExpiresAt)
	httpx.WriteJSON(w, http.StatusCreated, result.User)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req credentialsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	ip := clientIP(r)
	result, err := h.service.Login(r.Context(), ip, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrRateLimited) {
			httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Quá nhiều lần đăng nhập không thành công, vui lòng thử lại sau", nil)
			return
		}
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Email hoặc mật khẩu không chính xác", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	h.setSessionCookie(w, result.SessionToken, result.ExpiresAt)
	httpx.WriteJSON(w, http.StatusOK, result.User)
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}

	h.clearSessionCookie(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	user, err := h.service.GetCurrentUser(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Phiên đăng nhập đã hết hạn hoặc không hợp lệ", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
