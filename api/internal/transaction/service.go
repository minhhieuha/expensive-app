package transaction

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type CategoryInfo struct {
	ID   int64
	Kind string
}

type CategoryValidator interface {
	GetCategory(ctx context.Context, userID int64, categoryID int64) (*CategoryInfo, error)
}

type Service struct {
	repo        *Repo
	catVal      CategoryValidator
}

func NewService(repo *Repo, catVal CategoryValidator) *Service {
	return &Service{
		repo:   repo,
		catVal: catVal,
	}
}

func validateDate(dateStr string) error {
	dateStr = strings.TrimSpace(dateStr)
	if len(dateStr) != 10 {
		return fmt.Errorf("%w: ngày phải có định dạng YYYY-MM-DD", ErrValidationFailed)
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("%w: ngày không hợp lệ hoặc sai định dạng YYYY-MM-DD", ErrValidationFailed)
	}
	if t.Format("2006-01-02") != dateStr {
		return fmt.Errorf("%w: ngày không tồn tại trên lịch", ErrValidationFailed)
	}
	return nil
}

func (s *Service) List(ctx context.Context, userID int64, f Filter) (*PaginatedResponse, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	} else if f.PageSize > 100 {
		f.PageSize = 100
	}

	f.Type = strings.TrimSpace(f.Type)
	if f.Type != "" && f.Type != "expense" && f.Type != "income" {
		return nil, fmt.Errorf("%w: type phải là 'expense' hoặc 'income'", ErrValidationFailed)
	}

	f.From = strings.TrimSpace(f.From)
	if f.From != "" {
		if err := validateDate(f.From); err != nil {
			return nil, err
		}
	}

	f.To = strings.TrimSpace(f.To)
	if f.To != "" {
		if err := validateDate(f.To); err != nil {
			return nil, err
		}
	}

	items, total, err := s.repo.List(ctx, userID, f)
	if err != nil {
		return nil, err
	}

	return &PaginatedResponse{
		Items:    items,
		Page:     f.Page,
		PageSize: f.PageSize,
		Total:    total,
	}, nil
}

func (s *Service) GetByID(ctx context.Context, userID int64, id int64) (*Transaction, error) {
	return s.repo.GetByID(ctx, userID, id)
}

type CreateInput struct {
	CategoryID int64  `json:"category_id"`
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	OccurredOn string `json:"occurred_on"`
	Note       string `json:"note"`
}

func (s *Service) validateTxInput(ctx context.Context, userID int64, categoryID int64, txType string, amount int64, occurredOn, note string) error {
	txType = strings.TrimSpace(txType)
	if txType != "expense" && txType != "income" {
		return fmt.Errorf("%w: type phải là 'expense' hoặc 'income'", ErrValidationFailed)
	}

	if amount <= 0 {
		return fmt.Errorf("%w: số tiền phải lớn hơn 0", ErrValidationFailed)
	}
	if amount > 1000000000000 {
		return fmt.Errorf("%w: số tiền vượt quá hạn mức tối đa (1000 tỷ)", ErrValidationFailed)
	}

	if err := validateDate(occurredOn); err != nil {
		return err
	}

	if len([]rune(note)) > 500 {
		return fmt.Errorf("%w: ghi chú tối đa 500 ký tự", ErrValidationFailed)
	}

	// Validate category
	cat, err := s.catVal.GetCategory(ctx, userID, categoryID)
	if err != nil {
		return fmt.Errorf("%w: không tìm thấy danh mục hoặc danh mục không thuộc về bạn", ErrCategoryNotFound)
	}

	if cat.Kind != txType {
		return fmt.Errorf("%w: loại giao dịch (%s) không khớp với loại danh mục (%s)", ErrCategoryKindMismatch, txType, cat.Kind)
	}

	return nil
}

func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (*Transaction, error) {
	if err := s.validateTxInput(ctx, userID, in.CategoryID, in.Type, in.Amount, in.OccurredOn, in.Note); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return s.repo.Create(ctx, userID, in.CategoryID, in.Type, in.Amount, in.OccurredOn, in.Note, now)
}

type UpdateInput struct {
	CategoryID int64  `json:"category_id"`
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	OccurredOn string `json:"occurred_on"`
	Note       string `json:"note"`
}

func (s *Service) Update(ctx context.Context, userID int64, id int64, in UpdateInput) (*Transaction, error) {
	// Kiểm tra transaction hiện tại có tồn tại không
	_, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if err := s.validateTxInput(ctx, userID, in.CategoryID, in.Type, in.Amount, in.OccurredOn, in.Note); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return s.repo.Update(ctx, userID, id, in.CategoryID, in.Type, in.Amount, in.OccurredOn, in.Note, now)
}

func (s *Service) Delete(ctx context.Context, userID int64, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}