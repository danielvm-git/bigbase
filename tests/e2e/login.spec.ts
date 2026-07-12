// story: e78s01
import { test, expect } from '@playwright/test';

test('POST /api/auth/register creates user and returns token', async ({ request }) => {
  const email = `e2e-${Date.now()}@test.com`;
  const res = await request.post('/api/auth/register', {
    data: { email, password: 'TestPass123!' },
  });
  expect(res.status()).toBe(201);

  const body = await res.json();
  expect(body).toHaveProperty('token');
  expect(typeof body.token).toBe('string');
  expect(body).toHaveProperty('user');
  expect(body.user).toHaveProperty('email', email);
  expect(body.user).toHaveProperty('id');
});

test('POST /api/auth/login with valid credentials returns JWT', async ({ request }) => {
  const email = `e2e-login-${Date.now()}@test.com`;

  // Register first
  await request.post('/api/auth/register', {
    data: { email, password: 'TestPass123!' },
  });

  // Login
  const res = await request.post('/api/auth/login', {
    data: { email, password: 'TestPass123!' },
  });
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('token');
  expect(typeof body.token).toBe('string');
  expect(body).toHaveProperty('user');
  expect(body.user).toHaveProperty('email', email);
});

test('POST /api/auth/register rejects duplicate email', async ({ request }) => {
  const email = `e2e-dup-${Date.now()}@test.com`;

  await request.post('/api/auth/register', {
    data: { email, password: 'TestPass123!' },
  });

  const res = await request.post('/api/auth/register', {
    data: { email, password: 'TestPass123!' },
  });
  expect(res.status()).toBe(409);
});
