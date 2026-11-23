import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { BudgetService } from './budget.service';
import { Budget } from '../models/budget.model';

describe('BudgetService', () => {
  let service: BudgetService;
  let httpTesting: HttpTestingController;

  const mockBudget: Budget = {
    id: 1,
    user_id: 1,
    category_id: 1,
    category_name: 'Ăn uống',
    month: '2026-11',
    amount: 2000000,
    spent: 1200000,
    status: 'ok',
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(BudgetService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('gets budgets by month', () => {
    service.getBudgets('2026-11').subscribe((res) => {
      expect(res).toEqual([mockBudget]);
    });

    const req = httpTesting.expectOne('/api/budgets?month=2026-11');
    expect(req.request.method).toBe('GET');
    req.flush([mockBudget]);
  });

  it('upserts a budget', () => {
    service.upsertBudget({
      category_id: 1,
      month: '2026-11',
      amount: 2500000,
    }).subscribe((res) => {
      expect(res.amount).toBe(2500000);
    });

    const req = httpTesting.expectOne('/api/budgets');
    expect(req.request.method).toBe('PUT');
    req.flush({ ...mockBudget, amount: 2500000 });
  });

  it('deletes a budget', () => {
    service.deleteBudget(1).subscribe((res) => {
      expect(res.status).toBe('ok');
    });

    const req = httpTesting.expectOne('/api/budgets/1');
    expect(req.request.method).toBe('DELETE');
    req.flush({ status: 'ok' });
  });
});