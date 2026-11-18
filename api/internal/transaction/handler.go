package transaction

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

	mux.Handle("GET /api/transactions", wrap(h.handleList))
	mux.Handle("POST /api/transactions", wrap(h.handleCreate))
	mux.Handle("GET /api/transactions/{id}", wrap(h.handleGet))
	mux.Handle("PATCH /api/transactions/{id}", wrap(h.handleUpdate))
	mux.Handle("DELETE /api/transactions/{id}", wrap(h.handleDelete))
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	q := r.URL.Query()
	f := Filter{
		From:  q.Get("from"),
		To:    q.Get("to"),
		Type:  q.Get("type"),
		Query: q.Get("q"),
	}

	if catStr := q.Get("category_id"); catStr != "" {
		if catID, err := strconv.ParseInt(catStr, 10, 64); err == nil && catID > 0 {
			f.CategoryID = &catID
		} else {
			httpx.WriteError(w, http.StatusBadRequest, "bad_request", "category_id không hợp lệ", nil)
			return
		}
	}

	if pageStr := q.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			f.Page = page
		}
	}
	if sizeStr := q.Get("page_size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			f.PageSize = size
		}
	}

	res, err := h.service.List(r.Context(), userID, f)
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

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	tx, err := h.service.Create(r.Context(), userID, in)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) || errors.Is(err, ErrCategoryNotFound) || errors.Is(err, ErrCategoryKindMismatch) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, tx)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID giao dịch không hợp lệ", nil)
		return
	}

	tx, err := h.service.GetByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy giao dịch", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tx)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID giao dịch không hợp lệ", nil)
		return
	}

	var in UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	tx, err := h.service.Update(r.Context(), userID, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy giao dịch", nil)
			return
		}
		if errors.Is(err, ErrValidationFailed) || errors.Is(err, ErrCategoryNotFound) || errors.Is(err, ErrCategoryKindMismatch) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tx)
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
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID giao dịch không hợp lệ", nil)
		return
	}

	if err := h.service.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy giao dịch", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}