package category

import (
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("category not found")
	ErrDuplicateName    = errors.New("category name already exists")
	ErrValidationFailed = errors.New("validation failed")
	ErrHasTransactions  = errors.New("category has associated transactions")
	ErrKindMismatch     = errors.New("replacement category kind mismatch")
	ErrSelfReplacement  = errors.New("cannot replace category with itself")
)

type Category struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"` // "expense" | "income"
	CreatedAt time.Time `json:"created_at"`
}