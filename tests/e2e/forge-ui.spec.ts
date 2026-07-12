// story: e78s06
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-forge-ui-${Date.now()}@test.com`;
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

test.describe('Forge & Repos UI', () => {
  test('Forge page renders with repo selector and issue tracker', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/forge');
    await page.waitForLoadState('networkidle');

    // Page header renders
    await expect(page.locator('.page-header')).toContainText('Forge');

    // The initial view shows a repo selector (no repo selected yet)
    const repoSelect = page.locator('select');
    await expect(repoSelect).toBeVisible();
    await expect(repoSelect).toContainText('Select repo...');
  });

  test('Forge page shows issue tracker after selecting a repo', async ({ page }) => {
    // First, create a git repo via the API so there is something to select
    const repoName = `e2e-forge-ui-${Date.now()}`;
    const repoRes = await page.request.post('/api/git/repos', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName },
    });
    expect(repoRes.status()).toBe(201);
    const repoBody = await repoRes.json();
    const repoId = repoBody.id as string;

    await page.goto('http://localhost:9999/admin/#/forge');
    await page.waitForLoadState('networkidle');

    // Select the created repo from the dropdown
    await page.locator('select').first().selectOption({ label: repoName });
    await page.waitForTimeout(1000);

    // After repo selection, the issue tracker and Kanban board tabs render (as .tab buttons)
    await expect(page.locator('.tab', { hasText: 'Issues' })).toBeVisible();
    await expect(page.locator('.tab', { hasText: 'Board' })).toBeVisible();

    // The New Issue button renders
    await expect(page.getByRole('button', { name: 'New Issue' })).toBeVisible();

    // Clean up the created repo
    await page.request.delete(`/api/git/repos/${repoId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    }).catch(() => {});
  });

  test('Forge Board tab renders Kanban columns', async ({ page }) => {
    // Create a repo for the board test
    const repoName = `e2e-forge-board-${Date.now()}`;
    const repoRes = await page.request.post('/api/git/repos', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName },
    });
    expect(repoRes.status()).toBe(201);
    const repoBody = await repoRes.json();
    const repoId = repoBody.id as string;

    await page.goto('http://localhost:9999/admin/#/forge');
    await page.waitForLoadState('networkidle');
    await page.locator('select').first().selectOption({ label: repoName });
    await page.waitForTimeout(1000);

    // Switch to the Board tab
    await page.locator('.tab', { hasText: 'Board' }).click();
    await page.waitForTimeout(500);

    // Kanban columns render: open, in progress, review, closed
    const board = page.locator('.board');
    await expect(board).toBeVisible();
    await expect(board.locator('.board-col-title')).toHaveText([
      'open',
      'in progress',
      'review',
      'closed',
    ]);

    // Clean up
    await page.request.delete(`/api/git/repos/${repoId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    }).catch(() => {});
  });
});
