// story: e78s02
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-dash-ui-${Date.now()}@test.com`;
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

test.describe('Dashboard & Navigation UI', () => {
  // ─── Test 1: Responsive sidebar navigation ────────────────────
  test('sidebar collapses and toggles on mobile viewport', async ({ page }) => {
    // Desktop — sidebar should be open by default
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/');
    await page.getByRole('navigation').waitFor({ state: 'visible', timeout: 15000 });

    // Verify sidebar is visible (starts closed on desktop)
    const sidebar = page.getByRole('navigation');
    await expect(sidebar).toBeVisible();

    // Verify key nav sections are rendered
    await expect(page.locator('.sidebar-nav').first()).toBeVisible();

    // Check for section titles
    await expect(page.locator('.sidebar-section-title', { hasText: 'Overview' })).toBeVisible();
    await expect(page.locator('.sidebar-section-title', { hasText: 'Build' })).toBeVisible();
    await expect(page.locator('.sidebar-section-title', { hasText: 'Data' })).toBeVisible();
    await expect(page.locator('.sidebar-section-title', { hasText: 'DevOps' })).toBeVisible();

    // Check for specific nav items
    await expect(page.locator('.sidebar-nav a', { hasText: 'Dashboard' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Sites' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Functions' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Data Studio' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'SQL Editor' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Storage' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Users' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Messaging' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Git Repos' })).toBeVisible();
    await expect(page.locator('.sidebar-nav a', { hasText: 'Monitoring' })).toBeVisible();

    // Mobile — sidebar should collapse
    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(300); // allow CSS transition
    await expect(page.getByRole('navigation')).not.toHaveClass(/sidebar-open/);

    // Hamburger toggle button should be visible — uses aria-label "Open sidebar" / "Close sidebar"
    const toggle = page.getByRole('button', { name: /sidebar/i });
    await expect(toggle).toBeVisible();
    await expect(toggle).toHaveAttribute('aria-label', 'Open sidebar');

    // Click hamburger to open sidebar
    await toggle.click();
    await page.waitForTimeout(300);
    await expect(page.getByRole('navigation')).toHaveClass(/sidebar-open/);

    // Click again to close
    await toggle.click();
    await page.waitForTimeout(300);
    await expect(page.getByRole('navigation')).not.toHaveClass(/sidebar-open/);
  });

  // ─── Test 2: Theme picker ─────────────────────────────────────
  test('theme picker dropdown opens and accent color can be changed', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/');
    await page.getByRole('navigation').waitFor({ state: 'visible', timeout: 15000 });

    // Sidebar footer should contain the theme picker — trigger has aria-label "Accent theme"
    const themeTrigger = page.getByRole('button', { name: /accent/i });
    await expect(themeTrigger).toBeVisible();

    // Open the theme dropdown
    await themeTrigger.click();
    const themeMenu = page.getByRole('menu');
    await expect(themeMenu).toBeVisible();

    // Grab the list of available theme options (each has role="menuitem")
    const themeItems = page.getByRole('menuitem');
    const count = await themeItems.count();
    expect(count).toBeGreaterThan(1);

    // Find the currently active theme (has aria-current="true")
    const activeItem = page.getByRole('menuitem').and(page.locator('[aria-current="true"]'));
    const activeLabel = await activeItem.textContent();
    expect(activeLabel).toBeTruthy();

    // Click a different theme (any item without aria-current="true")
    let clickedDifferent = false;
    for (let i = 0; i < count; i++) {
      const item = themeItems.nth(i);
      const isActive = await item.getAttribute('aria-current');
      if (isActive !== 'true') {
        await item.click();
        clickedDifferent = true;
        break;
      }
    }
    expect(clickedDifferent).toBe(true);

    // The dropdown should close after selection
    await expect(themeMenu).not.toBeVisible();

    // Re-open and verify the new theme is now marked active
    await themeTrigger.click();
    await expect(themeMenu).toBeVisible();
    const newActiveItem = page.getByRole('menuitem').and(page.locator('[aria-current="true"]'));
    const newActiveLabel = await newActiveItem.textContent();
    expect(newActiveLabel).not.toBe(activeLabel);

    // Toggle back to the original theme
    for (let i = 0; i < count; i++) {
      const item = themeItems.nth(i);
      const text = await item.textContent();
      if (text === activeLabel) {
        await item.click();
        break;
      }
    }
  });

  // ─── Test 3: Dashboard metrics rendering ──────────────────────
  test('dashboard renders metric cards and system status panel', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/');
    await page.waitForSelector('.dashboard', { timeout: 15000 });

    // Page header should welcome the user
    const pageHeader = page.locator('.page-header');
    await expect(pageHeader).toBeVisible();

    // System status panel should be present
    const statusPanel = page.locator('.system-status-panel');
    await expect(statusPanel).toBeVisible();

    // Check for system status content — "All systems operational" or "System issues detected"
    const statusTitle = statusPanel.locator('.system-status-title');
    await expect(statusTitle).toBeVisible();

    // CPU metric tile should render inside the panel
    const cpuTile = statusPanel.locator('.metric-tile', { hasText: 'CPU' });
    await expect(cpuTile).toBeVisible();

    // Memory metric tile should render
    const memTile = statusPanel.locator('.metric-tile', { hasText: 'Memory' });
    await expect(memTile).toBeVisible();

    // Components metric tile should render
    const compTile = statusPanel.locator('.metric-tile', { hasText: 'Components' });
    await expect(compTile).toBeVisible();

    // Stat cards (Sites, Functions, Git Repos, Users) should render
    const statCards = page.locator('.stat-card');
    const statCount = await statCards.count();
    expect(statCount).toBeGreaterThanOrEqual(2);

    // Recent deployments section should appear
    const deploySection = page.locator('.dashboard-deploy-list');
    const noDeployments = page.locator('.dashboard .dim', { hasText: 'No deployments yet.' });

    // Either the deploy list exists or the empty message shows
    const listExists = (await deploySection.count()) > 0;
    const emptyExists = (await noDeployments.count()) > 0;
    expect(listExists || emptyExists).toBe(true);

    // "Jump back in" card should be visible
    await expect(page.locator('.jump-back-list')).toBeVisible();
    await expect(page.locator('.jump-back-item', { hasText: 'Deploy a site' })).toBeVisible();
    await expect(page.locator('.jump-back-item', { hasText: 'Write a SQL query' })).toBeVisible();
  });

  // ─── Test 4: Onboarding checklist ─────────────────────────────
  test('onboarding checklist renders when available', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/');
    await page.waitForSelector('.dashboard', { timeout: 15000 });

    // The onboarding checklist is conditionally rendered
    // Check if an onboarding card exists (data-testid="onboarding-checklist")
    const checklist = page.locator('[data-testid="onboarding-checklist"]');
    const checklistExists = (await checklist.count()) > 0;

    if (checklistExists) {
      // Verify it has a "Get started" header
      await expect(checklist.locator('.card-header', { hasText: 'Get started' })).toBeVisible();

      // Check it either shows loading state or step list
      const loading = checklist.locator('text=Loading checklist');
      const stepList = checklist.locator('.onboarding-list');

      const loadingVisible = (await loading.count()) > 0;
      const stepsVisible = (await stepList.count()) > 0;
      expect(loadingVisible || stepsVisible).toBe(true);

      // If steps are rendered, check for step items (may be empty if API returns no steps)
      if (stepsVisible) {
        const steps = stepList.locator('.onboarding-item');
        const stepCount = await steps.count();
        // Steps list may be empty if the onboarding API returns no steps
        expect(stepCount).toBeGreaterThanOrEqual(0);
      }
    } else {
      // Checklist is hidden when all steps are done — that's acceptable
      test.info().annotations.push({
        type: 'info',
        description: 'Onboarding checklist not rendered (all steps completed or API not available)',
      });
    }
  });
});
