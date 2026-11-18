import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, catchError, of } from 'rxjs';
import { User } from '../models/user.model';

export interface Credentials {
  email: string;
  password: string;
}

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly http = inject(HttpClient);
  readonly currentUser = signal<User | null>(null);

  register(creds: Credentials): Observable<User> {
    return this.http.post<User>('/api/auth/register', creds).pipe(
      tap((user) => this.currentUser.set(user))
    );
  }

  login(creds: Credentials): Observable<User> {
    return this.http.post<User>('/api/auth/login', creds).pipe(
      tap((user) => this.currentUser.set(user))
    );
  }

  logout(): Observable<{ status: string }> {
    return this.http.post<{ status: string }>('/api/auth/logout', {}).pipe(
      tap(() => this.currentUser.set(null)),
      catchError(() => {
        this.currentUser.set(null);
        return of({ status: 'ok' });
      })
    );
  }

  getMe(): Observable<User | null> {
    return this.http.get<User>('/api/me').pipe(
      tap((user) => this.currentUser.set(user)),
      catchError(() => {
        this.currentUser.set(null);
        return of(null);
      })
    );
  }

  clearSession(): void {
    this.currentUser.set(null);
  }
}