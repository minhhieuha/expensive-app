import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { BudgetService } from '../../core/services/budget.service';
import { CategoryService } from '../../core/services/category.service';
import { Budget } from '../../core/models/budget.model';
import { Category } from '../../core/models/category.model';
import { MoneyPipe } from '../../shared/pipes/money.pipe';

@Component({
  selector: 'app-budget-list',
  standalone: true,
  imports: [CommonModule, FormsModule, ReactiveFormsModule, MoneyPipe],
  templateUrl: './budget-list.component.html',
  styleUrl: './budget-list.component.scss',
})
export class BudgetListComponent implements OnInit {
  private readonly budgetService = inject(BudgetService);
  private readonly categoryService = inject(CategoryService);
  private readonly fb = inject(FormBuilder);

  selectedMonth = this.currentMonthStr();
  readonly budgets = signal<Budget[]>([]);
  readonly expenseCategories = signal<Category[]>([]);
  readonly loading = signal<boolean>(false);
  readonly errorMsg = signal<string | null>(null);
  readonly successMsg = signal<string | null>(null);

  // Form đặt/sửa ngân sách
  budgetForm: FormGroup = this.fb.group({
    category_id: [null, [Validators.required]],
    amount: [null, [Validators.required, Validators.min(1), Validators.max(1000000000000)]],
  });

  currentMonthStr(): string {
    const d = new Date();
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    return `${year}-${month}`;
  }

  ngOnInit(): void {
    this.loadExpenseCategories();
    this.loadBudgets();
  }

  loadExpenseCategories(): void {
    this.categoryService.getCategories('expense').subscribe({
      next: (cats) => {
        this.expenseCategories.set(cats);
        if (cats.length > 0 && !this.budgetForm.get('category_id')?.value) {
          this.budgetForm.patchValue({ category_id: cats[0].id });
        }
      },
    });
  }

  loadBudgets(): void {
    this.loading.set(true);
    this.errorMsg.set(null);

    this.budgetService.getBudgets(this.selectedMonth).subscribe({
      next: (data) => {
        this.loading.set(false);
        this.budgets.set(data);
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không thể tải danh sách ngân sách');
      },
    });
  }

  onMonthChange(): void {
    this.successMsg.set(null);
    this.loadBudgets();
  }

  get totalBudget(): number {
    return this.budgets().reduce((acc, b) => acc + b.amount, 0);
  }

  get totalSpent(): number {
    return this.budgets().reduce((acc, b) => acc + b.spent, 0);
  }

  calcPercent(spent: number, amount: number): number {
    if (amount <= 0) return 0;
    return Math.round((spent * 100) / amount);
  }

  calcProgressWidth(spent: number, amount: number): number {
    return Math.min(100, this.calcPercent(spent, amount));
  }

  editBudget(b: Budget): void {
    this.budgetForm.patchValue({
      category_id: b.category_id,
      amount: b.amount,
    });
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  onFormSubmit(): void {
    if (this.budgetForm.invalid) {
      this.budgetForm.markAllAsTouched();
      return;
    }

    this.loading.set(true);
    this.errorMsg.set(null);
    this.successMsg.set(null);

    const val = this.budgetForm.value;
    const amountInt = Math.round(Number(val.amount));

    this.budgetService
      .upsertBudget({
        category_id: Number(val.category_id),
        month: this.selectedMonth,
        amount: amountInt,
      })
      .subscribe({
        next: () => {
          this.loading.set(false);
          this.successMsg.set('Lưu ngân sách thành công!');
          this.loadBudgets();
        },
        error: (err) => {
          this.loading.set(false);
          this.errorMsg.set(err.error?.error?.message || 'Lỗi khi lưu ngân sách');
        },
      });
  }

  deleteBudget(b: Budget): void {
    if (!confirm(`Bạn có chắc muốn xóa ngân sách danh mục "${b.category_name}" tháng ${b.month}?`)) return;

    this.loading.set(true);
    this.errorMsg.set(null);

    this.budgetService.deleteBudget(b.id).subscribe({
      next: () => {
        this.loading.set(false);
        this.successMsg.set('Đã xóa ngân sách!');
        this.loadBudgets();
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không thể xóa ngân sách');
      },
    });
  }
}