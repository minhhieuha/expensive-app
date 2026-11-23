export interface Category {
  id: number;
  user_id: number;
  name: string;
  kind: 'expense' | 'income';
  created_at: string;
}

export interface CreateCategoryRequest {
  name: string;
  kind: 'expense' | 'income';
}

export interface UpdateCategoryRequest {
  name: string;
}