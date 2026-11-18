import { HttpInterceptorFn } from '@angular/common/http';

export const csrfInterceptor: HttpInterceptorFn = (req, next) => {
  const method = req.method.toUpperCase();
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    const cloned = req.clone({
      setHeaders: {
        'X-Requested-With': 'expense-app',
      },
    });
    return next(cloned);
  }
  return next(req);
};