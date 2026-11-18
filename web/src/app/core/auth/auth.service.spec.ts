import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { AuthService } from './auth.service';
import { User } from '../models/user.model';

describe('AuthService', () => {
  let service: AuthService;
  let httpTesting: HttpTestingController;

  const mockUser: User = {
    id: 1,
    email: 'test@example.com',
    timezone: 'Asia/Ho_Chi_Minh',
    created_at: '2026-11-18T00:00:00Z',
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(AuthService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created and have null currentUser initially', () => {
    expect(service).toBeTruthy();
    expect(service.currentUser()).toBeNull();
  });

  it('should login and set currentUser', () => {
    service.login({ email: 'test@example.com', password: 'password123' }).subscribe((user) => {
      expect(user).toEqual(mockUser);
    });

    const req = httpTesting.expectOne('/api/auth/login');
    expect(req.request.method).toBe('POST');
    req.flush(mockUser);

    expect(service.currentUser()).toEqual(mockUser);
  });

  it('should register and set currentUser', () => {
    service.register({ email: 'test@example.com', password: 'password123' }).subscribe((user) => {
      expect(user).toEqual(mockUser);
    });

    const req = httpTesting.expectOne('/api/auth/register');
    expect(req.request.method).toBe('POST');
    req.flush(mockUser);

    expect(service.currentUser()).toEqual(mockUser);
  });

  it('should logout and clear currentUser', () => {
    service.currentUser.set(mockUser);

    service.logout().subscribe((res) => {
      expect(res.status).toBe('ok');
    });

    const req = httpTesting.expectOne('/api/auth/logout');
    expect(req.request.method).toBe('POST');
    req.flush({ status: 'ok' });

    expect(service.currentUser()).toBeNull();
  });

  it('should getMe and set currentUser', () => {
    service.getMe().subscribe((user) => {
      expect(user).toEqual(mockUser);
    });

    const req = httpTesting.expectOne('/api/me');
    expect(req.request.method).toBe('GET');
    req.flush(mockUser);

    expect(service.currentUser()).toEqual(mockUser);
  });
});