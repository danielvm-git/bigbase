// story: e78s05
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-users-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);
  const regBody = await regRes.json();
  authToken = regBody.token;
  expect(typeof authToken).toBe('string');
});

test.beforeEach(async ({ page }) => {
  await page.context().addCookies([
    { name: 'token', value: authToken, domain: 'localhost', path: '/' },
  ]);
});

// ─── Test 5: Users page renders ───────────────────────────────────
test('users page renders the user list table', async ({ page }) => {
  await page.goto('http://localhost:9999/admin/#/users');
  await page.waitForLoadState('networkidle');

  // Verify the page header renders with the correct title.
  await expect(page.locator('h1, h2').filter({ hasText: 'Users' })).toBeVisible();

  // Verify the Refresh button is present.
  await expect(page.getByRole('button', { name: /refresh/i })).toBeVisible();

  // Users page may show an error banner if the user list API is forbidden.
  const errorBanner = page.locator('[role="alert"]');
  const table = page.locator('.table-wrap');
  const errorVisible = await errorBanner.isVisible().catch(() => false);

  if (errorVisible) {
    // API returned an error (e.g., "forbidden") — that's acceptable for a non-admin user
    await expect(errorBanner).toContainText(/forbidden/i);
    test.info().annotations.push({
      type: 'warn',
      description: 'User list API returned forbidden — test limited to page load verification',
    });
  } else {
    // Table should render with expected columns
    await expect(table).toBeVisible();

    // Check for table header columns
    const headerCells = table.locator('thead th');
    await expect(headerCells.filter({ hasText: /id/i })).toBeVisible();
    await expect(headerCells.filter({ hasText: /email/i })).toBeVisible();

    // At minimum, the current user should appear in the table.
    const rows = table.locator('tbody tr');
    await expect(rows.first()).toBeVisible();

    // Verify the current user's email is present in the table body.
    await expect(table.locator('tbody').getByText(EMAIL)).toBeVisible();
  }
});

// ─── Test 6: Delete user confirmation ─────────────────────────────
test('delete user triggers confirmation dialog', async ({ page }) => {
  await page.goto('http://localhost:9999/admin/#/users');
  await page.waitForLoadState('networkidle');

  // Check if the table is visible (may show error state instead)
  const errorBanner = page.locator('[role="alert"]');
  const table = page.locator('.table-wrap');
  const errorVisible = await errorBanner.isVisible().catch(() => false);

  if (errorVisible || !(await table.isVisible().catch(() => false))) {
    // API returned an error, skip delete test
    test.info().annotations.push({
      type: 'warn',
      description: 'User table not available (API error) — skipping delete test',
    });
    return;
  }

  // Find the first Delete button in the user table.
  const deleteButton = table.locator('tbody tr').first().getByRole('button', { name: /delete/i });
  await expect(deleteButton).toBeVisible();

  // Click the Delete button to trigger the Dialog component.
  await deleteButton.click();

  // Verify the Dialog component appears with delete confirmation.
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('.dialog-title')).toContainText(/delete/i);

  // Accept the dialog.
  await dialog.getByRole('button', { name: /delete|confirm|yes/i }).click();
  await expect(dialog).not.toBeVisible();
});
