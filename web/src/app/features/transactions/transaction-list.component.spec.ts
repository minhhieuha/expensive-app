import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { of } from 'rxjs';
import { TransactionListComponent } from './transaction-list.component';
import { TransactionService } from '../../core/services/transaction.service';
import { CategoryService } from '../../core/services/category.service';
import { Category } from '../../core/models/category.model';
import { Transaction, PaginatedTransactions } from '../../core/models/transaction.model';

describe('TransactionListComponent', () => {
  let component: TransactionListComponent;
  let fixture: ComponentFixture<TransactionListComponent>;
  let mockTxService: Partial<TransactionService>;
  let mockCatService: Partial<CategoryService>;

  const mockCategories: Category[] = [
    { id: 1, user_id: 1, name: 'Ăn uống', kind: 'expense', created_at: '2026-11-20T00:00:00Z' },
    { id: 2, user_id: 1, name: 'Lương', kind: 'income', created_at: '2026-11-20T00:00:00Z' },
  ];

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

  beforeEach(async () => {
    mockTxService = {
      getTransactions: vi.fn().mockReturnValue(of(mockPaginated)),
      deleteTransaction: vi.fn().mockReturnValue(of(undefined)),
    };

    mockCatService = {
      getCategories: vi.fn().mockReturnValue(of(mockCategories)),
    };

    await TestBed.configureTestingModule({
      imports: [TransactionListComponent],
      providers: [
        provideRouter([]),
        { provide: TransactionService, useValue: mockTxService },
        { provide: CategoryService, useValue: mockCatService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TransactionListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('creates component and loads transactions and categories', () => {
    expect(component).toBeTruthy();
    expect(mockTxService.getTransactions).toHaveBeenCalled();
    expect(mockCatService.getCategories).toHaveBeenCalled();
    expect(component.paginatedData().items.length).toBe(1);
  });

  it('computes totalPages correctly', () => {
    expect(component.totalPages).toBe(1);

    component.paginatedData.set({
      items: [],
      page: 1,
      page_size: 20,
      total: 45,
    });
    expect(component.totalPages).toBe(3);
  });

  it('resets filters and reloads transactions', () => {
    component.filterQuery = 'test';
    component.filterType = 'income';
    component.resetFilter();

    expect(component.filterQuery).toBe('');
    expect(component.filterType).toBe('');
    expect(component.currentPage).toBe(1);
    expect(mockTxService.getTransactions).toHaveBeenCalled();
  });
});