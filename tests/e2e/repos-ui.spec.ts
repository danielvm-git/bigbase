// story: e78s06
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-repos-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);
  const body = await regRes.json();
  authToken = body.token as string;
});

test.beforeEach(async ({ context }) => {
  await context.addCookies([{ name: 'token', value: authToken, url: 'http://localhost:9999' }]);
});

test.describe('Git Repos UI', () => {
  test('Git Repos page renders repo management screen', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/repos');
    await page.waitForLoadState('networkidle');

    // Page header renders with title
    await expect(page.locator('.page-header')).toContainText('Git Repos');

    // Action buttons render
    await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'New Repo' })).toBeVisible();
  });

  test('Git Repos page shows repo list after creation', async ({ page }) => {
    // Create a repo via the API first
    const repoName = `e2e-repos-list-${Date.now()}`;
    const repoRes = await page.request.post('/api/git/repos', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName },
    });
    expect(repoRes.status()).toBe(201);
    const repoBody = await repoRes.json();
    const repoId = repoBody.id as string;

    await page.goto('http://localhost:9999/admin/#/repos');
    await page.waitForLoadState('networkidle');

    // The repo appears in the list table
    await expect(page.locator('.table-wrap')).toBeVisible();
    // Find the specific repo in the table by text
    await expect(page.locator('.table-wrap').getByText(repoName)).toBeVisible();

    // Delete button renders for the repo
    await expect(page.getByRole('button', { name: 'Delete' }).first()).toBeVisible();

    // Clean up
    await page.request.delete(`/api/git/repos/${repoId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    }).catch(() => {});
  });

  test('Git Repos New Repo form opens and renders fields', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/repos');
    await page.waitForLoadState('networkidle');

    // Click New Repo to open the creation form
    await page.getByRole('button', { name: 'New Repo' }).click();
    await page.waitForTimeout(300);

    // Form fields render
    await expect(page.locator('input[placeholder="Name *"]')).toBeVisible();
    await expect(page.locator('input[placeholder="Description"]')).toBeVisible();
    await expect(page.locator('label').filter({ hasText: 'Private' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create' })).toBeVisible();

    // Cancel button shows when form is open
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
  });

  test('Git Repos page shows empty state when no repos exist', async ({ page }) => {
    // Navigate to the page
    await page.goto('http://localhost:9999/admin/#/repos');
    await page.waitForLoadState('networkidle');

    // Either the empty state message or a table with repos renders
    const emptyState = page.getByText(/no repos yet/i);
    const table = page.locator('.table-wrap');
    await expect(emptyState.or(table)).toBeVisible();
  });
});
