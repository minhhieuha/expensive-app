import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { CategoryService } from './category.service';
import { Category } from '../models/category.model';

describe('CategoryService', () => {
  let service: CategoryService;
  let httpTesting: HttpTestingController;

  const mockCategories: Category[] = [
    { id: 1, user_id: 1, name: 'Ăn uống', kind: 'expense', created_at: '2026-11-20T00:00:00Z' },
    { id: 2, user_id: 1, name: 'Lương', kind: 'income', created_at: '2026-11-20T00:00:00Z' },
  ];

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(CategoryService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('fetches categories and populates signal', () => {
    service.getCategories().subscribe((res) => {
      expect(res).toEqual(mockCategories);
    });

    const req = httpTesting.expectOne('/api/categories');
    expect(req.request.method).toBe('GET');
    req.flush(mockCategories);

    expect(service.categories()).toEqual(mockCategories);
  });

  it('fetches categories filtered by kind', () => {
    service.getCategories('expense').subscribe((res) => {
      expect(res.length).toBe(1);
    });

    const req = httpTesting.expectOne('/api/categories?kind=expense');
    expect(req.request.method).toBe('GET');
    req.flush([mockCategories[0]]);
  });

  it('creates category and appends to signal', () => {
    const newCat: Category = { id: 3, user_id: 1, name: 'Mua sắm', kind: 'expense', created_at: '2026-11-20T00:00:00Z' };
    service.categories.set([...mockCategories]);

    service.createCategory({ name: 'Mua sắm', kind: 'expense' }).subscribe((res) => {
      expect(res).toEqual(newCat);
    });

    const req = httpTesting.expectOne('/api/categories');
    expect(req.request.method).toBe('POST');
    req.flush(newCat);

    expect(service.categories()).toContainEqual(newCat);
  });

  it('updates category and updates signal', () => {
    service.categories.set([...mockCategories]);
    const updated: Category = { ...mockCategories[0], name: 'Ăn uống ngoài' };

    service.updateCategory(1, { name: 'Ăn uống ngoài' }).subscribe((res) => {
      expect(res.name).toBe('Ăn uống ngoài');
    });

    const req = httpTesting.expectOne('/api/categories/1');
    expect(req.request.method).toBe('PATCH');
    req.flush(updated);

    expect(service.categories()[0].name).toBe('Ăn uống ngoài');
  });

  it('deletes category with replace_with query param', () => {
    service.categories.set([...mockCategories]);

    service.deleteCategory(1, 2).subscribe((res) => {
      expect(res.status).toBe('ok');
    });

    const req = httpTesting.expectOne('/api/categories/1?replace_with=2');
    expect(req.request.method).toBe('DELETE');
    req.flush({ status: 'ok' });

    expect(service.categories().find((c) => c.id === 1)).toBeUndefined();
  });
});