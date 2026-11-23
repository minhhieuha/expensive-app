import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { Category, CreateCategoryRequest, UpdateCategoryRequest } from '../models/category.model';

@Injectable({
  providedIn: 'root',
})
export class CategoryService {
  private readonly http = inject(HttpClient);
  readonly categories = signal<Category[]>([]);

  getCategories(kind?: 'expense' | 'income'): Observable<Category[]> {
    let params = new HttpParams();
    if (kind) {
      params = params.set('kind', kind);
    }
    return this.http.get<Category[]>('/api/categories', { params }).pipe(
      tap((items) => {
        if (!kind) {
          this.categories.set(items);
        }
      })
    );
  }

  createCategory(req: CreateCategoryRequest): Observable<Category> {
    return this.http.post<Category>('/api/categories', req).pipe(
      tap((created) => {
        this.categories.update((prev) => [...prev, created]);
      })
    );
  }

  updateCategory(id: number, req: UpdateCategoryRequest): Observable<Category> {
    return this.http.patch<Category>(`/api/categories/${id}`, req).pipe(
      tap((updated) => {
        this.categories.update((prev) =>
          prev.map((c) => (c.id === id ? updated : c))
        );
      })
    );
  }

  deleteCategory(id: number, replaceWithId?: number): Observable<{ status: string }> {
    let params = new HttpParams();
    if (replaceWithId) {
      params = params.set('replace_with', replaceWithId.toString());
    }
    return this.http.delete<{ status: string }>(`/api/categories/${id}`, { params }).pipe(
      tap(() => {
        this.categories.update((prev) => prev.filter((c) => c.id !== id));
      })
    );
  }
}