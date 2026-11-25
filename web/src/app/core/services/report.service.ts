import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ReportSummary, CategoryReportItem, TrendItem } from '../models/report.model';

@Injectable({
  providedIn: 'root',
})
export class ReportService {
  private readonly http = inject(HttpClient);

  getSummary(month: string): Observable<ReportSummary> {
    const params = new HttpParams().set('month', month);
    return this.http.get<ReportSummary>('/api/reports/summary', { params });
  }

  getByCategory(month: string): Observable<CategoryReportItem[]> {
    const params = new HttpParams().set('month', month);
    return this.http.get<CategoryReportItem[]>('/api/reports/by-category', { params });
  }

  getTrend(month: string, months = 6): Observable<TrendItem[]> {
    const params = new HttpParams().set('month', month).set('months', months.toString());
    return this.http.get<TrendItem[]>('/api/reports/trend', { params });
  }
}