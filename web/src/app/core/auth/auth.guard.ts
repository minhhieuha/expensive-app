import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';
import { AuthService } from './auth.service';

export const authGuard: CanActivateFn = () => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (authService.currentUser() !== null) {
    return true;
  }

  return authService.getMe().pipe(
    map((user) => {
      if (user !== null) {
        return true;
      }
      return router.createUrlTree(['/login']);
    })
  );
};