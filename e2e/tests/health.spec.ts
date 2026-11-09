import { test, expect } from '@playwright/test';

test.describe('Health check API', () => {
  test('GET /api/health should return 200 OK with status ok', async ({ request }) => {
    const response = await request.get('/api/health');
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toEqual({ status: 'ok' });
  });

  test('POST /api/health should return 405 Method Not Allowed', async ({ request }) => {
    const response = await request.post('/api/health');
    expect(response.status()).toBe(405);
  });
});
