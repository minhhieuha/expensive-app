package httpx

import (
	"context"
	"log"
	"net/http"
)

// contextKey dùng unexported type để tránh collision với package khác.
type contextKey int

const userIDKey contextKey = 1

// UserID lấy userID từ context. Trả về 0 nếu chưa có (request chưa qua auth middleware).
func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

// setUserID đặt userID vào context (chỉ dùng bởi middleware).
func setUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// AuthSessionFunc là kiểu hàm validate phiên; nhận token, trả userID và lỗi.
// Đây là cách đơn giản nhất để tránh import cycle mà vẫn testable.
type AuthSessionFunc func(ctx context.Context, token string) (int64, error)

// RequireAuth là middleware xác thực phiên. Nếu hợp lệ, đặt userID vào context.
// Nếu không hợp lệ, trả 401 và dừng chuỗi middleware.
// Nhận AuthSessionFunc để tránh import cycle với package auth.
func RequireAuth(validateFn AuthSessionFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value == "" {
				WriteError(w, http.StatusUnauthorized, "unauthenticated", "Chưa đăng nhập", nil)
				return
			}

			userID, err := validateFn(r.Context(), cookie.Value)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "unauthenticated", "Phiên đăng nhập đã hết hạn hoặc không hợp lệ", nil)
				return
			}

			ctx := setUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireCSRF là middleware kiểm tra header X-Requested-With trên mọi request ghi.
// GET và HEAD được bỏ qua. Thiếu header trả 403.
func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if r.Header.Get("X-Requested-With") != "expense-app" {
				WriteError(w, http.StatusForbidden, "forbidden", "Header X-Requested-With: expense-app bắt buộc với mọi request ghi", nil)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders thêm các header bảo mật cơ bản vào mọi response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// Recovery bắt panic và trả 500, không để server crash.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC recovered: %v", rec)
				WriteError(w, http.StatusInternalServerError, "internal", "Lỗi hệ thống nội bộ", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain kết nối nhiều middleware, áp dụng từ ngoài vào trong (đầu tiên được gọi trước).
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
