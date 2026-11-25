import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ReportService } from './report.service';
import { ReportSummary, CategoryReportItem, TrendItem } from '../models/report.model';

describe('ReportService', () => {
  let service: ReportService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(ReportService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('gets summary report for month', () => {
    const mockSummary: ReportSummary = {
      total_expense: 3000000,
      total_income: 10000000,
      balance: 7000000,
    };

    service.getSummary('2026-11').subscribe((res) => {
      expect(res).toEqual(mockSummary);
    });

    const req = httpTesting.expectOne('/api/reports/summary?month=2026-11');
    expect(req.request.method).toBe('GET');
    req.flush(mockSummary);
  });

  it('gets by-category report for month', () => {
    const mockByCat: CategoryReportItem[] = [
      { category_id: 1, category_name: 'Ăn uống', amount: 3000000, percentage: 75 },
    ];

    service.getByCategory('2026-11').subscribe((res) => {
      expect(res).toEqual(mockByCat);
    });

    const req = httpTesting.expectOne('/api/reports/by-category?month=2026-11');
    expect(req.request.method).toBe('GET');
    req.flush(mockByCat);
  });

  it('gets trend report for month with custom months count', () => {
    const mockTrend: TrendItem[] = [
      { month: '2026-10', total_expense: 2000000, total_income: 8000000 },
      { month: '2026-11', total_expense: 3000000, total_income: 10000000 },
    ];

    service.getTrend('2026-11', 2).subscribe((res) => {
      expect(res).toEqual(mockTrend);
    });

    const req = httpTesting.expectOne('/api/reports/trend?month=2026-11&months=2');
    expect(req.request.method).toBe('GET');
    req.flush(mockTrend);
  });
});