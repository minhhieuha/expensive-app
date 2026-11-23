package budget

import (
	"encoding/json"
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

	mux.Handle("GET /api/budgets", wrap(h.handleGetByMonth))
	mux.Handle("PUT /api/budgets", wrap(h.handleUpsert))
	mux.Handle("DELETE /api/budgets/{id}", wrap(h.handleDelete))
}

func (h *Handler) handleGetByMonth(w http.ResponseWriter, r *http.Request) {
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

	budgets, err := h.service.GetByMonth(r.Context(), userID, month)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, budgets)
}

func (h *Handler) handleUpsert(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	var in UpsertInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	b, err := h.service.Upsert(r.Context(), userID, in)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) || errors.Is(err, ErrCategoryNotFound) || errors.Is(err, ErrCategoryKindMismatch) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, b)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID ngân sách không hợp lệ", nil)
		return
	}

	if err := h.service.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy ngân sách", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}