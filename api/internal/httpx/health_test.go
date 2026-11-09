package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	mux := SetupRoutes()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET /api/health returns 200 and status ok",
			method:         http.MethodGet,
			path:           "/api/health",
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "POST /api/health returns 405 Method Not Allowed",
			method:         http.MethodPost,
			path:           "/api/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "PUT /api/health returns 405 Method Not Allowed",
			method:         http.MethodPut,
			path:           "/api/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "DELETE /api/health returns 405 Method Not Allowed",
			method:         http.MethodDelete,
			path:           "/api/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "PATCH /api/health returns 405 Method Not Allowed",
			method:         http.MethodPatch,
			path:           "/api/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "GET /api/health/subpath returns 404 Not Found",
			method:         http.MethodGet,
			path:           "/api/health/subpath",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "GET /unknown returns 404 Not Found",
			method:         http.MethodGet,
			path:           "/unknown",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				ct := rec.Header().Get("Content-Type")
				if ct != "application/json; charset=utf-8" {
					t.Errorf("expected content-type 'application/json; charset=utf-8', got %q", ct)
				}

				var resp HealthResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if resp.Status != tt.expectedBody {
					t.Errorf("expected status %q, got %q", tt.expectedBody, resp.Status)
				}
			}
		})
	}
}
