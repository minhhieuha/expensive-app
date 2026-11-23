import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { CategoryService } from '../../core/services/category.service';
import { Category } from '../../core/models/category.model';

@Component({
  selector: 'app-category-list',
  standalone: true,
  imports: [CommonModule, FormsModule, ReactiveFormsModule],
  templateUrl: './category-list.component.html',
  styleUrl: './category-list.component.scss',
})
export class CategoryListComponent implements OnInit {
  private readonly categoryService = inject(CategoryService);
  private readonly fb = inject(FormBuilder);

  readonly activeTab = signal<'expense' | 'income'>('expense');
  readonly loading = signal<boolean>(false);
  readonly errorMsg = signal<string | null>(null);
  readonly successMsg = signal<string | null>(null);

  // Danh sách danh mục từ service
  readonly allCategories = this.categoryService.categories;

  // Form thêm mới
  addForm: FormGroup = this.fb.group({
    name: ['', [Validators.required, Validators.maxLength(50)]],
    kind: ['expense', Validators.required],
  });

  // State sửa danh mục
  editingCategory = signal<Category | null>(null);
  editName = '';

  // State modal xóa cần replace_with
  deletingCategory = signal<Category | null>(null);
  selectedReplaceId = signal<number | null>(null);

  ngOnInit(): void {
    this.loadCategories();
  }

  loadCategories(): void {
    this.loading.set(true);
    this.errorMsg.set(null);
    this.categoryService.getCategories().subscribe({
      next: () => {
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không thể tải danh sách danh mục');
      },
    });
  }

  get filteredCategories(): Category[] {
    return this.allCategories().filter((c) => c.kind === this.activeTab());
  }

  setTab(tab: 'expense' | 'income'): void {
    this.activeTab.set(tab);
    this.addForm.patchValue({ kind: tab });
    this.cancelEdit();
  }

  onAddSubmit(): void {
    if (this.addForm.invalid) return;

    this.loading.set(true);
    this.errorMsg.set(null);
    this.successMsg.set(null);

    const { name, kind } = this.addForm.value;
    this.categoryService.createCategory({ name: name.trim(), kind }).subscribe({
      next: () => {
        this.loading.set(false);
        this.successMsg.set('Thêm danh mục thành công!');
        this.addForm.patchValue({ name: '' });
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Lỗi khi thêm danh mục');
      },
    });
  }

  startEdit(cat: Category): void {
    this.editingCategory.set(cat);
    this.editName = cat.name;
  }

  cancelEdit(): void {
    this.editingCategory.set(null);
    this.editName = '';
  }

  saveEdit(): void {
    const cat = this.editingCategory();
    if (!cat || !this.editName.trim()) return;

    this.loading.set(true);
    this.errorMsg.set(null);

    this.categoryService.updateCategory(cat.id, { name: this.editName.trim() }).subscribe({
      next: () => {
        this.loading.set(false);
        this.cancelEdit();
        this.successMsg.set('Cập nhật danh mục thành công!');
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Lỗi khi cập nhật danh mục');
      },
    });
  }

  onDeleteClick(cat: Category): void {
    if (!confirm(`Bạn có chắc muốn xóa danh mục "${cat.name}"?`)) return;

    this.loading.set(true);
    this.errorMsg.set(null);

    this.categoryService.deleteCategory(cat.id).subscribe({
      next: () => {
        this.loading.set(false);
        this.successMsg.set('Đã xóa danh mục thành công!');
      },
      error: (err) => {
        this.loading.set(false);
        // Nếu lỗi do danh mục có giao dịch -> mở modal chọn replace_with
        if (err.status === 422 && err.error?.error?.message?.includes('thay thế')) {
          this.deletingCategory.set(cat);
          const others = this.otherCategoriesOfSameKind(cat);
          this.selectedReplaceId.set(others.length > 0 ? others[0].id : null);
        } else {
          this.errorMsg.set(err.error?.error?.message || 'Không thể xóa danh mục');
        }
      },
    });
  }

  otherCategoriesOfSameKind(cat: Category): Category[] {
    return this.allCategories().filter((c) => c.kind === cat.kind && c.id !== cat.id);
  }

  confirmReplaceAndDelete(): void {
    const cat = this.deletingCategory();
    const repId = this.selectedReplaceId();
    if (!cat || !repId) return;

    this.loading.set(true);
    this.errorMsg.set(null);

    this.categoryService.deleteCategory(cat.id, repId).subscribe({
      next: () => {
        this.loading.set(false);
        this.deletingCategory.set(null);
        this.successMsg.set('Đã chuyển giao dịch và xóa danh mục!');
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMsg.set(err.error?.error?.message || 'Không thể chuyển và xóa danh mục');
      },
    });
  }

  cancelReplace(): void {
    this.deletingCategory.set(null);
    this.selectedReplaceId.set(null);
  }
}