// story: e78s04
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const APP = 'http://localhost:9999';
const EMAIL = `e2e-storage-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  // Register a fresh user for UI tests
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);

  // Login to obtain the auth token
  const loginRes = await request.post('/api/auth/login', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(loginRes.status()).toBe(200);
  const body = await loginRes.json();
  authToken = body.token;
  expect(typeof authToken).toBe('string');
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

test.describe('Storage UI — browser E2E', () => {

  test('Storage page renders', async ({ page }) => {
    // Navigate to Storage
    await page.goto(`${APP}/admin/#/storage`);
    await page.waitForLoadState('networkidle');

    // Verify the page header
    await expect(page.locator('.page-title', { hasText: 'Storage' })).toBeVisible();

    // Verify upload area is present
    await expect(page.locator('.upload-form')).toBeVisible();
    await expect(page.locator('input[type="file"]')).toBeVisible();

    // Verify the upload button is present
    await expect(page.getByRole('button', { name: 'Upload' })).toBeVisible();

    // Verify grid/list view toggle buttons are present
    await expect(page.getByRole('button', { name: 'List' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Grid' })).toBeVisible();

    // Verify the file list section or empty state (no files = no table rendered)
    const emptyText = page.getByText('No files uploaded.');
    const tableWrap = page.locator('.table-wrap');
    const emptyExists = await emptyText.isVisible().catch(() => false);
    const tableExists = await tableWrap.isVisible().catch(() => false);
    await expect(emptyExists || tableExists).toBeTruthy();
  });

  test('Storage empty state', async ({ page }) => {
    // Navigate to Storage
    await page.goto(`${APP}/admin/#/storage`);
    await page.waitForLoadState('networkidle');

    // Verify empty state message appears when no files exist
    await expect(page.getByText('No files uploaded.')).toBeVisible();

    // Verify upload controls remain functional
    await expect(page.locator('.upload-form')).toBeVisible();
    await expect(page.locator('input[type="file"]')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Upload' })).toBeVisible();
  });

});
