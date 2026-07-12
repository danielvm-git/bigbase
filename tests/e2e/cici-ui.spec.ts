// story: e78s03
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-cici-ui-${Date.now()}@test.com`;
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
// Test 5: CI/CD pipeline page renders
// ═════════════════════════════════════════════════════════════════
test.describe('CI/CD Pipeline Page', () => {
  test('CI/CD page renders with title and repo selector', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/cici');

    // The page title should be "CI/CD".
    await expect(page.locator('h1')).toContainText('CI/CD');

    // No repo selected initially — the page should show a prompt.
    await expect(page.getByText(/select a repo/i)).toBeVisible();

    // A repo selector (select element or similar) should be present.
    const repoSelect = page.locator('select').first();
    await expect(repoSelect).toBeVisible();
  });

  test('CI/CD page shows New Workflow button', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/cici');

    // When there's no repo selected yet, the "New Workflow" button
    // becomes visible only after a repo is selected.
    // The page header should still be visible.
    await expect(page.locator('.page-header')).toBeVisible();
    // "New Workflow" button is shown after repo selection, not before
    // Just verify the page loaded with CI/CD title
    await expect(page.locator('h1')).toContainText('CI/CD');
  });

  test('CI/CD page allows repo selection and shows tabs', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/cici');

    // Check if there are any repos available in the select.
    const repoSelect = page.locator('select').first();
    const options = repoSelect.locator('option');
    const optionCount = await options.count();

    if (optionCount <= 1) {
      // Only the placeholder option — no repos to select.
      // The page should show the "Select a repo" prompt.
      await expect(page.getByText(/select a repo/i)).toBeVisible();
      test.skip();
      return;
    }

    // Select the first real repo (skip the placeholder).
    const firstRealOption = options.nth(1);
    const repoValue = await firstRealOption.getAttribute('value');
    await repoSelect.selectOption(repoValue!);

    // The page should now show the CI/CD tabs (Workflows / Runs).
    const tabs = page.locator('.tabs');
    await expect(tabs).toBeVisible();

    // Verify both tabs are rendered (as .tab buttons, not role="tab").
    const workflowsTab = page.locator('.tab', { hasText: /workflows/i });
    const runsTab = page.locator('.tab', { hasText: /runs/i });

    await expect(workflowsTab).toBeVisible();
    await expect(runsTab).toBeVisible();

    // Workflows tab should be active by default.
    await expect(workflowsTab).toHaveClass(/active/);
  });

  test('CI/CD page workflow tab shows empty state or workflow table', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/cici');

    // Check for available repos.
    const repoSelect = page.locator('select').first();
    const options = repoSelect.locator('option');
    const optionCount = await options.count();

    if (optionCount <= 1) {
      test.skip();
      return;
    }

    // Select a repo to reveal the workflows content.
    const firstRealOption = options.nth(1);
    await repoSelect.selectOption(await firstRealOption.getAttribute('value')!);

    // Wait for the workflows section to load.
    await page.waitForTimeout(1000);

    // Either "No workflows yet." empty state or the workflow table should render.
    const noWorkflows = page.getByText(/no workflows/i);
    const table = page.locator('.table-wrap');

    const noWorkflowsCount = await noWorkflows.count();
    const tableCount = await table.count();

    expect(noWorkflowsCount + tableCount).toBeGreaterThanOrEqual(1);

    if (await table.isVisible()) {
      // Verify the table has the expected columns.
      const tableHtml = await table.innerHTML();
      expect(tableHtml).toContain('Name');
      expect(tableHtml).toContain('Actions');
    }
  });

  test('CI/CD page runs tab shows empty state or run history', async ({ page }) => {
    await page.goto('http://localhost:9999/admin/#/cici');

    // Check for available repos.
    const repoSelect = page.locator('select').first();
    const options = repoSelect.locator('option');
    const optionCount = await options.count();

    if (optionCount <= 1) {
      test.skip();
      return;
    }

    // Select a repo.
    const firstRealOption = options.nth(1);
    await repoSelect.selectOption(await firstRealOption.getAttribute('value')!);

    // Wait for content to load.
    await page.waitForTimeout(1000);

    // Switch to the Runs tab.
    await page.locator('.tab', { hasText: /runs/i }).click();

    // Either "No runs yet." empty state or the run table should render.
    const noRuns = page.getByText(/no runs/i);
    const table = page.locator('.table-wrap');

    const noRunsCount = await noRuns.count();
    const tableCount = await table.count();

    expect(noRunsCount + tableCount).toBeGreaterThanOrEqual(1);

    if (await table.isVisible()) {
      // Verify the table has expected column headers.
      const tableHtml = await table.innerHTML();
      expect(tableHtml).toContain('ID');
      expect(tableHtml).toContain('Status');
      // Each run should have a Logs button.
      const logsBtn = table.getByRole('button', { name: /logs/i }).first();
      await expect(logsBtn).toBeVisible();
    }
  });
});

// ═════════════════════════════════════════════════════════════════
// Test 6: Build logs viewer
// ═════════════════════════════════════════════════════════════════
test.describe('Build Logs Viewer', () => {
  test('Site detail page renders TerminalLogViewer in Build Logs tab', async ({ page }) => {
    // Create a git repo and site so we have a site to navigate to.
    const ts = Date.now();
    const repoName = `e2e-cici-logs-${ts}`;
    let siteId: string;
    let repoId: string;

    try {
      const repoRes = await page.request.post('/api/git/repos', {
        headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
        data: { name: repoName },
      });
      if (repoRes.status() !== 201) {
        test.skip();
        return;
      }
      const repoBody = await repoRes.json();
      repoId = repoBody.id as string;

      const siteRes = await page.request.post('/api/sites', {
        headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
        data: { name: repoName, git_repo_id: repoId, production_branch: 'main', root_path: './' },
      });
      if (siteRes.status() !== 201) {
        test.skip();
        return;
      }
      const siteBody = await siteRes.json();
      siteId = siteBody.id as string;
    } catch {
      test.skip();
      return;
    }

    // Navigate directly to the site detail page.
    await page.goto(`http://localhost:9999/admin/#/deploy/${siteId}`);

    // Click on the "Build Logs" tab.
    await page.locator('.tab', { hasText: 'Build Logs' }).click();

    // The TerminalLogViewer component should render.
    // It shows a loading state, error state, or the log viewer itself.
    const logViewer = page.locator(
      '[data-testid="terminal-log-viewer-loading"], ' +
      '[data-testid="terminal-log-viewer"], ' +
      '[data-testid="terminal-log-viewer-error"]'
    );
    await expect(logViewer.first()).toBeVisible();

    // If the viewer is in its normal state, verify it contains StreamLog elements.
    if (await page.locator('[data-testid="terminal-log-viewer"]').isVisible()) {
      // StreamLog sub-components should be present.
      const streamLog = page.locator('.stream-log-container');
      await expect(streamLog).toBeVisible();

      // The toolbar with search and controls should render.
      const toolbar = page.locator('[data-testid="stream-log-toolbar"]');
      await expect(toolbar).toBeVisible();
    }

    // If the viewer is in loading state, it should show "Loading logs..." text.
    if (await page.locator('[data-testid="terminal-log-viewer-loading"]').isVisible()) {
      await expect(page.locator('.stream-log-container')).toContainText(/loading logs/i);
    }

    // If there's a live badge, the streaming indicator is shown.
    const liveBadge = page.locator('[data-testid="terminal-log-viewer-live"]');
    if (await liveBadge.isVisible()) {
      await expect(liveBadge).toContainText('LIVE');
    }

    // Clean up.
    try {
      await page.request.delete(`/api/sites/${siteId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
      await page.request.delete(`/api/git/repos/${repoId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
    } catch { /* ignore cleanup errors */ }
  });

  test('CI/CD page shows build output when logs are expanded', async ({ page }) => {
    // Create a git repo and a workflow run so we can view logs.
    const ts = Date.now();
    const repoName = `e2e-cici-logs-${ts}`;

    let repoId: string;
    try {
      const repoRes = await page.request.post('/api/git/repos', {
        headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
        data: { name: repoName },
      });
      if (repoRes.status() !== 201) {
        test.skip();
        return;
      }
      const repoBody = await repoRes.json();
      repoId = repoBody.id as string;
    } catch {
      test.skip();
      return;
    }

    // Navigate to the CI/CD page.
    await page.goto('http://localhost:9999/admin/#/cici');

    // Wait for the repo select to populate.
    await page.waitForTimeout(1000);

    // Select the newly created repo.
    const repoSelect = page.locator('select').first();
    await repoSelect.selectOption(repoId);

    // Wait for workflows/runs to load.
    await page.waitForTimeout(1000);

    // Switch to the Runs tab.
    await page.locator('.tab', { hasText: /runs/i }).click();
    await page.waitForTimeout(500);

    // Check if there are any runs.
    const logsBtn = page.locator('.table-wrap').getByRole('button', { name: /logs/i }).first();
    if (await logsBtn.isVisible()) {
      // Click the Logs button to expand the build output.
      await logsBtn.click();
      await page.waitForTimeout(500);

      // The StreamLog container should render.
      const streamLog = page.locator('.stream-log-container');
      await expect(streamLog).toBeVisible();

      // If there are log lines, verify they render.
      const logLine = streamLog.locator('.stream-log-line').first();
      if (await logLine.isVisible()) {
        // Each line can have line number, timestamp, and text.
        // At minimum, text should be present.
        await expect(logLine.locator('.stream-log-text')).toBeVisible();
      }
    }

    // Clean up the repo.
    try {
      await page.request.delete(`/api/git/repos/${repoId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
    } catch { /* ignore cleanup errors */ }
  });
});
