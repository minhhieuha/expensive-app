import {
  Component,
  OnInit,
  AfterViewInit,
  OnDestroy,
  ElementRef,
  ViewChild,
  inject,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { Chart, registerables } from 'chart.js';
import { AuthService } from '../../core/auth/auth.service';
import { ReportService } from '../../core/services/report.service';
import { ReportSummary, CategoryReportItem, TrendItem } from '../../core/models/report.model';
import { MoneyPipe } from '../../shared/pipes/money.pipe';

Chart.register(...registerables);

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, MoneyPipe],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent implements OnInit, AfterViewInit, OnDestroy {
  readonly authService = inject(AuthService);
  private readonly reportService = inject(ReportService);

  @ViewChild('categoryCanvas') categoryCanvasRef?: ElementRef<HTMLCanvasElement>;
  @ViewChild('trendCanvas') trendCanvasRef?: ElementRef<HTMLCanvasElement>;

  private categoryChart?: Chart;
  private trendChart?: Chart;

  selectedMonth = this.currentMonthStr();
  readonly loading = signal<boolean>(false);
  readonly errorMsg = signal<string | null>(null);

  readonly summary = signal<ReportSummary>({
    total_expense: 0,
    total_income: 0,
    balance: 0,
  });
  readonly byCategory = signal<CategoryReportItem[]>([]);
  readonly trend = signal<TrendItem[]>([]);

  currentMonthStr(): string {
    const d = new Date();
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    return `${year}-${month}`;
  }

  ngOnInit(): void {
    this.loadData();
  }

  ngAfterViewInit(): void {
    this.renderCharts();
  }

  ngOnDestroy(): void {
    this.destroyCharts();
  }

  onMonthChange(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading.set(true);
    this.errorMsg.set(null);

    let completed = 0;
    const checkDone = () => {
      completed++;
      if (completed >= 3) {
        this.loading.set(false);
        this.renderCharts();
      }
    };

    this.reportService.getSummary(this.selectedMonth).subscribe({
      next: (s) => {
        this.summary.set(s);
        checkDone();
      },
      error: () => {
        this.errorMsg.set('Không thể tải tóm tắt báo cáo');
        checkDone();
      },
    });

    this.reportService.getByCategory(this.selectedMonth).subscribe({
      next: (cats) => {
        this.byCategory.set(cats);
        checkDone();
      },
      error: () => {
        checkDone();
      },
    });

    this.reportService.getTrend(this.selectedMonth, 6).subscribe({
      next: (t) => {
        this.trend.set(t);
        checkDone();
      },
      error: () => {
        checkDone();
      },
    });
  }

  private destroyCharts(): void {
    if (this.categoryChart) {
      this.categoryChart.destroy();
      this.categoryChart = undefined;
    }
    if (this.trendChart) {
      this.trendChart.destroy();
      this.trendChart = undefined;
    }
  }

  private renderCharts(): void {
    this.destroyCharts();

    // 1. Render biểu đồ tròn phân bổ chi theo danh mục
    const catCanvas = this.categoryCanvasRef?.nativeElement;
    if (catCanvas && this.byCategory().length > 0) {
      const ctx = catCanvas.getContext('2d');
      if (ctx) {
        const labels = this.byCategory().map((c) => c.category_name);
        const data = this.byCategory().map((c) => c.amount);
        const bgColors = [
          '#ef4444', '#f97316', '#f59e0b', '#10b981', '#06b6d4',
          '#3b82f6', '#6366f1', '#8b5cf6', '#ec4899', '#64748b',
        ];

        this.categoryChart = new Chart(ctx, {
          type: 'doughnut',
          data: {
            labels,
            datasets: [
              {
                data,
                backgroundColor: bgColors.slice(0, data.length),
                borderWidth: 1,
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
              legend: {
                position: 'bottom',
              },
            },
          },
        });
      }
    }

    // 2. Render biểu đồ cột xu hướng 6 tháng (Thu & Chi)
    const trendCanvas = this.trendCanvasRef?.nativeElement;
    if (trendCanvas && this.trend().length > 0) {
      const ctx = trendCanvas.getContext('2d');
      if (ctx) {
        const labels = this.trend().map((t) => t.month);
        const incomeData = this.trend().map((t) => t.total_income);
        const expenseData = this.trend().map((t) => t.total_expense);

        this.trendChart = new Chart(ctx, {
          type: 'bar',
          data: {
            labels,
            datasets: [
              {
                label: 'Thu nhập',
                data: incomeData,
                backgroundColor: '#16a34a',
              },
              {
                label: 'Chi phí',
                data: expenseData,
                backgroundColor: '#dc2626',
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
              y: {
                beginAtZero: true,
              },
            },
            plugins: {
              legend: {
                position: 'top',
              },
            },
          },
        });
      }
    }
  }
}