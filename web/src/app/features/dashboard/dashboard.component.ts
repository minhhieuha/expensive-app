import { Component, inject } from '@angular/core';
import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  template: `
    <div class="dashboard-container">
      <h1>Bảng điều khiển</h1>
      <p>Xin chào, <strong>{{ authService.currentUser()?.email }}</strong>!</p>
    </div>
  `,
  styles: [`
    .dashboard-container {
      padding: 2rem;
      max-width: 800px;
      margin: 0 auto;
    }
  `],
})
export class DashboardComponent {
  readonly authService = inject(AuthService);
}