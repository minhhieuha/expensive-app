import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { TransactionService } from '../../core/services/transaction.service';
import { CategoryService } from '../../core/services/category.service';
import { Transaction, TransactionFilter, PaginatedTransactions } from '../../core/models/transaction.model';
import { Category } from '../../core/models/category.model';
import { MoneyPipe } from '../../shared/pipes/money.pipe';

@Component({
  selector: 'app-transaction-list',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule, MoneyPipe],
  templateUrl: './transaction-list.component.html',
  styleUrl: './transaction-list.component.scss',
})
export class TransactionListComponent implements OnInit {
  private readonly txService = inject(TransactionService);
  private readonly catService = inject(CategoryService);

  readonly loading = signal<boolean>(false);
  readonly errorMsg = signal<string | null>(null);
  readonly successMsg = signal<string | null>(null);

  // Dữ liệu phân trang
  readonly paginatedData = signal<PaginatedTransactions>({
    items: [],
    page: 1,
    page_size: 20,
    total: 0,
  });

  // Danh mục để lọc
  readonly categories = signal<Category[]>([]);

  // Bộ lọc hiện tại
  filterFrom = '';
  filterTo = '';
  filterType: '' | 'expense' | 'income' = '';
  filterCategoryId: number | '' = '';
  filterQuery = '';
  currentPage = 1;
  pageSize = 20;

  ngOnInit(): void {
    this.loadCategories();
    this.loadTransactions();
  }

  loadCategories(): void {
    this.catService.getCategories().subscribe({
      next: (cats) => this.categories.set(cats),
    });
  }

  get filteredCategories(): Category[] {
    if (!this.filterType) return this.categories();
    return this.categories().filter((c) => c.kind === this.filterType);
  }

  onTypeFilterChange(): void {
    // Reset category nếu category hiện tại không khớp type mới
    if (this.filterCategoryId) {
      const cat = this.categories().find((c) => c.id === this.filterCategoryId);
      if (cat && this.filterType && cat.kind !== this.filterType) {
        this.filterCategoryId = '';
      }
    }
    this.applyFilter();
  }

  applyFilter(): void {
    this.currentPage = 1;
    this.loadTransactions();
  }

  resetFilter(): void {
    this.filterFrom = '';
    this.filterTo = '';
    this.filterType = '';
    this.filterCategoryId = '';
    this.filterQuery = '';
    this.currentPage = 1;
    this.loadTransactions();
  }

  loadTransactions(): void {
    this.loading.set(true);
    this.errorMsg.set(null);

    const f: TransactionFilter = {
      page: this.currentPage,
      page_size: this.pageSize,
    };
    if (this.filterFrom) f.from = this.filterFrom;
    if (this.filterTo) f.to = this.filterTo;
    if (this.filterType) f.type = this.filterType;
    if (this.filterCategoryId) f.category_id = Number(this.filterCategoryId);
    if (this.filterQuery.trim()) f.q = this.filterQuery.trim();

    this.txService.getTransactions(f).subscribe({
      next: (res) => {
        this.loading.set(false);
        this.paginatedData.set(res);
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không thể tải danh sách giao dịch');
      },
    });
  }

  get totalPages(): number {
    const total = this.paginatedData().total;
    return Math.max(1, Math.ceil(total / this.pageSize));
  }

  goToPage(p: number): void {
    if (p < 1 || p > this.totalPages || p === this.currentPage) return;
    this.currentPage = p;
    this.loadTransactions();
  }

  deleteTransaction(tx: Transaction): void {
    if (!confirm(`Bạn có chắc muốn xóa giao dịch ngày ${tx.occurred_on}?`)) return;

    this.loading.set(true);
    this.errorMsg.set(null);
    this.txService.deleteTransaction(tx.id).subscribe({
      next: () => {
        this.loading.set(false);
        this.successMsg.set('Đã xóa giao dịch thành công!');
        this.loadTransactions();
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không thể xóa giao dịch');
      },
    });
  }
}