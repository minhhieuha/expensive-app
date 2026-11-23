export interface Transaction {
  id: number;
  user_id: number;
  category_id: number;
  category_name?: string;
  type: 'expense' | 'income';
  amount: number;
  occurred_on: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface TransactionFilter {
  from?: string;
  to?: string;
  category_id?: number;
  type?: 'expense' | 'income';
  q?: string;
  page?: number;
  page_size?: number;
}

export interface PaginatedTransactions {
  items: Transaction[];
  page: number;
  page_size: number;
  total: number;
}

export interface CreateTransactionRequest {
  category_id: number;
  type: 'expense' | 'income';
  amount: number;
  occurred_on: string;
  note?: string;
}

export interface UpdateTransactionRequest {
  category_id: number;
  type: 'expense' | 'income';
  amount: number;
  occurred_on: string;
  note?: string;
}