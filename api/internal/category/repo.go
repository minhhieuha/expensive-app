package category

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context, userID int64, kind string) ([]Category, error) {
	var query string
	var args []interface{}

	if kind != "" {
		query = `SELECT id, user_id, name, kind, created_at FROM categories WHERE user_id = ? AND kind = ? ORDER BY id ASC`
		args = []interface{}{userID, kind}
	} else {
		query = `SELECT id, user_id, name, kind, created_at FROM categories WHERE user_id = ? ORDER BY id ASC`
		args = []interface{}{userID}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var createdAtStr string
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Kind, &createdAtStr); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			c.CreatedAt = t
		}
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	if categories == nil {
		categories = []Category{}
	}
	return categories, nil
}

func (r *Repo) GetByID(ctx context.Context, userID int64, id int64) (*Category, error) {
	query := `SELECT id, user_id, name, kind, created_at FROM categories WHERE id = ? AND user_id = ?`
	row := r.db.QueryRowContext(ctx, query, id, userID)

	var c Category
	var createdAtStr string
	if err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Kind, &createdAtStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get category by id: %w", err)
	}

	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		c.CreatedAt = t
	}
	return &c, nil
}

func (r *Repo) Create(ctx context.Context, userID int64, name, kind, createdAt string) (*Category, error) {
	query := `INSERT INTO categories (user_id, name, kind, created_at) VALUES (?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, userID, name, kind, createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("create category: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	t, _ := time.Parse(time.RFC3339, createdAt)
	return &Category{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Kind:      kind,
		CreatedAt: t,
	}, nil
}

func (r *Repo) UpdateName(ctx context.Context, userID int64, id int64, name string) (*Category, error) {
	query := `UPDATE categories SET name = ? WHERE id = ? AND user_id = ?`
	res, err := r.db.ExecContext(ctx, query, name, id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("update category name: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get rows affected: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, userID, id)
}

func (r *Repo) CountTransactions(ctx context.Context, userID int64, categoryID int64) (int, error) {
	query := `SELECT COUNT(*) FROM transactions WHERE category_id = ? AND user_id = ?`
	var count int
	if err := r.db.QueryRowContext(ctx, query, categoryID, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count transactions by category: %w", err)
	}
	return count, nil
}

func (r *Repo) Delete(ctx context.Context, userID int64, id int64) error {
	query := `DELETE FROM categories WHERE id = ? AND user_id = ?`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) DeleteWithReplace(ctx context.Context, userID int64, id int64, replaceWithID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Chuyển toàn bộ transactions từ id cũ sang replaceWithID (cùng user_id)
	updateTxQuery := `UPDATE transactions SET category_id = ? WHERE category_id = ? AND user_id = ?`
	if _, err := tx.ExecContext(ctx, updateTxQuery, replaceWithID, id, userID); err != nil {
		return fmt.Errorf("reassign transactions: %w", err)
	}

	// 2. Xóa category cũ
	deleteCatQuery := `DELETE FROM categories WHERE id = ? AND user_id = ?`
	res, err := tx.ExecContext(ctx, deleteCatQuery, id, userID)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete with replace: %w", err)
	}
	return nil
}