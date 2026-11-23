import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { TransactionService } from './transaction.service';
import { Transaction, PaginatedTransactions } from '../models/transaction.model';

describe('TransactionService', () => {
  let service: TransactionService;
  let httpTesting: HttpTestingController;

  const mockTx: Transaction = {
    id: 1,
    user_id: 1,
    category_id: 1,
    category_name: 'Ăn uống',
    type: 'expense',
    amount: 50000,
    occurred_on: '2026-11-20',
    note: 'Cà phê sáng',
    created_at: '2026-11-20T08:00:00Z',
    updated_at: '2026-11-20T08:00:00Z',
  };

  const mockPaginated: PaginatedTransactions = {
    items: [mockTx],
    page: 1,
    page_size: 20,
    total: 1,
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(TransactionService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('fetches transactions with all filter parameters', () => {
    service.getTransactions({
      from: '2026-11-01',
      to: '2026-11-30',
      type: 'expense',
      category_id: 1,
      q: 'cà phê',
      page: 1,
      page_size: 20,
    }).subscribe((res) => {
      expect(res).toEqual(mockPaginated);
    });

    const req = httpTesting.expectOne((r) =>
      r.url === '/api/transactions' &&
      r.params.get('from') === '2026-11-01' &&
      r.params.get('to') === '2026-11-30' &&
      r.params.get('type') === 'expense' &&
      r.params.get('category_id') === '1' &&
      r.params.get('q') === 'cà phê' &&
      r.params.get('page') === '1' &&
      r.params.get('page_size') === '20'
    );
    expect(req.request.method).toBe('GET');
    req.flush(mockPaginated);
  });

  it('gets a single transaction by ID', () => {
    service.getTransaction(1).subscribe((tx) => {
      expect(tx).toEqual(mockTx);
    });

    const req = httpTesting.expectOne('/api/transactions/1');
    expect(req.request.method).toBe('GET');
    req.flush(mockTx);
  });

  it('creates a new transaction', () => {
    service.createTransaction({
      category_id: 1,
      type: 'expense',
      amount: 50000,
      occurred_on: '2026-11-20',
      note: 'Cà phê sáng',
    }).subscribe((tx) => {
      expect(tx).toEqual(mockTx);
    });

    const req = httpTesting.expectOne('/api/transactions');
    expect(req.request.method).toBe('POST');
    req.flush(mockTx);
  });

  it('updates an existing transaction', () => {
    service.updateTransaction(1, {
      category_id: 1,
      type: 'expense',
      amount: 60000,
      occurred_on: '2026-11-20',
      note: 'Cà phê sáng và bánh',
    }).subscribe((tx) => {
      expect(tx.amount).toBe(60000);
    });

    const req = httpTesting.expectOne('/api/transactions/1');
    expect(req.request.method).toBe('PATCH');
    req.flush({ ...mockTx, amount: 60000 });
  });

  it('deletes a transaction', () => {
    service.deleteTransaction(1).subscribe();

    const req = httpTesting.expectOne('/api/transactions/1');
    expect(req.request.method).toBe('DELETE');
    req.flush(null);
  });
});