import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, ActivatedRoute } from '@angular/router';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { of } from 'rxjs';
import { TransactionFormComponent } from './transaction-form.component';
import { TransactionService } from '../../core/services/transaction.service';
import { CategoryService } from '../../core/services/category.service';
import { Category } from '../../core/models/category.model';
import { Transaction } from '../../core/models/transaction.model';

describe('TransactionFormComponent', () => {
  let component: TransactionFormComponent;
  let fixture: ComponentFixture<TransactionFormComponent>;
  let mockTxService: Partial<TransactionService>;
  let mockCatService: Partial<CategoryService>;

  const mockCategories: Category[] = [
    { id: 1, user_id: 1, name: 'Ăn uống', kind: 'expense', created_at: '2026-11-20T00:00:00Z' },
    { id: 2, user_id: 1, name: 'Lương', kind: 'income', created_at: '2026-11-20T00:00:00Z' },
  ];

  const mockTx: Transaction = {
    id: 10,
    user_id: 1,
    category_id: 1,
    category_name: 'Ăn uống',
    type: 'expense',
    amount: 150000,
    occurred_on: '2026-11-20',
    note: 'Ăn trưa',
    created_at: '2026-11-20T12:00:00Z',
    updated_at: '2026-11-20T12:00:00Z',
  };

  beforeEach(async () => {
    mockTxService = {
      getTransaction: vi.fn().mockReturnValue(of(mockTx)),
      createTransaction: vi.fn().mockReturnValue(of(mockTx)),
      updateTransaction: vi.fn().mockReturnValue(of(mockTx)),
    };

    mockCatService = {
      getCategories: vi.fn().mockReturnValue(of(mockCategories)),
    };

    await TestBed.configureTestingModule({
      imports: [TransactionFormComponent],
      providers: [
        provideRouter([
          { path: 'transactions', component: TransactionFormComponent },
        ]),
        { provide: TransactionService, useValue: mockTxService },
        { provide: CategoryService, useValue: mockCatService },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'id' ? null : null),
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TransactionFormComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('initializes form in create mode with default expense and today date', () => {
    expect(component).toBeTruthy();
    expect(component.isEditMode()).toBe(false);
    expect(component.txForm.get('type')?.value).toBe('expense');
    expect(component.txForm.get('occurred_on')?.value).toBe(component.todayStr());
  });

  it('validates amount boundaries (min > 0, max <= 10^12)', () => {
    const amountCtrl = component.txForm.get('amount');

    amountCtrl?.setValue(0);
    expect(amountCtrl?.valid).toBe(false);

    amountCtrl?.setValue(-500);
    expect(amountCtrl?.valid).toBe(false);

    amountCtrl?.setValue(1000000000001);
    expect(amountCtrl?.valid).toBe(false);

    amountCtrl?.setValue(50000);
    expect(amountCtrl?.valid).toBe(true);
  });

  it('switches type and adjusts category filter', () => {
    component.onTypeChange('income');
    expect(component.txForm.get('type')?.value).toBe('income');
    expect(component.filteredCategories.every((c) => c.kind === 'income')).toBe(true);
  });

  it('submits valid new transaction', () => {
    component.txForm.setValue({
      type: 'expense',
      category_id: 1,
      amount: 80000,
      occurred_on: '2026-11-20',
      note: 'Trưa nay ăn phở',
    });

    component.onSubmit();
    expect(mockTxService.createTransaction).toHaveBeenCalledWith({
      type: 'expense',
      category_id: 1,
      amount: 80000,
      occurred_on: '2026-11-20',
      note: 'Trưa nay ăn phở',
    });
  });
});