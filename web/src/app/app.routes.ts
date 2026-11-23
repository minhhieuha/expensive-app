import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import { guestGuard } from './core/auth/guest.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('./features/auth/login/login.component').then((m) => m.LoginComponent),
    canActivate: [guestGuard],
  },
  {
    path: 'register',
    loadComponent: () =>
      import('./features/auth/register/register.component').then((m) => m.RegisterComponent),
    canActivate: [guestGuard],
  },
  {
    path: '',
    loadComponent: () =>
      import('./features/dashboard/dashboard.component').then((m) => m.DashboardComponent),
    canActivate: [authGuard],
  },
  {
    path: 'transactions',
    loadComponent: () =>
      import('./features/transactions/transaction-list.component').then(
        (m) => m.TransactionListComponent
      ),
    canActivate: [authGuard],
  },
  {
    path: 'transactions/new',
    loadComponent: () =>
      import('./features/transactions/transaction-form.component').then(
        (m) => m.TransactionFormComponent
      ),
    canActivate: [authGuard],
  },
  {
    path: 'transactions/:id/edit',
    loadComponent: () =>
      import('./features/transactions/transaction-form.component').then(
        (m) => m.TransactionFormComponent
      ),
    canActivate: [authGuard],
  },
  {
    path: 'categories',
    loadComponent: () =>
      import('./features/categories/category-list.component').then(
        (m) => m.CategoryListComponent
      ),
    canActivate: [authGuard],
  },
  {
    path: 'budgets',
    loadComponent: () =>
      import('./features/budgets/budget-list.component').then(
        (m) => m.BudgetListComponent
      ),
    canActivate: [authGuard],
  },
  {
    path: '**',
    redirectTo: '',
  },
];