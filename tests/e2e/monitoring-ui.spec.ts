// story: e78s06
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-monitoring-ui-${Date.now()}@test.com`;
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

test.describe('Monitoring UI', () => {
  test('Monitoring page renders overview metrics', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/monitoring');
    await page.waitForLoadState('networkidle');

    // Wait for the page header to render
    await expect(page.locator('.page-header')).toBeVisible();
    await expect(page.locator('.page-header')).toContainText('Monitoring');

    // Verify the Overview tab is active by default and system stats render
    await expect(page.locator('.stats-grid').first()).toBeVisible();
    // CPU stat should display a percentage
    await expect(page.locator('.stat-label')).toContainText(['CPU', 'Heap MB', 'Goroutines', 'Uptime']);
  });

  test('Monitoring Host tab shows hardware gauges', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/monitoring');
    await page.waitForLoadState('networkidle');

    // switch to the Host tab (tabs are .tab buttons, not role="tab")
    await page.locator('.tab', { hasText: 'Host' }).click();
    await page.waitForTimeout(500);

    // Host tab sections: CPU, Memory, Disk, Network
    await expect(page.locator('.section-title')).toContainText(['CPU', 'Memory', 'Disk', 'Network']);
  });

  test('Monitoring Logs tab renders log search', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/monitoring');
    await page.waitForLoadState('networkidle');

    // Switch to the Logs tab
    await page.locator('.tab', { hasText: 'Logs' }).click();
    await page.waitForTimeout(500);

    // Verify log search input renders
    await expect(page.locator('input[placeholder="Search logs..."]')).toBeVisible();
    // The log table or empty-state text renders
    await expect(page.getByText(/no logs/i).or(page.locator('.table-wrap'))).toBeVisible();
  });

  test('Monitoring Alerts tab shows alert configuration', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/monitoring');
    await page.waitForLoadState('networkidle');

    // Switch to the Alerts tab
    await page.locator('.tab', { hasText: 'Alerts' }).click();
    await page.waitForTimeout(500);

    // New Alert button renders
    await expect(page.getByRole('button', { name: 'New Alert' })).toBeVisible();

    // Click New Alert to show the alert creation form
    await page.getByRole('button', { name: 'New Alert' }).click();
    await page.waitForTimeout(300);

    // Alert form fields appear: Name, Metric, Threshold inputs
    await expect(page.locator('input[placeholder="Name *"]')).toBeVisible();
    await expect(page.locator('input[placeholder="Metric *"]')).toBeVisible();
    await expect(page.locator('input[placeholder="Threshold"]')).toBeVisible();
    // Cancel button renders after form is shown
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
  });
});
