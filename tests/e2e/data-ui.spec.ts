// story: e78s04
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const APP = 'http://localhost:9999';
const EMAIL = `e2e-data-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;
let collectionName: string;

test.beforeAll(async ({ request }) => {
  // Register a fresh user for UI tests
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);

  // Login to get a token for API setup
  const loginRes = await request.post('/api/auth/login', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(loginRes.status()).toBe(200);
  const loginBody = await loginRes.json() as { token: string };
  authToken = loginBody.token;

  // Create a test collection so Data Studio has data to display
  collectionName = `e2e_data_${Date.now()}`;
  const colRes = await request.post(`/api/collections/${collectionName}`, {
    headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
    data: { title: 'test record', value: 42 },
  });
  expect(colRes.status()).toBe(201);
});

test.beforeEach(async ({ context }) => {
  // Set the auth cookie so the SPA authenticates via /api/auth/me.
  await context.addCookies([
    {
      name: 'token',
      value: authToken,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      sameSite: 'Strict',
    },
  ]);
});

test.describe('Data UI — browser E2E', () => {

  test('Data Studio page renders', async ({ page }) => {
    // Navigate to Data Studio
    await page.goto(`${APP}/admin/#/data`);
    await page.waitForLoadState('networkidle');

    // Verify the page title is present (use .first() to avoid sidebar text match)
    await expect(page.locator('.page-title', { hasText: 'Data Studio' })).toBeVisible();

    // Verify the collection list sidebar is present
    await expect(page.locator('.collection-list')).toBeVisible();

    // Wait for collection buttons to load (async fetch completes)
    await expect(page.locator('.collection-btn').first()).toBeVisible({ timeout: 10000 });

    // Click the test collection to reveal Data/Schema toggle
    await page.locator('.collection-btn', { hasText: collectionName }).click();

    // Verify Data/Schema toggle buttons appear (scope to toggle area to avoid collection-btn matches)
    await expect(page.locator('.studio-mode-toggle').getByRole('button', { name: 'Data', exact: true })).toBeVisible();
    await expect(page.locator('.studio-mode-toggle').getByRole('button', { name: 'Schema', exact: true })).toBeVisible();

    // Switch to Schema view and verify schema controls
    await page.getByRole('button', { name: 'Schema' }).click();
    await expect(page.getByText('Add column')).toBeVisible();

    // Switch back to Data view and verify data controls
    await page.locator('.studio-mode-toggle').getByRole('button', { name: 'Data', exact: true }).click();
    await expect(page.locator('input[placeholder*="filter"]')).toBeVisible();
  });

  test('SQL Editor page renders', async ({ page }) => {
    // Navigate to SQL Editor
    await page.goto(`${APP}/admin/#/sql`);
    await page.waitForLoadState('networkidle');

    // Verify the page title
    await expect(page.locator('.page-title', { hasText: 'SQL Editor' })).toBeVisible();

    // Verify the SQL textarea / editor input is present
    await expect(page.locator('.sql-textarea')).toBeVisible();

    // Verify a "Run" or "Execute" button is present
    const runBtn = page.getByRole('button', { name: /run/i });
    await expect(runBtn).toBeVisible();
  });

});
