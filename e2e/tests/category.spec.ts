import { test, expect } from '@playwright/test';

test.describe('Category API E2E', () => {
  const timestamp = Date.now();
  const userA = {
    email: `cat_a_${timestamp}@example.com`,
    password: 'password123',
  };
  const userB = {
    email: `cat_b_${timestamp}@example.com`,
    password: 'password123',
  };

  test('full category CRUD and user isolation', async ({ playwright }) => {
    const baseURL = process.env['BASE_URL'] || `http://127.0.0.1:${process.env['PORT'] || '8088'}`;

    // 1. Context cho User A
    const ctxA = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });

    const regA = await ctxA.post('/api/auth/register', { data: userA });
    expect(regA.status()).toBe(201);

    // List categories mặc định của A (11 danh mục)
    const listA1 = await ctxA.get('/api/categories');
    expect(listA1.status()).toBe(200);
    const catsA1 = await listA1.json();
    expect(catsA1.length).toBe(11);

    // Tạo category mới cho A
    const createResp = await ctxA.post('/api/categories', {
      data: { name: 'Sách vở', kind: 'expense' },
    });
    expect(createResp.status()).toBe(201);
    const newCat = await createResp.json();
    expect(newCat.name).toBe('Sách vở');

    // Đổi tên category
    const patchResp = await ctxA.patch(`/api/categories/${newCat.id}`, {
      data: { name: 'Sách báo' },
    });
    expect(patchResp.status()).toBe(200);
    const patchedCat = await patchResp.json();
    expect(patchedCat.name).toBe('Sách báo');

    // 2. Context cho User B
    const ctxB = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Requested-With': 'expense-app' },
    });

    const regB = await ctxB.post('/api/auth/register', { data: userB });
    expect(regB.status()).toBe(201);

    // User B không thấy 'Sách báo'
    const listB = await ctxB.get('/api/categories');
    expect(listB.status()).toBe(200);
    const catsB = await listB.json();
    expect(catsB.some((c: { id: number; name: string }) => c.id === newCat.id || c.name === 'Sách báo')).toBe(false);

    // User B cố sửa hoặc xóa category của A -> 404
    const patchOther = await ctxB.patch(`/api/categories/${newCat.id}`, { data: { name: 'Hack' } });
    expect(patchOther.status()).toBe(404);

    const deleteOther = await ctxB.delete(`/api/categories/${newCat.id}`);
    expect(deleteOther.status()).toBe(404);

    // 3. User A xóa category của mình -> 200
    const deleteResp = await ctxA.delete(`/api/categories/${newCat.id}`);
    expect(deleteResp.status()).toBe(200);

    // Kiểm tra đã bị xóa -> 404
    const patchDeleted = await ctxA.patch(`/api/categories/${newCat.id}`, { data: { name: 'Recheck' } });
    expect(patchDeleted.status()).toBe(404);

    await ctxA.dispose();
    await ctxB.dispose();
  });
});