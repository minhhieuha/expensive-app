import { TestBed } from '@angular/core/testing';
import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { csrfInterceptor } from './csrf.interceptor';

describe('csrfInterceptor', () => {
  let client: HttpClient;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([csrfInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    client = TestBed.inject(HttpClient);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should add X-Requested-With header on POST request', () => {
    client.post('/test', {}).subscribe();
    const req = httpTesting.expectOne('/test');
    expect(req.request.headers.get('X-Requested-With')).toBe('expense-app');
    req.flush({});
  });

  it('should add X-Requested-With header on DELETE request', () => {
    client.delete('/test').subscribe();
    const req = httpTesting.expectOne('/test');
    expect(req.request.headers.get('X-Requested-With')).toBe('expense-app');
    req.flush({});
  });

  it('should NOT add X-Requested-With header on GET request', () => {
    client.get('/test').subscribe();
    const req = httpTesting.expectOne('/test');
    expect(req.request.headers.has('X-Requested-With')).toBe(false);
    req.flush({});
  });
});