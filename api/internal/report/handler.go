package report

import (
	"errors"
	"net/http"
	"strconv"

	"expense-app/api/internal/httpx"
)

type Handler struct {
	service *Service
	authMid func(http.Handler) http.Handler
}

func NewHandler(service *Service, authMid func(http.Handler) http.Handler) *Handler {
	return &Handler{
		service: service,
		authMid: authMid,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	wrap := func(hf http.HandlerFunc) http.Handler {
		if h.authMid != nil {
			return h.authMid(hf)
		}
		return hf
	}

	mux.Handle("GET /api/reports/summary", wrap(h.handleSummary))
	mux.Handle("GET /api/reports/by-category", wrap(h.handleByCategory))
	mux.Handle("GET /api/reports/trend", wrap(h.handleTrend))
}

func (h *Handler) handleSummary(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Tham số month là bắt buộc (YYYY-MM)", nil)
		return
	}

	res, err := h.service.GetSummary(r.Context(), userID, month)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) handleByCategory(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Tham số month là bắt buộc (YYYY-MM)", nil)
		return
	}

	res, err := h.service.GetByCategory(r.Context(), userID, month)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) handleTrend(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Tham số month là bắt buộc (YYYY-MM)", nil)
		return
	}

	monthsCount := 6
	if mStr := r.URL.Query().Get("months"); mStr != "" {
		if val, err := strconv.Atoi(mStr); err == nil && val > 0 {
			monthsCount = val
		}
	}

	res, err := h.service.GetTrend(r.Context(), userID, month, monthsCount)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}