// story: e78s01
// story: e77s01
// story: e43s01
import { test, expect } from '@playwright/test';

test('GET /health returns 200 with status field', async ({ request }) => {
  const res = await request.get('/health');
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('status');
  expect(typeof body.status).toBe('string');
});

test('GET /health returns components and running counts', async ({ request }) => {
  const res = await request.get('/health');
  expect(res.status()).toBe(200);

  const body = await res.json();
  expect(body).toHaveProperty('components');
  expect(typeof body.components).toBe('number');
  expect(body).toHaveProperty('running');
  expect(typeof body.running).toBe('number');
});
