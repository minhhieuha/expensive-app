import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree } from '@angular/router';
import { of, firstValueFrom } from 'rxjs';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { authGuard } from './auth.guard';
import { AuthService } from './auth.service';
import { User } from '../models/user.model';

describe('authGuard', () => {
  let authServiceMock: { currentUser: ReturnType<typeof vi.fn>; getMe: ReturnType<typeof vi.fn> };
  let routerMock: { createUrlTree: ReturnType<typeof vi.fn> };
  const dummyRoute = {} as ActivatedRouteSnapshot;
  const dummyState = {} as RouterStateSnapshot;

  const mockUser: User = {
    id: 1,
    email: 'test@example.com',
    timezone: 'Asia/Ho_Chi_Minh',
    created_at: '2026-11-18T00:00:00Z',
  };

  beforeEach(() => {
    authServiceMock = {
      currentUser: vi.fn().mockReturnValue(null),
      getMe: vi.fn(),
    };
    routerMock = {
      createUrlTree: vi.fn(),
    };

    TestBed.configureTestingModule({
      providers: [
        { provide: AuthService, useValue: authServiceMock },
        { provide: Router, useValue: routerMock },
      ],
    });
  });

  it('should allow activation when currentUser signal is set', () => {
    authServiceMock.currentUser.mockReturnValue(mockUser);

    const result = TestBed.runInInjectionContext(() => authGuard(dummyRoute, dummyState));
    expect(result).toBe(true);
  });

  it('should allow activation when getMe returns user', async () => {
    authServiceMock.currentUser.mockReturnValue(null);
    authServiceMock.getMe.mockReturnValue(of(mockUser));

    const result = TestBed.runInInjectionContext(() => authGuard(dummyRoute, dummyState));
    if (typeof result === 'object' && result !== null && 'subscribe' in result) {
      const val = await firstValueFrom(result);
      expect(val).toBe(true);
    } else {
      expect(result).toBe(true);
    }
  });

  it('should redirect to login when getMe returns null', async () => {
    authServiceMock.currentUser.mockReturnValue(null);
    authServiceMock.getMe.mockReturnValue(of(null));
    const mockTree = {} as UrlTree;
    routerMock.createUrlTree.mockReturnValue(mockTree);

    const result = TestBed.runInInjectionContext(() => authGuard(dummyRoute, dummyState));
    if (typeof result === 'object' && result !== null && 'subscribe' in result) {
      const val = await firstValueFrom(result);
      expect(val).toBe(mockTree);
      expect(routerMock.createUrlTree).toHaveBeenCalledWith(['/login']);
    } else {
      expect(result).toBe(mockTree);
    }
  });
});