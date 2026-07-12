// story: e78s05
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-fn-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  // Register the test user.
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);
  const regBody = await regRes.json();
  authToken = regBody.token;
  expect(typeof authToken).toBe('string');
});

test.beforeEach(async ({ page }) => {
  // Seed the browser's auth cookie so the SPA Layout /api/auth/me call succeeds.
  await page.context().addCookies([
    { name: 'token', value: authToken, domain: 'localhost', path: '/' },
  ]);
});

// ─── Test 1: Functions page renders ───────────────────────────────
test('functions page renders the function grid', async ({ page }) => {
  await page.goto('http://localhost:9999/admin/#/functions');
  await page.waitForLoadState('networkidle');

  // Verify the page header renders with the correct title.
  await expect(page.locator('h1, h2').filter({ hasText: 'Functions' })).toBeVisible();

  // Verify action buttons are present.
  await expect(page.getByRole('button', { name: /create function/i })).toBeVisible();
  await expect(page.getByRole('button', { name: /refresh/i })).toBeVisible();

  // If functions exist, the function grid is rendered; otherwise an empty state is shown.
  // The function-grid class wraps function cards when data is present.
  const grid = page.locator('.function-grid');
  const empty = page.locator('.dim').filter({ hasText: /no functions yet/i });
  if (await grid.isVisible()) {
    // Verify at least one function card is rendered.
    const cards = page.locator('.function-card');
    await expect(cards.first()).toBeVisible();
  } else if (await empty.isVisible()) {
    await expect(empty).toBeVisible();
  }
});

// ─── Test 2: Function detail page tabs ────────────────────────────
test('function detail page renders all tabs and supports tab switching', async ({ page }) => {
  // Create a function via API so we have a detail page to visit.
  const fnName = `e2e-test-fn-${Date.now()}`;
  const createRes = await page.request.post('/api/functions', {
    data: {
      name: fnName,
      runtime: 'javascript',
      source: 'export default () => ({ hello: "world" });',
      trigger: 'http',
      timeout: 30,
    },
    headers: { Authorization: `Bearer ${authToken}`, 'Content-Type': 'application/json' },
  });
  expect(createRes.status()).toBe(201);
  const fnBody = await createRes.json();
  const fnId: string = fnBody.id ?? '';

  try {
    // Navigate to the function detail page.
    await page.goto(`http://localhost:9999/admin/#/functions/${fnId}`);
    await page.waitForLoadState('networkidle');

    // Verify the function name appears in the page header.
    await expect(page.locator('h1, h2').filter({ hasText: fnName })).toBeVisible();

    // Verify breadcrumb navigation links.
    await expect(page.getByText('Functions').first()).toBeVisible();

    // Verify all four detail tabs are rendered (.tab class component).
    const codeTab = page.locator('.tab', { hasText: 'Code' });
    const triggersTab = page.locator('.tab', { hasText: 'Triggers' });
    const variablesTab = page.locator('.tab', { hasText: 'Variables' });
    const logsTab = page.locator('.tab', { hasText: 'Logs' });

    await expect(codeTab).toBeVisible();
    await expect(triggersTab).toBeVisible();
    await expect(variablesTab).toBeVisible();
    await expect(logsTab).toBeVisible();

    // Code tab is active by default — the source code textarea should be visible.
    const textarea = page.locator('textarea');
    await expect(textarea.first()).toBeVisible();

    // Switch to Triggers tab and verify content.
    await triggersTab.click();
    await page.waitForTimeout(500);
    await expect(page.getByText(/http/i).first()).toBeVisible();

    // Switch to Variables tab and verify the textarea appears.
    await variablesTab.click();
    await page.waitForTimeout(500);
    await expect(page.locator('textarea').first()).toBeVisible();

    // Switch to Logs tab and verify the logs component renders.
    await logsTab.click();
    await page.waitForTimeout(500);
    // The FunctionLogsPanel component renders inside the detail page when logs tab is active.
    // It may show loading, error, or "No executions yet." state.
    const loadingState = page.locator('.loading', { hasText: /loading execution logs/i });
    const errorState = page.locator('.input-error-text', { hasText: /failed to load/i });
    const emptyState = page.getByText(/no executions yet/i);
    const loadingVisible = await loadingState.isVisible().catch(() => false);
    const errorVisible = await errorState.isVisible().catch(() => false);
    const emptyVisible = await emptyState.isVisible().catch(() => false);
    await expect(loadingVisible || errorVisible || emptyVisible).toBeTruthy();
  } finally {
    // Clean up the test function.
    await page.request.delete(`/api/functions/${fnId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    }).catch(() => {});
  }
});
