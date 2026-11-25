import { test, expect } from '@playwright/test';

test.describe('Report API E2E', () => {
  const timestamp = Date.now();
  const userA = {
    email: `report_user_a_${timestamp}@example.com`,
    password: 'password123',
  };
  const userB = {
    email: `report_user_b_${timestamp}@example.com`,
    password: 'password123',
  };

  test('reports summary, by-category, trend, and user isolation', async ({ playwright }) => {
    const baseURL = process.env['BASE_URL'] || `http://127.0.0.1:${process.env['PORT'] || '8088'}`;

    // 1. User A context
    const ctxA = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });

    const regA = await ctxA.post('/api/auth/register', { data: userA });
    expect(regA.status()).toBe(201);

    // Lấy danh mục mặc định
    const listExpA = await ctxA.get('/api/categories?kind=expense');
    const expCats = await listExpA.json();
    const catExp1 = expCats[0].id;
    const catExp2 = expCats[1].id;

    const listIncA = await ctxA.get('/api/categories?kind=income');
    const incCats = await listIncA.json();
    const catInc = incCats[0].id;

    // Thêm thu: 15,000,000 ngày 2026-11-01
    await ctxA.post('/api/transactions', {
      data: {
        category_id: catInc,
        type: 'income',
        amount: 15000000,
        occurred_on: '2026-11-01',
        note: 'Tiền lương',
      },
    });

    // Thêm chi 1: 3,000,000 ngày 2026-11-10
    await ctxA.post('/api/transactions', {
      data: {
        category_id: catExp1,
        type: 'expense',
        amount: 3000000,
        occurred_on: '2026-11-10',
        note: 'Khoản chi 1',
      },
    });

    // Thêm chi 2: 1,000,000 ngày 2026-11-20
    await ctxA.post('/api/transactions', {
      data: {
        category_id: catExp2,
        type: 'expense',
        amount: 1000000,
        occurred_on: '2026-11-20',
        note: 'Khoản chi 2',
      },
    });

    // GET /api/reports/summary
    const sumResp = await ctxA.get('/api/reports/summary?month=2026-11');
    expect(sumResp.status()).toBe(200);
    const sum = await sumResp.json();
    expect(sum.total_income).toBe(15000000);
    expect(sum.total_expense).toBe(4000000);
    expect(sum.balance).toBe(11000000);

    // GET /api/reports/by-category
    const byCatResp = await ctxA.get('/api/reports/by-category?month=2026-11');
    expect(byCatResp.status()).toBe(200);
    const byCat = await byCatResp.json();
    expect(byCat.length).toBe(2);
    expect(byCat[0].amount).toBe(3000000);
    expect(byCat[0].percentage).toBe(75);
    expect(byCat[1].amount).toBe(1000000);
    expect(byCat[1].percentage).toBe(25);

    // GET /api/reports/trend
    const trendResp = await ctxA.get('/api/reports/trend?month=2026-11&months=6');
    expect(trendResp.status()).toBe(200);
    const trend = await trendResp.json();
    expect(trend.length).toBe(6);
    expect(trend[5].month).toBe('2026-11');
    expect(trend[5].total_expense).toBe(4000000);
    expect(trend[5].total_income).toBe(15000000);

    // 2. User B context - kiểm tra cô lập
    const ctxB = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });
    const regB = await ctxB.post('/api/auth/register', { data: userB });
    expect(regB.status()).toBe(201);

    const sumB = await (await ctxB.get('/api/reports/summary?month=2026-11')).json();
    expect(sumB.total_expense).toBe(0);
    expect(sumB.total_income).toBe(0);
    expect(sumB.balance).toBe(0);

    const byCatB = await (await ctxB.get('/api/reports/by-category?month=2026-11')).json();
    expect(byCatB.length).toBe(0);
  });
});