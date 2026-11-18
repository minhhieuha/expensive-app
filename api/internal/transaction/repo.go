package transaction

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

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return "%" + s + "%"
}

func (r *Repo) List(ctx context.Context, userID int64, f Filter) ([]Transaction, int, error) {
	whereClauses := []string{"t.user_id = ?"}
	args := []interface{}{userID}

	if f.From != "" {
		whereClauses = append(whereClauses, "t.occurred_on >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		whereClauses = append(whereClauses, "t.occurred_on <= ?")
		args = append(args, f.To)
	}
	if f.CategoryID != nil {
		whereClauses = append(whereClauses, "t.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.Type != "" {
		whereClauses = append(whereClauses, "t.type = ?")
		args = append(args, f.Type)
	}
	if f.Query != "" {
		whereClauses = append(whereClauses, "t.note LIKE ? ESCAPE '\\'")
		args = append(args, escapeLike(f.Query))
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Đếm tổng số bản ghi
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transactions t WHERE %s", whereSQL)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	// Lấy danh sách kèm tên danh mục
	listQuery := fmt.Sprintf(`
		SELECT t.id, t.user_id, t.category_id, COALESCE(c.name, ''), t.type, t.amount, t.occurred_on, t.note, t.created_at, t.updated_at
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id AND t.user_id = c.user_id
		WHERE %s
		ORDER BY t.occurred_on DESC, t.id DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	offset := (f.Page - 1) * f.PageSize
	listArgs := append(args, f.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var items []Transaction
	for rows.Next() {
		var tx Transaction
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.CategoryID,
			&tx.CategoryName,
			&tx.Type,
			&tx.Amount,
			&tx.OccurredOn,
			&tx.Note,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			tx.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			tx.UpdatedAt = t
		}
		items = append(items, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate transactions: %w", err)
	}

	if items == nil {
		items = []Transaction{}
	}
	return items, total, nil
}

func (r *Repo) GetByID(ctx context.Context, userID int64, id int64) (*Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.category_id, COALESCE(c.name, ''), t.type, t.amount, t.occurred_on, t.note, t.created_at, t.updated_at
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id AND t.user_id = c.user_id
		WHERE t.id = ? AND t.user_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id, userID)

	var tx Transaction
	var createdAtStr, updatedAtStr string
	if err := row.Scan(
		&tx.ID,
		&tx.UserID,
		&tx.CategoryID,
		&tx.CategoryName,
		&tx.Type,
		&tx.Amount,
		&tx.OccurredOn,
		&tx.Note,
		&createdAtStr,
		&updatedAtStr,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get transaction by id: %w", err)
	}

	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		tx.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		tx.UpdatedAt = t
	}
	return &tx, nil
}

func (r *Repo) Create(ctx context.Context, userID int64, categoryID int64, txType string, amount int64, occurredOn, note, now string) (*Transaction, error) {
	query := `
		INSERT INTO transactions (user_id, category_id, type, amount, occurred_on, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query, userID, categoryID, txType, amount, occurredOn, note, now, now)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return r.GetByID(ctx, userID, id)
}

func (r *Repo) Update(ctx context.Context, userID int64, id int64, categoryID int64, txType string, amount int64, occurredOn, note, now string) (*Transaction, error) {
	query := `
		UPDATE transactions
		SET category_id = ?, type = ?, amount = ?, occurred_on = ?, note = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`
	res, err := r.db.ExecContext(ctx, query, categoryID, txType, amount, occurredOn, note, now, id, userID)
	if err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
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

func (r *Repo) Delete(ctx context.Context, userID int64, id int64) error {
	query := `DELETE FROM transactions WHERE id = ? AND user_id = ?`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
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