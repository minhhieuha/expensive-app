import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { of } from 'rxjs';
import { CategoryListComponent } from './category-list.component';
import { CategoryService } from '../../core/services/category.service';
import { Category } from '../../core/models/category.model';

describe('CategoryListComponent', () => {
  let component: CategoryListComponent;
  let fixture: ComponentFixture<CategoryListComponent>;
  let mockCategoryService: Partial<CategoryService>;

  const mockCategories: Category[] = [
    { id: 1, user_id: 1, name: 'Ăn uống', kind: 'expense', created_at: '2026-11-20T00:00:00Z' },
    { id: 2, user_id: 1, name: 'Lương', kind: 'income', created_at: '2026-11-20T00:00:00Z' },
  ];

  beforeEach(async () => {
    mockCategoryService = {
      categories: signal<Category[]>(mockCategories),
      getCategories: vi.fn().mockReturnValue(of(mockCategories)),
      createCategory: vi.fn().mockReturnValue(of({
        id: 3,
        user_id: 1,
        name: 'Tiền thưởng',
        kind: 'income',
        created_at: '2026-11-20T00:00:00Z',
      })),
      updateCategory: vi.fn().mockReturnValue(of({
        id: 1,
        user_id: 1,
        name: 'Ăn uống ngoài',
        kind: 'expense',
        created_at: '2026-11-20T00:00:00Z',
      })),
      deleteCategory: vi.fn().mockReturnValue(of({ status: 'ok' })),
    };

    await TestBed.configureTestingModule({
      imports: [CategoryListComponent],
      providers: [
        { provide: CategoryService, useValue: mockCategoryService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CategoryListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('creates component and loads categories', () => {
    expect(component).toBeTruthy();
    expect(mockCategoryService.getCategories).toHaveBeenCalled();
  });

  it('filters categories by active tab', () => {
    component.setTab('expense');
    expect(component.filteredCategories.every((c) => c.kind === 'expense')).toBe(true);

    component.setTab('income');
    expect(component.filteredCategories.every((c) => c.kind === 'income')).toBe(true);
  });

  it('validates add category form', () => {
    component.addForm.setValue({ name: '', kind: 'expense' });
    expect(component.addForm.invalid).toBe(true);

    component.addForm.setValue({ name: 'Mua sắm', kind: 'expense' });
    expect(component.addForm.valid).toBe(true);

    component.onAddSubmit();
    expect(mockCategoryService.createCategory).toHaveBeenCalledWith({
      name: 'Mua sắm',
      kind: 'expense',
    });
  });
});