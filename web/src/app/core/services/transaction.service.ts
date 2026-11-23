import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  Transaction,
  TransactionFilter,
  PaginatedTransactions,
  CreateTransactionRequest,
  UpdateTransactionRequest,
} from '../models/transaction.model';

@Injectable({
  providedIn: 'root',
})
export class TransactionService {
  private readonly http = inject(HttpClient);

  getTransactions(filter?: TransactionFilter): Observable<PaginatedTransactions> {
    let params = new HttpParams();
    if (filter) {
      if (filter.from) {
        params = params.set('from', filter.from);
      }
      if (filter.to) {
        params = params.set('to', filter.to);
      }
      if (filter.type) {
        params = params.set('type', filter.type);
      }
      if (filter.category_id) {
        params = params.set('category_id', filter.category_id.toString());
      }
      if (filter.q) {
        params = params.set('q', filter.q);
      }
      if (filter.page) {
        params = params.set('page', filter.page.toString());
      }
      if (filter.page_size) {
        params = params.set('page_size', filter.page_size.toString());
      }
    }
    return this.http.get<PaginatedTransactions>('/api/transactions', { params });
  }

  getTransaction(id: number): Observable<Transaction> {
    return this.http.get<Transaction>(`/api/transactions/${id}`);
  }

  createTransaction(req: CreateTransactionRequest): Observable<Transaction> {
    return this.http.post<Transaction>('/api/transactions', req);
  }

  updateTransaction(id: number, req: UpdateTransactionRequest): Observable<Transaction> {
    return this.http.patch<Transaction>(`/api/transactions/${id}`, req);
  }

  deleteTransaction(id: number): Observable<void> {
    return this.http.delete<void>(`/api/transactions/${id}`);
  }
}