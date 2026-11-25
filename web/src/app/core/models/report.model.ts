export interface ReportSummary {
  total_expense: number;
  total_income: number;
  balance: number;
}

export interface CategoryReportItem {
  category_id: number;
  category_name: string;
  amount: number;
  percentage: number;
}

export interface TrendItem {
  month: string;
  total_expense: number;
  total_income: number;
}