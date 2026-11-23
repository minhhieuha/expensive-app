import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, ActivatedRoute } from '@angular/router';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { TransactionService } from '../../core/services/transaction.service';
import { CategoryService } from '../../core/services/category.service';
import { Category } from '../../core/models/category.model';

@Component({
  selector: 'app-transaction-form',
  standalone: true,
  imports: [CommonModule, RouterModule, ReactiveFormsModule],
  templateUrl: './transaction-form.component.html',
  styleUrl: './transaction-form.component.scss',
})
export class TransactionFormComponent implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly txService = inject(TransactionService);
  private readonly catService = inject(CategoryService);

  readonly isEditMode = signal<boolean>(false);
  readonly editId = signal<number | null>(null);
  readonly loading = signal<boolean>(false);
  readonly submitting = signal<boolean>(false);
  readonly errorMsg = signal<string | null>(null);

  readonly allCategories = signal<Category[]>([]);

  txForm: FormGroup = this.fb.group({
    type: ['expense', Validators.required],
    category_id: [null, [Validators.required]],
    amount: [null, [Validators.required, Validators.min(1), Validators.max(1000000000000)]],
    occurred_on: [this.todayStr(), [Validators.required, Validators.pattern(/^\d{4}-\d{2}-\d{2}$/)]],
    note: ['', [Validators.maxLength(500)]],
  });

  todayStr(): string {
    const d = new Date();
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  ngOnInit(): void {
    this.loadCategoriesAndCheckMode();
  }

  loadCategoriesAndCheckMode(): void {
    this.loading.set(true);
    this.catService.getCategories().subscribe({
      next: (cats) => {
        this.allCategories.set(cats);
        this.loading.set(false);
        this.checkRouteParam();
      },
      error: () => {
        this.loading.set(false);
        this.errorMsg.set('Không thể tải danh sách danh mục');
      },
    });
  }

  checkRouteParam(): void {
    const idParam = this.route.snapshot.paramMap.get('id');
    if (idParam) {
      const id = Number(idParam);
      if (id > 0) {
        this.isEditMode.set(true);
        this.editId.set(id);
        this.loadExistingTransaction(id);
      }
    } else {
      // Create mode: pre-select first matching category
      this.ensureCategoryMatch();
    }
  }

  loadExistingTransaction(id: number): void {
    this.loading.set(true);
    this.txService.getTransaction(id).subscribe({
      next: (tx) => {
        this.loading.set(false);
        this.txForm.patchValue({
          type: tx.type,
          category_id: tx.category_id,
          amount: tx.amount,
          occurred_on: tx.occurred_on,
          note: tx.note || '',
        });
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không tìm thấy giao dịch');
      },
    });
  }

  get filteredCategories(): Category[] {
    const currentType = this.txForm.get('type')?.value;
    return this.allCategories().filter((c) => c.kind === currentType);
  }

  onTypeChange(newType: 'expense' | 'income'): void {
    this.txForm.patchValue({ type: newType });
    this.ensureCategoryMatch();
  }

  private ensureCategoryMatch(): void {
    const matching = this.filteredCategories;
    const currentCatId = this.txForm.get('category_id')?.value;
    const exists = matching.some((c) => c.id === currentCatId);
    if (!exists) {
      this.txForm.patchValue({
        category_id: matching.length > 0 ? matching[0].id : null,
      });
    }
  }

  onSubmit(): void {
    if (this.txForm.invalid) {
      this.txForm.markAllAsTouched();
      return;
    }

    this.submitting.set(true);
    this.errorMsg.set(null);

    const val = this.txForm.value;
    const amountInt = Math.round(Number(val.amount));

    if (this.isEditMode() && this.editId()) {
      this.txService
        .updateTransaction(this.editId()!, {
          category_id: Number(val.category_id),
          type: val.type,
          amount: amountInt,
          occurred_on: val.occurred_on,
          note: val.note ? val.note.trim() : '',
        })
        .subscribe({
          next: () => {
            this.submitting.set(false);
            this.router.navigate(['/transactions']);
          },
          error: (err) => {
            this.submitting.set(false);
            this.errorMsg.set(err.error?.error?.message || 'Lỗi khi cập nhật giao dịch');
          },
        });
    } else {
      this.txService
        .createTransaction({
          category_id: Number(val.category_id),
          type: val.type,
          amount: amountInt,
          occurred_on: val.occurred_on,
          note: val.note ? val.note.trim() : '',
        })
        .subscribe({
          next: () => {
            this.submitting.set(false);
            this.router.navigate(['/transactions']);
          },
          error: (err) => {
            this.submitting.set(false);
            this.errorMsg.set(err.error?.error?.message || 'Lỗi khi tạo giao dịch');
          },
        });
    }
  }
}