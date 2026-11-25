package report

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
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

func (s *Service) GetSummary(ctx context.Context, userID int64, month string) (*Summary, error) {
	if err := validateMonth(month); err != nil {
		return nil, err
	}
	return s.repo.GetSummary(ctx, userID, month)
}

func (s *Service) GetByCategory(ctx context.Context, userID int64, month string) ([]CategoryReportItem, error) {
	if err := validateMonth(month); err != nil {
		return nil, err
	}
	return s.repo.GetByCategory(ctx, userID, month)
}

func (s *Service) GetTrend(ctx context.Context, userID int64, month string, monthsCount int) ([]TrendItem, error) {
	if err := validateMonth(month); err != nil {
		return nil, err
	}
	if monthsCount <= 0 {
		monthsCount = 6
	}
	if monthsCount > 24 {
		return nil, fmt.Errorf("%w: số tháng tối đa là 24", ErrValidationFailed)
	}
	return s.repo.GetTrend(ctx, userID, month, monthsCount)
}