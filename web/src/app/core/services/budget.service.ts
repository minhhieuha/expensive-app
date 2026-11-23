import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Budget, UpsertBudgetRequest } from '../models/budget.model';

@Injectable({
  providedIn: 'root',
})
export class BudgetService {
  private readonly http = inject(HttpClient);

  getBudgets(month: string): Observable<Budget[]> {
    const params = new HttpParams().set('month', month);
    return this.http.get<Budget[]>('/api/budgets', { params });
  }

  upsertBudget(req: UpsertBudgetRequest): Observable<Budget> {
    return this.http.put<Budget>('/api/budgets', req);
  }

  deleteBudget(id: number): Observable<{ status: string }> {
    return this.http.delete<{ status: string }>(`/api/budgets/${id}`);
  }
}