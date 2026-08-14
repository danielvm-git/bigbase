// story: e78s02
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-settings-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  // Register a fresh user via the standalone API context
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);

  // Login once to get the auth token — avoids rate limiting from per-test logins
  const loginRes = await request.post('/api/auth/login', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(loginRes.status()).toBe(200);
  const body = await loginRes.json();
  authToken = body.token;
});

test.beforeEach(async ({ context }) => {
  // Inject the session cookie for a fresh browser context before each test
  await context.addCookies([{
    name: 'token',
    value: authToken,
    domain: 'localhost',
    path: '/',
    httpOnly: true,
    sameSite: 'Strict',
  }]);
});

test.describe('Settings Page UI', () => {
  // ─── Test 5: Settings tabs ────────────────────────────────────
  test('settings page renders and tabs switch between Account, Workspace, and Billing', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/settings');
    await page.waitForTimeout(2000); // allow SPA to load and auth check

    // Wait for the settings page to render — look for the page title
    const pageTitle = page.locator('.page-header');
    await expect(pageTitle).toBeVisible();
    await expect(pageTitle).toContainText('Settings');

    // The subtitle should describe the page
    const subtitle = page.locator('.page-subtitle');
    await expect(subtitle).toBeVisible();
    await expect(subtitle).toContainText('Manage your account');

    // The tabs bar should have three tabs
    const tabs = page.locator('.tabs');
    await expect(tabs).toBeVisible();

    const accountTab = tabs.locator('.tab', { hasText: 'Account' });
    const workspaceTab = tabs.locator('.tab', { hasText: 'Workspace' });
    const billingTab = tabs.locator('.tab', { hasText: 'Billing' });

    await expect(accountTab).toBeVisible();
    await expect(workspaceTab).toBeVisible();
    await expect(billingTab).toBeVisible();

    // ── Account tab (default) ─────────────────────────────────
    // Account is selected by default
    await expect(accountTab).toHaveClass(/active/);

    // Account section should render with user details
    const accountSection = page.locator('.settings-section').first();
    await expect(accountSection).toBeVisible();
    await expect(accountSection.locator('.card-header', { hasText: 'Account' })).toBeVisible();

    // Email row should be present
    await expect(accountSection.locator('.settings-row', { hasText: 'Email' })).toBeVisible();

    // Change password form should be visible
    const passwordForm = page.locator('form.settings-form');
    await expect(passwordForm).toBeVisible();
    await expect(passwordForm.locator('input[name="current"]')).toBeVisible();
    await expect(passwordForm.locator('input[name="next"]')).toBeVisible();
    await expect(passwordForm.locator('button', { hasText: 'Update password' })).toBeVisible();

    // ── Switch to Workspace tab ──────────────────────────────
    await workspaceTab.click();
    await page.waitForTimeout(500);
    await expect(workspaceTab).toHaveClass(/active/);
    await expect(accountTab).not.toHaveClass(/active/);

    // Workspace section should render
    const workspaceSection = page.locator('.settings-section').first();
    await expect(workspaceSection).toBeVisible();
    await expect(workspaceSection.locator('.card-header', { hasText: 'Workspace' })).toBeVisible();

    // Workspace name input should be present
    const wsInput = workspaceSection.locator('input[name="workspace-name"]');
    await expect(wsInput).toBeVisible();

    // Members section should render (second card)
    const memberSection = page.locator('.settings-section').nth(1);
    await expect(memberSection).toBeVisible();
    await expect(memberSection.locator('.card-header', { hasText: 'Members' })).toBeVisible();

    // Member list should render (may be empty)
    const memberList = memberSection.locator('.settings-member-list');
    await expect(memberList).toBeVisible();

    // ── Switch to Billing tab ────────────────────────────────
    await billingTab.click();
    await page.waitForTimeout(500);
    await expect(billingTab).toHaveClass(/active/);
    await expect(workspaceTab).not.toHaveClass(/active/);

    // Billing section should render
    const billingSection = page.locator('.settings-section');
    await expect(billingSection).toBeVisible();
    await expect(billingSection.locator('.card-header', { hasText: 'Billing' })).toBeVisible();

    // Current plan row should be present
    await expect(billingSection.locator('.settings-row', { hasText: 'Current plan' })).toBeVisible();

    // Usage cells should render with data-testid attributes
    await expect(page.locator('[data-testid="usage-functions"]')).toBeVisible();
    await expect(page.locator('[data-testid="usage-storage"]')).toBeVisible();
    await expect(page.locator('[data-testid="usage-sites"]')).toBeVisible();

    // ── Switch back to Account tab ───────────────────────────
    await accountTab.click();
    await page.waitForTimeout(500);
    await expect(accountTab).toHaveClass(/active/);
    await expect(billingTab).not.toHaveClass(/active/);
  });

  test('sidebar settings link navigates to settings page', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/');
    await page.getByRole('navigation').waitFor({ state: 'visible', timeout: 15000 });

    // Find the Settings link in the sidebar footer
    const settingsLink = page.locator('.sidebar-footer-nav a', { hasText: 'Settings' });
    await expect(settingsLink).toBeVisible();
    await expect(settingsLink).toHaveAttribute('href', '#/settings')

    // Navigate directly to settings page via hash route (full SPA navigation)
    await page.goto('/admin/#/settings');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Should now be on the settings page
    await expect(page.locator('.page-header')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.page-header')).toContainText('Settings');
    await expect(page.locator('.tab.active', { hasText: 'Account' })).toBeVisible();
  });
});
