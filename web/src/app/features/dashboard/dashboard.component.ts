import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [RouterLink],
  template: `
    <div class="dashboard-container">
      <h1>Bảng điều khiển</h1>
      <p>Xin chào, <strong>{{ authService.currentUser()?.email }}</strong>!</p>
      <div class="quick-links">
        <a routerLink="/transactions" class="btn btn-primary">📖 Xem sổ giao dịch</a>
        <a routerLink="/transactions/new" class="btn btn-success">+ Ghi chép mới</a>
        <a routerLink="/categories" class="btn btn-secondary">🏷️ Quản lý danh mục</a>
      </div>
    </div>
  `,
  styles: [`
    .dashboard-container {
      padding: 2rem 1rem;
      max-width: 800px;
      margin: 0 auto;
    }
    .quick-links {
      display: flex;
      gap: 1rem;
      margin-top: 1.5rem;
      flex-wrap: wrap;
    }
    .btn {
      padding: 0.65rem 1.25rem;
      font-size: 0.95rem;
      font-weight: 600;
      border-radius: 6px;
      text-decoration: none;
      display: inline-block;
      transition: background-color 0.2s;
    }
    .btn-primary {
      background: #2563eb;
      color: #fff;
    }
    .btn-primary:hover {
      background: #1d4ed8;
    }
    .btn-success {
      background: #16a34a;
      color: #fff;
    }
    .btn-success:hover {
      background: #15803d;
    }
    .btn-secondary {
      background: #f1f5f9;
      color: #334155;
      border: 1px solid #cbd5e1;
    }
    .btn-secondary:hover {
      background: #e2e8f0;
    }
  `],
})
export class DashboardComponent {
  readonly authService = inject(AuthService);
}
