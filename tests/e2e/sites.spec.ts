import { test, expect } from '@playwright/test';

const email = `e2e-sites-${Date.now()}@test.com`;
const password = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  // Register and login to get auth token
  await request.post('/api/auth/register', {
    data: { email, password },
  });

  const res = await request.post('/api/auth/login', {
    data: { email, password },
  });
  const body = await res.json();
  authToken = body.token;
});

test('GET /api/sites returns 200 with data array', async ({ request }) => {
  const res = await request.get('/api/sites', {
    headers: { Authorization: `Bearer ${authToken}` },
  });
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('data');
  expect(Array.isArray(body.data)).toBe(true);
});

test('POST /api/sites returns 400 when required fields missing', async ({ request }) => {
  const res = await request.post('/api/sites', {
    headers: { Authorization: `Bearer ${authToken}` },
    data: { name: 'e2e-test-site' },
  });
  expect(res.status()).toBe(400);

  const body = await res.json();
  expect(body).toHaveProperty('error');
});

test('GET /api/deploy returns 200 with data array', async ({ request }) => {
  const res = await request.get('/api/deploy', {
    headers: { Authorization: `Bearer ${authToken}` },
  });
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('data');
  expect(Array.isArray(body.data)).toBe(true);
});
