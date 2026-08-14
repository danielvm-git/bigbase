// story: e78s03
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-deploy-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  // Register a fresh user for the UI tests.
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);

  // Login to obtain the auth token.
  const loginRes = await request.post('/api/auth/login', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(loginRes.status()).toBe(200);
  const body = await loginRes.json();
  authToken = body.token;
  expect(typeof authToken).toBe('string');
});

test.beforeEach(async ({ context }) => {
  // Set the HttpOnly auth cookie so the SPA fetches /api/auth/me successfully.
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

// ═════════════════════════════════════════════════════════════════
// Test 1: Deploy page renders with site list
// ═════════════════════════════════════════════════════════════════
test.describe('Deploy Page', () => {
  test('Deploy page renders with site list or empty state', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/deploy');

    // The route loads site data asynchronously; wait for either stable state.
    await expect(page.locator('.empty-state, .sites-toolbar')).toBeVisible({ timeout: 15000 });

    // Either the empty state is shown with "Create your first site",
    // or the site list / toolbar is rendered.
    const emptyState = page.locator('.empty-state');
    const toolbar = page.locator('.sites-toolbar');

    const emptyCount = await emptyState.count();
    const toolbarCount = await toolbar.count();
    expect(emptyCount + toolbarCount).toBeGreaterThanOrEqual(1);

    // If in empty state, verify the "Create site" button is present.
    if (emptyCount > 0 && await emptyState.isVisible()) {
      await expect(emptyState).toContainText('Create your first site');
      const createBtn = emptyState.getByRole('button', { name: /create site/i });
      await expect(createBtn).toBeVisible();
      await expect(createBtn).toBeEnabled();
    } else {
      // Otherwise the toolbar (filter/search) should be visible.
      await expect(toolbar).toBeVisible();
      // The "Create site" button in the page header should also be visible.
      const createBtn = page.getByRole('button', { name: /create site/i });
      await expect(createBtn).toBeVisible();
    }
  });

  test('Deploy page shows page subtitle', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/deploy');

    // Verify the subtitle text is rendered.
    await expect(page.locator('.page-subtitle')).toContainText(
      /Deploy and host web apps straight from Git/
    );
  });
});

// ═════════════════════════════════════════════════════════════════
// Test 2: Create Site / Deploy New page renders
// ═════════════════════════════════════════════════════════════════
test.describe('Create Site Page', () => {
  test('Create Site / Deploy New page renders', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/deploy/new');

    // Verify the page title.
    await expect(page.locator('h1')).toContainText('Create a new site');

    // Verify the wizard steps indicator is present.
    const wizardSteps = page.locator('.wizard-steps');
    await expect(wizardSteps).toBeVisible();

    // Verify the source selection step is active (step 1).
    await expect(page.locator('.wizard-panel')).toBeVisible();

    // Verify the "Where's your code?" heading.
    await expect(page.locator('h2')).toContainText("Where's your code?");

    // Verify the source choice cards are rendered.
    const choiceCards = page.locator('.choice-card');
    const cardCount = await choiceCards.count();
    expect(cardCount).toBeGreaterThanOrEqual(2);

    // Verify source selection options.
    await expect(choiceCards.first()).toContainText(/connect git/i);
    await expect(choiceCards.nth(1)).toContainText(/existing bigbase repo/i);

    // Verify navigation controls are present.
    const cancelBtn = page.getByRole('button', { name: /cancel/i }).first();
    await expect(cancelBtn).toBeVisible();

    const continueBtn = page.getByRole('button', { name: /continue/i });
    await expect(continueBtn).toBeVisible();
    // Continue should be disabled when no source is selected.
    await expect(continueBtn).toBeDisabled();
  });

  test('Create site page has breadcrumb navigation', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/deploy/new');

    // Verify the breadcrumb links back to Sites.
    const breadcrumb = page.locator('.breadcrumb');
    await expect(breadcrumb).toBeVisible();
    await expect(breadcrumb).toContainText('Sites');
    await expect(breadcrumb).toContainText('Create site');
  });
});

// ═════════════════════════════════════════════════════════════════
// Test 3: Site detail page tabs render
// ═════════════════════════════════════════════════════════════════
test.describe('Site Detail Page', () => {
  let siteId: string;
  let repoId: string;

  test.beforeAll(async ({ request }) => {
    // Create a git repo and site so we have something to navigate to.
    const ts = Date.now();
    const repoName = `e2e-deploy-ui-${ts}`;

    const repoRes = await request.post('/api/git/repos', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName },
    });
    expect(repoRes.status()).toBe(201);
    const repoBody = await repoRes.json();
    repoId = repoBody.id as string;

    const siteRes = await request.post('/api/sites', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName, git_repo_id: repoId, production_branch: 'main', root_path: './' },
    });
    expect(siteRes.status()).toBe(201);
    const siteBody = await siteRes.json();
    siteId = siteBody.id as string;
  });

  test.afterAll(async ({ request }) => {
    // Clean up the created site and repo.
    if (siteId) {
      await request.delete(`/api/sites/${siteId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
    }
    if (repoId) {
      await request.delete(`/api/git/repos/${repoId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
    }
  });

  test('Site detail page tabs render', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // The page should load with the site name as title.
    await expect(page.locator('h1')).toBeVisible();

    // Verify the tab bar renders with expected tabs.
    const tabs = page.locator('.tabs');
    await expect(tabs).toBeVisible();

    // Verify each expected tab label is present.
    const expectedTabs = [
      'Deployments',
      'Build Logs',
      'Request Logs',
      'Env Vars',
      'Domains',
      'Deploy Keys',
      'Cache',
      'Manifest',
    ];

    for (const label of expectedTabs) {
      const tab = page.locator('.tab', { hasText: label });
      await expect(tab).toBeVisible();
    }

    // The first tab (Deployments) should be active by default.
    await expect(page.locator('.tab.active', { hasText: 'Deployments' })).toBeVisible();
  });

  test('Site detail tab switching shows different content', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // Click on the "Manifest" tab.
    await page.locator('.tab', { hasText: 'Manifest' }).click();
    // Verify manifest section title appears.
    await expect(page.locator('h2.section-title')).toContainText(/App Manifest/i);

    // Click on the "Env Vars" tab.
    await page.locator('.tab', { hasText: 'Env Vars' }).click();
    // The env vars heading should render.
    const envVarsHeading = page.locator('h2.section-title').filter({ hasText: /Environment Variables/i });
    await expect(envVarsHeading.first()).toBeVisible();

    // Click on the "Cache" tab.
    await page.locator('.tab', { hasText: 'Cache' }).click();
    // The cache heading should render.
    const cacheHeading = page.locator('h2.section-title').filter({ hasText: /cache/i });
    await expect(cacheHeading.first()).toBeVisible();

    // Click on the "Build Logs" tab.
    await page.locator('.tab', { hasText: 'Build Logs' }).click();
    // The TerminalLogViewer or its loading/error state should render.
    await expect(page.locator('[data-testid="terminal-log-viewer-loading"], [data-testid="terminal-log-viewer"], [data-testid="terminal-log-viewer-error"]').first()).toBeVisible();

    // Click on the "Domains" tab.
    await page.locator('.tab', { hasText: 'Domains' }).click();
    // The domains tab renders an "Add Custom Domain" heading.
    await expect(page.locator('h3').filter({ hasText: /add custom domain/i }).first()).toBeVisible();

    // Click on the "Deploy Keys" tab.
    await page.locator('.tab', { hasText: 'Deploy Keys' }).click();
    // The deploy keys tab renders a "Deploy Keys" heading.
    await expect(page.locator('h3').filter({ hasText: /deploy keys/i }).first()).toBeVisible();
  });

  test('Site detail page shows configuration card', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // The page should show at least one card with configuration details.
    const cards = page.locator('.card');
    await expect(cards.first()).toBeVisible();

    // The status and configuration cards should be rendered.
    await expect(page.locator('.card').filter({ hasText: /status/i }).first()).toBeVisible();
    await expect(page.locator('.card').filter({ hasText: /configuration/i }).first()).toBeVisible();
  });

  test('Site detail page shows deployment history heading', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // Deployment History heading should appear in the default deployments tab.
    await expect(page.locator('h2.section-title').filter({ hasText: /Deployment History/i })).toBeVisible();
  });
});

// ═════════════════════════════════════════════════════════════════
// Test 4: Deploy actions UI
// ═════════════════════════════════════════════════════════════════
test.describe('Deploy Actions UI', () => {
  let siteId: string;
  let repoId: string;

  test.beforeAll(async ({ request }) => {
    // Create a git repo and site for destructive action tests.
    const ts = Date.now();
    const repoName = `e2e-actions-ui-${ts}`;

    const repoRes = await request.post('/api/git/repos', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName },
    });
    expect(repoRes.status()).toBe(201);
    const repoBody = await repoRes.json();
    repoId = repoBody.id as string;

    const siteRes = await request.post('/api/sites', {
      headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
      data: { name: repoName, git_repo_id: repoId, production_branch: 'main', root_path: './' },
    });
    expect(siteRes.status()).toBe(201);
    const siteBody = await siteRes.json();
    siteId = siteBody.id as string;
  });

  test.afterAll(async ({ request }) => {
    // Clean up (delete test may have already removed the site — that's OK).
    if (siteId) {
      await request.delete(`/api/sites/${siteId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      }).catch(() => {});
    }
    if (repoId) {
      await request.delete(`/api/git/repos/${repoId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
    }
  });

  test('Redeploy and Refresh buttons render on site detail', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // The page header should contain a "Redeploy" button.
    // Redeploy button should be in the page header area
    const redeployBtn = page.locator('.page-header').getByRole('button', { name: /redeploy/i });
    await expect(redeployBtn).toBeVisible();
    await expect(redeployBtn).toBeEnabled();

    // A "Refresh" button should also be present.
    const refreshBtn = page.locator('.page-header').getByRole('button', { name: /refresh/i });
    await expect(refreshBtn).toBeVisible();
  });

  test('Site detail page shows Danger Zone with delete button', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // Scroll down to the Danger Zone card.
    const dangerZone = page.locator('.card').filter({ hasText: /Danger Zone/i });
    await expect(dangerZone).toBeVisible();

    // Danger Zone description should be present.
    await expect(dangerZone).toContainText(/permanently delete/i);
    await expect(dangerZone).toContainText(/cannot be undone/i);

    // The delete button for the site should be visible.
    const deleteBtn = dangerZone.getByRole('button', { name: /delete/i });
    await expect(deleteBtn).toBeVisible();
  });

  test('Delete site triggers confirmation dialog', async ({ page }) => {
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // The Danger Zone card should be present with delete button.
    const dangerZone = page.locator('.card').filter({ hasText: /Danger Zone/i });
    await expect(dangerZone).toBeVisible();

    // Set up dialog handler to accept the browser's native confirm() dialog
    page.on('dialog', async dialog => {
      await dialog.accept();
    });

    // Click the delete button in the Danger Zone.
    await dangerZone.getByRole('button', { name: /delete/i }).click();

    // The page navigates back to deploy after delete (site uses window.confirm + API).
    await page.waitForURL('**/deploy', { timeout: 10000 });
    await expect(page.locator('h1')).toContainText('Sites');
  });
});
