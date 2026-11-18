package category

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

func (s *Service) List(ctx context.Context, userID int64, kind string) ([]Category, error) {
	kind = strings.TrimSpace(kind)
	if kind != "" && kind != "expense" && kind != "income" {
		return nil, fmt.Errorf("%w: kind phải là 'expense' hoặc 'income'", ErrValidationFailed)
	}
	return s.repo.List(ctx, userID, kind)
}

func (s *Service) Create(ctx context.Context, userID int64, name, kind string) (*Category, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 50 {
		return nil, fmt.Errorf("%w: tên danh mục phải từ 1 đến 50 ký tự", ErrValidationFailed)
	}

	kind = strings.TrimSpace(kind)
	if kind != "expense" && kind != "income" {
		return nil, fmt.Errorf("%w: kind phải là 'expense' hoặc 'income'", ErrValidationFailed)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return s.repo.Create(ctx, userID, name, kind, now)
}

func (s *Service) UpdateName(ctx context.Context, userID int64, id int64, name string) (*Category, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 50 {
		return nil, fmt.Errorf("%w: tên danh mục phải từ 1 đến 50 ký tự", ErrValidationFailed)
	}

	return s.repo.UpdateName(ctx, userID, id, name)
}

func (s *Service) Delete(ctx context.Context, userID int64, id int64, replaceWithID *int64) error {
	target, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}

	if replaceWithID != nil {
		if *replaceWithID == id {
			return ErrSelfReplacement
		}

		rep, err := s.repo.GetByID(ctx, userID, *replaceWithID)
		if err != nil {
			return err
		}

		if rep.Kind != target.Kind {
			return ErrKindMismatch
		}

		return s.repo.DeleteWithReplace(ctx, userID, id, *replaceWithID)
	}

	// Nếu không có replaceWithID, kiểm tra xem có giao dịch không
	count, err := s.repo.CountTransactions(ctx, userID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrHasTransactions
	}

	return s.repo.Delete(ctx, userID, id)
}