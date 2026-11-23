import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { of } from 'rxjs';
import { BudgetListComponent } from './budget-list.component';
import { BudgetService } from '../../core/services/budget.service';
import { CategoryService } from '../../core/services/category.service';
import { Budget } from '../../core/models/budget.model';
import { Category } from '../../core/models/category.model';

describe('BudgetListComponent', () => {
  let component: BudgetListComponent;
  let fixture: ComponentFixture<BudgetListComponent>;
  let mockBudgetService: Partial<BudgetService>;
  let mockCatService: Partial<CategoryService>;

  const mockCategories: Category[] = [
    { id: 1, user_id: 1, name: 'Ăn uống', kind: 'expense', created_at: '2026-11-20T00:00:00Z' },
    { id: 2, user_id: 1, name: 'Di chuyển', kind: 'expense', created_at: '2026-11-20T00:00:00Z' },
  ];

  const mockBudgets: Budget[] = [
    {
      id: 1,
      user_id: 1,
      category_id: 1,
      category_name: 'Ăn uống',
      month: '2026-11',
      amount: 2000000,
      spent: 1000000,
      status: 'ok',
    },
    {
      id: 2,
      user_id: 1,
      category_id: 2,
      category_name: 'Di chuyển',
      month: '2026-11',
      amount: 1000000,
      spent: 850000,
      status: 'warning',
    },
  ];

  beforeEach(async () => {
    mockBudgetService = {
      getBudgets: vi.fn().mockReturnValue(of(mockBudgets)),
      upsertBudget: vi.fn().mockReturnValue(of(mockBudgets[0])),
      deleteBudget: vi.fn().mockReturnValue(of({ status: 'ok' })),
    };

    mockCatService = {
      categories: signal<Category[]>(mockCategories),
      getCategories: vi.fn().mockReturnValue(of(mockCategories)),
    };

    await TestBed.configureTestingModule({
      imports: [BudgetListComponent],
      providers: [
        { provide: BudgetService, useValue: mockBudgetService },
        { provide: CategoryService, useValue: mockCatService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(BudgetListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('creates component and loads budgets and categories', () => {
    expect(component).toBeTruthy();
    expect(mockBudgetService.getBudgets).toHaveBeenCalled();
    expect(mockCatService.getCategories).toHaveBeenCalledWith('expense');
    expect(component.budgets().length).toBe(2);
  });

  it('calculates totals and percentage correctly', () => {
    expect(component.totalBudget).toBe(3000000);
    expect(component.totalSpent).toBe(1850000);
    expect(component.calcPercent(1850000, 3000000)).toBe(62);
  });

  it('submits valid budget upsert', () => {
    component.budgetForm.setValue({
      category_id: 1,
      amount: 2500000,
    });

    component.onFormSubmit();
    expect(mockBudgetService.upsertBudget).toHaveBeenCalledWith({
      category_id: 1,
      month: component.selectedMonth,
      amount: 2500000,
    });
  });
});