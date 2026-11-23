import { test, expect } from '@playwright/test';

test.describe('Budget API E2E', () => {
  const timestamp = Date.now();
  const userA = {
    email: `budget_user_a_${timestamp}@example.com`,
    password: 'password123',
  };
  const userB = {
    email: `budget_user_b_${timestamp}@example.com`,
    password: 'password123',
  };

  test('full budget upsert, spent & status calculation, and user isolation', async ({ playwright }) => {
    const baseURL = process.env['BASE_URL'] || `http://127.0.0.1:${process.env['PORT'] || '8088'}`;

    // 1. User A context
    const ctxA = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });

    const regA = await ctxA.post('/api/auth/register', { data: userA });
    expect(regA.status()).toBe(201);

    // Lấy danh mục mặc định của A
    const listCatsA = await ctxA.get('/api/categories?kind=expense');
    expect(listCatsA.status()).toBe(200);
    const catsA = await listCatsA.json();
    expect(catsA.length).toBeGreaterThan(0);
    const catExpA = catsA[0].id;

    // Đặt ngân sách 1,000,000 cho tháng 2026-11
    const putResp = await ctxA.put('/api/budgets', {
      data: {
        category_id: catExpA,
        month: '2026-11',
        amount: 1000000,
      },
    });
    expect(putResp.status()).toBe(200);
    const newBudget = await putResp.json();
    expect(newBudget.amount).toBe(1000000);
    expect(newBudget.status).toBe('ok');
    expect(newBudget.spent).toBe(0);

    // Thêm giao dịch chi tiêu 850,000 (85%)
    const createTxResp = await ctxA.post('/api/transactions', {
      data: {
        category_id: catExpA,
        type: 'expense',
        amount: 850000,
        occurred_on: '2026-11-15',
        note: 'Tiền ăn uống',
      },
    });
    expect(createTxResp.status()).toBe(201);

    // Lấy danh sách ngân sách tháng 2026-11 -> status phải là "warning"
    const getBudgetsResp = await ctxA.get('/api/budgets?month=2026-11');
    expect(getBudgetsResp.status()).toBe(200);
    const budgetsList = await getBudgetsResp.json();
    expect(budgetsList.length).toBe(1);
    expect(budgetsList[0].spent).toBe(850000);
    expect(budgetsList[0].status).toBe('warning');

    // 2. User B context - kiểm tra cô lập dữ liệu
    const ctxB = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });
    const regB = await ctxB.post('/api/auth/register', { data: userB });
    expect(regB.status()).toBe(201);

    // User B không thấy ngân sách của User A
    const getBudgetsB = await ctxB.get('/api/budgets?month=2026-11');
    expect(getBudgetsB.status()).toBe(200);
    const listB = await getBudgetsB.json();
    expect(listB.length).toBe(0);

    // User B không thể xóa ngân sách của User A
    const delB = await ctxB.delete(`/api/budgets/${newBudget.id}`);
    expect(delB.status()).toBe(404);

    // User A xóa ngân sách thành công
    const delA = await ctxA.delete(`/api/budgets/${newBudget.id}`);
    expect(delA.status()).toBe(200);
  });
});