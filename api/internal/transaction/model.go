package transaction

import (
	"errors"
	"time"
)

var (
	ErrNotFound             = errors.New("transaction not found")
	ErrValidationFailed     = errors.New("validation failed")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryKindMismatch = errors.New("category kind does not match transaction type")
)

type Transaction struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name,omitempty"`
	Type         string    `json:"type"` // "expense" | "income"
	Amount       int64     `json:"amount"` // số nguyên đơn vị nhỏ nhất (VND)
	OccurredOn   string    `json:"occurred_on"` // "YYYY-MM-DD"
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Filter struct {
	From       string // YYYY-MM-DD
	To         string // YYYY-MM-DD
	CategoryID *int64
	Type       string // "expense" | "income"
	Query      string // tìm kiếm trong note (LIKE %q%)
	Page       int
	PageSize   int
}

type PaginatedResponse struct {
	Items    []Transaction `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
}