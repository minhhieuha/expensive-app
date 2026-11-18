package category

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

	mux.Handle("GET /api/categories", wrap(h.handleList))
	mux.Handle("POST /api/categories", wrap(h.handleCreate))
	mux.Handle("PATCH /api/categories/{id}", wrap(h.handleUpdate))
	mux.Handle("DELETE /api/categories/{id}", wrap(h.handleDelete))
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	kind := r.URL.Query().Get("kind")
	categories, err := h.service.List(r.Context(), userID, kind)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, categories)
}

type createCategoryReq struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	userID := httpx.UserID(r.Context())
	if userID == 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
		return
	}

	var req createCategoryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	c, err := h.service.Create(r.Context(), userID, req.Name, req.Kind)
	if err != nil {
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		if errors.Is(err, ErrDuplicateName) {
			httpx.WriteError(w, http.StatusConflict, "conflict", "Tên danh mục đã tồn tại trong nhóm này", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, c)
}

type updateCategoryReq struct {
	Name string `json:"name"`
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
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID danh mục không hợp lệ", nil)
		return
	}

	var req updateCategoryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Dữ liệu JSON không hợp lệ", nil)
		return
	}

	c, err := h.service.UpdateName(r.Context(), userID, id, req.Name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy danh mục", nil)
			return
		}
		if errors.Is(err, ErrValidationFailed) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
			return
		}
		if errors.Is(err, ErrDuplicateName) {
			httpx.WriteError(w, http.StatusConflict, "conflict", "Tên danh mục đã tồn tại trong nhóm này", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, c)
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
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID danh mục không hợp lệ", nil)
		return
	}

	var replaceWithID *int64
	if repStr := r.URL.Query().Get("replace_with"); repStr != "" {
		repID, err := strconv.ParseInt(repStr, 10, 64)
		if err != nil || repID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "bad_request", "ID replace_with không hợp lệ", nil)
			return
		}
		replaceWithID = &repID
	}

	if err := h.service.Delete(r.Context(), userID, id, replaceWithID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy danh mục", nil)
			return
		}
		if errors.Is(err, ErrSelfReplacement) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Không thể thay thế danh mục bằng chính nó", nil)
			return
		}
		if errors.Is(err, ErrKindMismatch) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Danh mục thay thế phải cùng loại thu/chi", nil)
			return
		}
		if errors.Is(err, ErrHasTransactions) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Danh mục đang có giao dịch, cần chỉ định danh mục thay thế", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}