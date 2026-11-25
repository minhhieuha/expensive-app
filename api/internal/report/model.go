package report

import "errors"

var (
	ErrValidationFailed = errors.New("validation failed")
)

type Summary struct {
	TotalExpense int64 `json:"total_expense"`
	TotalIncome  int64 `json:"total_income"`
	Balance      int64 `json:"balance"` // total_income - total_expense
}

type CategoryReportItem struct {
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Amount       int64  `json:"amount"`     // Tổng tiền chi của danh mục (VND)
	Percentage   int    `json:"percentage"` // % trên tổng chi tiêu (0-100)
}

type TrendItem struct {
	Month        string `json:"month"` // "YYYY-MM"
	TotalExpense int64  `json:"total_expense"`
	TotalIncome  int64  `json:"total_income"`
}