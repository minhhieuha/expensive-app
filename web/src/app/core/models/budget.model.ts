export interface Budget {
  id: number;
  user_id: number;
  category_id: number;
  category_name: string;
  month: string;
  amount: number;
  spent: number;
  status: 'ok' | 'warning' | 'exceeded';
}

export interface UpsertBudgetRequest {
  category_id: number;
  month: string;
  amount: number;
}