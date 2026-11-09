import { test, expect } from '@playwright/test';

test.describe('Auth API E2E', () => {
  const testUser = {
    email: `e2e_${Date.now()}@example.com`,
    password: 'securePassword123',
  };

  test('register -> get /api/me -> logout -> get /api/me fails', async ({ request }) => {
    // 1. Register
    const regResp = await request.post('/api/auth/register', {
      data: testUser,
    });
    expect(regResp.status()).toBe(201);
    const regBody = await regResp.json();
    expect(regBody.email).toBe(testUser.email.toLowerCase());

    // 2. GET /api/me (cookie is maintained by request context)
    const meResp = await request.get('/api/me');
    expect(meResp.status()).toBe(200);
    const meBody = await meResp.json();
    expect(meBody.email).toBe(testUser.email.toLowerCase());

    // 3. Logout
    const logoutResp = await request.post('/api/auth/logout');
    expect(logoutResp.status()).toBe(200);

    // 4. GET /api/me again should fail with 401
    const meAfterLogout = await request.get('/api/me');
    expect(meAfterLogout.status()).toBe(401);
  });

  test('login with invalid credentials returns 401', async ({ request }) => {
    const resp = await request.post('/api/auth/login', {
      data: {
        email: 'nobody@example.com',
        password: 'wrongpassword',
      },
    });
    expect(resp.status()).toBe(401);
    const body = await resp.json();
    expect(body.error.code).toBe('invalid_credentials');
  });
});
