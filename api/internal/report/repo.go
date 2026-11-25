package report

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

func parseMonthRange(month string) (string, string, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", fmt.Errorf("invalid month: %w", err)
	}
	startStr := t.Format("2006-01") + "-01"
	endStr := t.AddDate(0, 1, 0).Format("2006-01") + "-01"
	return startStr, endStr, nil
}

func (r *Repo) GetSummary(ctx context.Context, userID int64, month string) (*Summary, error) {
	startStr, endStr, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income
		FROM transactions
		WHERE user_id = ? AND occurred_on >= ? AND occurred_on < ?
	`

	var s Summary
	err = r.db.QueryRowContext(ctx, query, userID, startStr, endStr).Scan(&s.TotalExpense, &s.TotalIncome)
	if err != nil {
		return nil, err
	}
	s.Balance = s.TotalIncome - s.TotalExpense
	return &s, nil
}

func (r *Repo) GetByCategory(ctx context.Context, userID int64, month string) ([]CategoryReportItem, error) {
	startStr, endStr, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			c.id,
			c.name,
			COALESCE(SUM(t.amount), 0) AS amount
		FROM categories c
		JOIN transactions t ON t.category_id = c.id AND t.user_id = c.user_id
		WHERE c.user_id = ?
		  AND t.type = 'expense'
		  AND t.occurred_on >= ? AND t.occurred_on < ?
		GROUP BY c.id, c.name
		ORDER BY amount DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CategoryReportItem
	var totalExpense int64
	for rows.Next() {
		var item CategoryReportItem
		if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.Amount); err != nil {
			return nil, err
		}
		totalExpense += item.Amount
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if items == nil {
		items = []CategoryReportItem{}
	}

	// Tính percentage
	if totalExpense > 0 {
		for i := range items {
			items[i].Percentage = int((items[i].Amount * 100) / totalExpense)
		}
	}

	return items, nil
}

func (r *Repo) GetTrend(ctx context.Context, userID int64, endMonth string, monthsCount int) ([]TrendItem, error) {
	endT, err := time.Parse("2006-01", endMonth)
	if err != nil {
		return nil, fmt.Errorf("invalid month: %w", err)
	}

	startT := endT.AddDate(0, -(monthsCount - 1), 0)
	startStr := startT.Format("2006-01") + "-01"
	endStr := endT.AddDate(0, 1, 0).Format("2006-01") + "-01"

	query := `
		SELECT
			substr(occurred_on, 1, 7) AS m,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income
		FROM transactions
		WHERE user_id = ? AND occurred_on >= ? AND occurred_on < ?
		GROUP BY m
	`

	rows, err := r.db.QueryContext(ctx, query, userID, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monthData := make(map[string]TrendItem)
	for rows.Next() {
		var item TrendItem
		if err := rows.Scan(&item.Month, &item.TotalExpense, &item.TotalIncome); err != nil {
			return nil, err
		}
		monthData[item.Month] = item
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Đảm bảo đủ monthsCount tháng theo thứ tự thời gian
	result := make([]TrendItem, 0, monthsCount)
	cur := startT
	for i := 0; i < monthsCount; i++ {
		mStr := cur.Format("2006-01")
		if item, ok := monthData[mStr]; ok {
			result = append(result, item)
		} else {
			result = append(result, TrendItem{
				Month:        mStr,
				TotalExpense: 0,
				TotalIncome:  0,
			})
		}
		cur = cur.AddDate(0, 1, 0)
	}

	return result, nil
}