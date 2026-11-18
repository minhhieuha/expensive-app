import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';
import { AuthService } from './auth.service';

export const guestGuard: CanActivateFn = () => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (authService.currentUser() !== null) {
    return router.createUrlTree(['/']);
  }

  return authService.getMe().pipe(
    map((user) => {
      if (user !== null) {
        return router.createUrlTree(['/']);
      }
      return true;
    })
  );
};