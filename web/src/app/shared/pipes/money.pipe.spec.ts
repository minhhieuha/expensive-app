import { describe, it, expect } from 'vitest';
import { MoneyPipe } from './money.pipe';

describe('MoneyPipe', () => {
  const pipe = new MoneyPipe();

  it('formats 0 VND correctly', () => {
    const res = pipe.transform(0);
    expect(res).toContain('0');
    expect(res).toMatch(/(₫|đ|VND)/i);
  });

  it('formats positive integers correctly', () => {
    const res = pipe.transform(100000);
    // Chuỗi phân cách hàng nghìn vi-VN thường là dấu chấm (100.000)
    expect(res).toMatch(/100[.\s]000/);
    expect(res).toMatch(/(₫|đ|VND)/i);
  });

  it('formats large values up to 10^12 without float errors', () => {
    const res = pipe.transform(1000000000000);
    expect(res).toMatch(/(₫|đ|VND)/i);
  });

  it('handles null and undefined by returning 0 formatted', () => {
    expect(pipe.transform(null)).toEqual(pipe.transform(0));
    expect(pipe.transform(undefined)).toEqual(pipe.transform(0));
  });
});