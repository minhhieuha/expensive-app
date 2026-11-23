import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'money',
  standalone: true,
})
export class MoneyPipe implements PipeTransform {
  private readonly formatter = new Intl.NumberFormat('vi-VN', {
    style: 'currency',
    currency: 'VND',
    maximumFractionDigits: 0,
  });

  transform(value: number | null | undefined): string {
    if (value === null || value === undefined || isNaN(value)) {
      return this.formatter.format(0);
    }
    // Đảm bảo làm tròn số nguyên không phần lẻ
    const intVal = Math.round(value);
    return this.formatter.format(intVal);
  }
}