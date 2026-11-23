package budget

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
	repo   *Repo
	catVal CategoryValidator
}

func NewService(repo *Repo, catVal CategoryValidator) *Service {
	return &Service{
		repo:   repo,
		catVal: catVal,
	}
}

func validateMonth(m string) error {
	m = strings.TrimSpace(m)
	if len(m) != 7 {
		return fmt.Errorf("%w: tháng phải có định dạng YYYY-MM", ErrValidationFailed)
	}
	t, err := time.Parse("2006-01", m)
	if err != nil || t.Format("2006-01") != m {
		return fmt.Errorf("%w: tháng không hợp lệ hoặc sai định dạng YYYY-MM", ErrValidationFailed)
	}
	return nil
}

func (s *Service) GetByMonth(ctx context.Context, userID int64, month string) ([]Budget, error) {
	if err := validateMonth(month); err != nil {
		return nil, err
	}
	return s.repo.GetByMonth(ctx, userID, month)
}

func (s *Service) Upsert(ctx context.Context, userID int64, in UpsertInput) (*Budget, error) {
	if err := validateMonth(in.Month); err != nil {
		return nil, err
	}

	if in.Amount <= 0 {
		return nil, fmt.Errorf("%w: số tiền ngân sách phải lớn hơn 0", ErrValidationFailed)
	}
	if in.Amount > 1000000000000 {
		return nil, fmt.Errorf("%w: số tiền ngân sách vượt quá hạn mức tối đa (1000 tỷ)", ErrValidationFailed)
	}

	// Validate category: phải tồn tại, thuộc user, và là loại expense
	cat, err := s.catVal.GetCategory(ctx, userID, in.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("%w: không tìm thấy danh mục hoặc danh mục không thuộc về bạn", ErrCategoryNotFound)
	}
	if cat.Kind != "expense" {
		return nil, fmt.Errorf("%w: chỉ có thể đặt ngân sách cho danh mục chi phí (expense)", ErrCategoryKindMismatch)
	}

	return s.repo.Upsert(ctx, userID, in.CategoryID, in.Month, in.Amount)
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}