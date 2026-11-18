import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { AuthService } from '../auth/auth.service';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);
  const authService = inject(AuthService);

  return next(req).pipe(
    catchError((err: unknown) => {
      if (err instanceof HttpErrorResponse && err.status === 401) {
        const isAuthRequest = req.url.includes('/api/auth/login') ||
          req.url.includes('/api/auth/register') ||
          req.url.includes('/api/me');

        if (!isAuthRequest) {
          authService.clearSession();
          router.navigate(['/login']);
        }
      }
      return throwError(() => err);
    })
  );
};