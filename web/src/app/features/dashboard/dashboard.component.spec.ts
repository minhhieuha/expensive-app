import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { signal } from '@angular/core';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { of } from 'rxjs';
import { DashboardComponent } from './dashboard.component';
import { ReportService } from '../../core/services/report.service';
import { AuthService } from '../../core/auth/auth.service';
import { ReportSummary, CategoryReportItem, TrendItem } from '../../core/models/report.model';
import { User } from '../../core/models/user.model';

describe('DashboardComponent', () => {
  let component: DashboardComponent;
  let fixture: ComponentFixture<DashboardComponent>;
  let mockReportService: Partial<ReportService>;
  let mockAuthService: Partial<AuthService>;

  const mockUser: User = {
    id: 1,
    email: 'user@example.com',
    timezone: 'Asia/Ho_Chi_Minh',
    created_at: '2026-11-20T00:00:00Z',
  };

  const mockSummary: ReportSummary = {
    total_expense: 4000000,
    total_income: 15000000,
    balance: 11000000,
  };

  const mockByCat: CategoryReportItem[] = [
    { category_id: 1, category_name: 'Ăn uống', amount: 3000000, percentage: 75 },
    { category_id: 2, category_name: 'Di chuyển', amount: 1000000, percentage: 25 },
  ];

  const mockTrend: TrendItem[] = [
    { month: '2026-10', total_expense: 2000000, total_income: 10000000 },
    { month: '2026-11', total_expense: 4000000, total_income: 15000000 },
  ];

  beforeEach(async () => {
    // Mock getContext for canvas in JSDOM environment
    HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue(null);

    mockReportService = {
      getSummary: vi.fn().mockReturnValue(of(mockSummary)),
      getByCategory: vi.fn().mockReturnValue(of(mockByCat)),
      getTrend: vi.fn().mockReturnValue(of(mockTrend)),
    };

    mockAuthService = {
      currentUser: signal<User | null>(mockUser),
    };

    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [
        provideRouter([]),
        { provide: ReportService, useValue: mockReportService },
        { provide: AuthService, useValue: mockAuthService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DashboardComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('creates component and loads dashboard data', () => {
    expect(component).toBeTruthy();
    expect(mockReportService.getSummary).toHaveBeenCalledWith(component.selectedMonth);
    expect(mockReportService.getByCategory).toHaveBeenCalledWith(component.selectedMonth);
    expect(mockReportService.getTrend).toHaveBeenCalledWith(component.selectedMonth, 6);

    expect(component.summary().total_expense).toBe(4000000);
    expect(component.summary().total_income).toBe(15000000);
    expect(component.summary().balance).toBe(11000000);
    expect(component.byCategory().length).toBe(2);
    expect(component.trend().length).toBe(2);
  });

  it('reloads data when month changes', () => {
    component.selectedMonth = '2026-10';
    component.onMonthChange();

    expect(mockReportService.getSummary).toHaveBeenCalledWith('2026-10');
    expect(mockReportService.getByCategory).toHaveBeenCalledWith('2026-10');
  });
});