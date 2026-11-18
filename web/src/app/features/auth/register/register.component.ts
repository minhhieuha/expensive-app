import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators, AbstractControl, ValidationErrors } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { AuthService } from '../../../core/auth/auth.service';

function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password')?.value;
  const confirm = control.get('confirmPassword')?.value;
  if (password && confirm && password !== confirm) {
    return { passwordMismatch: true };
  }
  return null;
}

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  template: `
    <div class="auth-card">
      <h2>Đăng ký tài khoản</h2>

      @if (errorMessage()) {
        <div class="error-alert" role="alert">
          {{ errorMessage() }}
        </div>
      }

      <form [formGroup]="form" (ngSubmit)="onSubmit()">
        <div class="form-group">
          <label for="reg-email">Email</label>
          <input
            id="reg-email"
            type="email"
            formControlName="email"
            placeholder="name@example.com"
          />
          @if (form.controls.email.touched && form.controls.email.invalid) {
            <span class="field-error">Email không hợp lệ</span>
          }
        </div>

        <div class="form-group">
          <label for="reg-password">Mật khẩu</label>
          <input
            id="reg-password"
            type="password"
            formControlName="password"
            placeholder="Tối thiểu 8 ký tự"
          />
          @if (form.controls.password.touched && form.controls.password.invalid) {
            <span class="field-error">Mật khẩu phải từ 8 đến 72 ký tự</span>
          }
        </div>

        <div class="form-group">
          <label for="reg-confirm">Xác nhận mật khẩu</label>
          <input
            id="reg-confirm"
            type="password"
            formControlName="confirmPassword"
            placeholder="Nhập lại mật khẩu"
          />
          @if (form.controls.confirmPassword.touched && form.hasError('passwordMismatch')) {
            <span class="field-error">Mật khẩu xác nhận không khớp</span>
          }
        </div>

        <button type="submit" [disabled]="form.invalid || loading()">
          {{ loading() ? 'Đang tạo tài khoản...' : 'Đăng ký' }}
        </button>
      </form>

      <p class="switch-link">
        Đã có tài khoản? <a routerLink="/login">Đăng nhập</a>
      </p>
    </div>
  `,
  styles: [`
    .auth-card {
      max-width: 400px;
      margin: 3rem auto;
      padding: 2rem;
      border: 1px solid #e0e0e0;
      border-radius: 8px;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
    }
    h2 { margin-top: 0; text-align: center; }
    .error-alert {
      background-color: #ffebee;
      color: #c62828;
      padding: 0.75rem;
      border-radius: 4px;
      margin-bottom: 1rem;
      font-size: 0.9rem;
    }
    .form-group {
      margin-bottom: 1.25rem;
      display: flex;
      flex-direction: column;
    }
    label { font-weight: 500; margin-bottom: 0.25rem; }
    input {
      padding: 0.5rem;
      border: 1px solid #ccc;
      border-radius: 4px;
      font-size: 1rem;
    }
    .field-error {
      color: #c62828;
      font-size: 0.8rem;
      margin-top: 0.25rem;
    }
    button {
      width: 100%;
      padding: 0.75rem;
      background-color: #2e7d32;
      color: #fff;
      border: none;
      border-radius: 4px;
      font-size: 1rem;
      cursor: pointer;
    }
    button:disabled { background-color: #a5d6a7; cursor: not-allowed; }
    .switch-link { text-align: center; margin-top: 1rem; font-size: 0.9rem; }
  `],
})
export class RegisterComponent {
  private readonly fb = inject(FormBuilder);
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  readonly form = this.fb.nonNullable.group(
    {
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(8), Validators.maxLength(72)]],
      confirmPassword: ['', [Validators.required]],
    },
    { validators: [passwordMatchValidator] }
  );

  readonly loading = signal(false);
  readonly errorMessage = signal('');

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.loading.set(true);
    this.errorMessage.set('');

    const { email, password } = this.form.getRawValue();
    this.authService.register({ email, password }).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/']);
      },
      error: (err: unknown) => {
        this.loading.set(false);
        if (err instanceof HttpErrorResponse) {
          if (err.status === 409) {
            this.errorMessage.set('Email này đã được sử dụng.');
          } else if (err.status === 422) {
            this.errorMessage.set('Dữ liệu không hợp lệ. Vui lòng kiểm tra lại.');
          } else {
            this.errorMessage.set('Đã có lỗi xảy ra. Vui lòng thử lại.');
          }
        } else {
          this.errorMessage.set('Lỗi kết nối máy chủ.');
        }
      },
    });
  }
}