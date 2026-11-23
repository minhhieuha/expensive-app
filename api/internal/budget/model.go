package budget

import "errors"

var (
	ErrNotFound             = errors.New("budget not found")
	ErrValidationFailed     = errors.New("validation failed")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryKindMismatch = errors.New("category kind must be expense")
)

type Budget struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Month        string `json:"month"` // "YYYY-MM"
	Amount       int64  `json:"amount"` // số nguyên (VND)
	Spent        int64  `json:"spent"`  // tổng chi trong tháng
	Status       string `json:"status"` // "ok" | "warning" | "exceeded"
}

type UpsertInput struct {
	CategoryID int64  `json:"category_id"`
	Month      string `json:"month"`
	Amount     int64  `json:"amount"`
}