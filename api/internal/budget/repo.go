package budget

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) GetByMonth(ctx context.Context, userID int64, month string) ([]Budget, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("invalid month: %w", err)
	}
	startStr := t.Format("2006-01") + "-01"
	nextMonth := t.AddDate(0, 1, 0)
	endStr := nextMonth.Format("2006-01") + "-01"

	query := `
		SELECT
			b.id,
			b.user_id,
			b.category_id,
			c.name,
			b.month,
			b.amount,
			COALESCE((
				SELECT SUM(t.amount)
				FROM transactions t
				WHERE t.user_id = b.user_id
				  AND t.category_id = b.category_id
				  AND t.type = 'expense'
				  AND t.occurred_on >= ? AND t.occurred_on < ?
			), 0) AS spent
		FROM budgets b
		JOIN categories c ON c.id = b.category_id AND c.user_id = b.user_id
		WHERE b.user_id = ? AND b.month = ?
		ORDER BY c.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, startStr, endStr, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.CategoryID,
			&b.CategoryName,
			&b.Month,
			&b.Amount,
			&b.Spent,
		); err != nil {
			return nil, err
		}
		b.Status = computeStatus(b.Spent, b.Amount)
		budgets = append(budgets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if budgets == nil {
		budgets = []Budget{}
	}

	return budgets, nil
}

func computeStatus(spent, amount int64) string {
	if spent > amount {
		return "exceeded"
	}
	if spent*100 >= amount*80 {
		return "warning"
	}
	return "ok"
}

func (r *Repo) Upsert(ctx context.Context, userID, categoryID int64, month string, amount int64) (*Budget, error) {
	query := `
		INSERT INTO budgets (user_id, category_id, month, amount)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, category_id, month) DO UPDATE SET
			amount = excluded.amount
		RETURNING id, user_id, category_id, month, amount
	`

	var b Budget
	err := r.db.QueryRowContext(ctx, query, userID, categoryID, month, amount).Scan(
		&b.ID,
		&b.UserID,
		&b.CategoryID,
		&b.Month,
		&b.Amount,
	)
	if err != nil {
		return nil, err
	}

	// Lấy category name và spent để trả về đầy đủ
	t, err := time.Parse("2006-01", month)
	if err == nil {
		startStr := t.Format("2006-01") + "-01"
		endStr := t.AddDate(0, 1, 0).Format("2006-01") + "-01"
		_ = r.db.QueryRowContext(ctx, `
			SELECT c.name,
			       COALESCE((
			           SELECT SUM(amount)
			           FROM transactions
			           WHERE user_id = ? AND category_id = ? AND type = 'expense' AND occurred_on >= ? AND occurred_on < ?
			       ), 0)
			FROM categories c
			WHERE c.id = ? AND c.user_id = ?
		`, userID, categoryID, startStr, endStr, categoryID, userID).Scan(&b.CategoryName, &b.Spent)
	}

	b.Status = computeStatus(b.Spent, b.Amount)
	return &b, nil
}

func (r *Repo) Delete(ctx context.Context, userID, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM budgets WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}