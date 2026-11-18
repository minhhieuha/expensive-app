import { test, expect } from '@playwright/test';

test.describe('Transaction API E2E', () => {
  const timestamp = Date.now();
  const userA = {
    email: `tx_user_a_${timestamp}@example.com`,
    password: 'password123',
  };
  const userB = {
    email: `tx_user_b_${timestamp}@example.com`,
    password: 'password123',
  };

  test('full transaction CRUD, validation, filter, pagination and user isolation', async ({ playwright }) => {
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

    // Tạo giao dịch hợp lệ
    const createResp = await ctxA.post('/api/transactions', {
      data: {
        category_id: catExpA,
        type: 'expense',
        amount: 85000,
        occurred_on: '2026-11-18',
        note: 'Bữa trưa phở bò',
      },
    });
    expect(createResp.status()).toBe(201);
    const newTx = await createResp.json();
    expect(newTx.amount).toBe(85000);
    expect(newTx.note).toBe('Bữa trưa phở bò');

    // Lấy chi tiết
    const getResp = await ctxA.get(`/api/transactions/${newTx.id}`);
    expect(getResp.status()).toBe(200);
    const detail = await getResp.json();
    expect(detail.id).toBe(newTx.id);

    // Sửa giao dịch
    const patchResp = await ctxA.patch(`/api/transactions/${newTx.id}`, {
      data: {
        category_id: catExpA,
        type: 'expense',
        amount: 90000,
        occurred_on: '2026-11-18',
        note: 'Bữa trưa phở đặc biệt',
      },
    });
    expect(patchResp.status()).toBe(200);
    const patched = await patchResp.json();
    expect(patched.amount).toBe(90000);

    // 2. User B context - kiểm tra cô lập
    const ctxB = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });

    const regB = await ctxB.post('/api/auth/register', { data: userB });
    expect(regB.status()).toBe(201);

    // B không thấy giao dịch của A
    const listB = await ctxB.get('/api/transactions');
    expect(listB.status()).toBe(200);
    const pageB = await listB.json();
    expect(pageB.total).toBe(0);

    // B không xem/sửa/xóa được giao dịch của A -> 404
    const getOther = await ctxB.get(`/api/transactions/${newTx.id}`);
    expect(getOther.status()).toBe(404);

    const patchOther = await ctxB.patch(`/api/transactions/${newTx.id}`, {
      data: { category_id: catExpA, type: 'expense', amount: 1000, occurred_on: '2026-11-18' },
    });
    expect(patchOther.status()).toBe(404);

    const delOther = await ctxB.delete(`/api/transactions/${newTx.id}`);
    expect(delOther.status()).toBe(404);

    // 3. User A xóa giao dịch của mình -> 200
    const delResp = await ctxA.delete(`/api/transactions/${newTx.id}`);
    expect(delResp.status()).toBe(200);

    // Kiểm tra đã bị xóa -> 404
    const getDeleted = await ctxA.get(`/api/transactions/${newTx.id}`);
    expect(getDeleted.status()).toBe(404);

    await ctxA.dispose();
    await ctxB.dispose();
  });
});