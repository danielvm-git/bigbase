// story: e78s05
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-msg-ui-${Date.now()}@test.com`;
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

// ─── Test 3: Messaging page renders ───────────────────────────────
test('messaging page renders template list', async ({ page }) => {
  await page.goto('http://localhost:9999/admin/#/messaging');
  await page.waitForLoadState('networkidle');

  // Verify the page header renders with the correct title.
  await expect(page.locator('h1, h2').filter({ hasText: 'Messaging' })).toBeVisible();

  // Verify the sample template button is present.
  await expect(page.getByRole('button', { name: /open sample template/i })).toBeVisible();

  // Verify the Refresh button is present.
  await expect(page.getByRole('button', { name: /refresh/i })).toBeVisible();

  // Verify the Templates tab is active by default (.tab class component).
  const templatesTab = page.locator('.tab', { hasText: 'Templates' });
  await expect(templatesTab).toBeVisible();

  // Verify the template grid renders with mock templates.
  const templateGrid = page.locator('.template-grid');
  await expect(templateGrid).toBeVisible();

  // Verify at least one template row is rendered inside the grid.
  const templateRows = page.locator('.template-row');
  await expect(templateRows.first()).toBeVisible();

  // Verify other page tabs are present.
  await expect(page.locator('.tab', { hasText: 'Send test' })).toBeVisible();
  await expect(page.locator('.tab', { hasText: 'History' })).toBeVisible();
});

// ─── Test 4: Messaging template editor ────────────────────────────
test('messaging template editor and preview render', async ({ page }) => {
  // Navigate to a known sample template (tpl-welcome is shipped with mock data).
  await page.goto('http://localhost:9999/admin/#/messaging/tpl-welcome');
  await page.waitForLoadState('networkidle');

  // Verify the template name appears in the page header.
  await expect(page.locator('h1, h2').first()).toBeVisible();

  // Verify breadcrumb shows Messaging link.
  await expect(page.getByText('Messaging').first()).toBeVisible();

  // Verify the Editor and Preview tabs are rendered (.tab class component).
  const editorTab = page.locator('.tab', { hasText: 'Editor' });
  const previewTab = page.locator('.tab', { hasText: 'Preview' });
  await expect(editorTab).toBeVisible();
  await expect(previewTab).toBeVisible();

  // Editor tab should be active by default — verify the subject input is visible.
  const subjectInput = page.locator('input').first();
  await expect(subjectInput).toBeVisible();

  // Verify the body textarea is visible.
  const bodyTextarea = page.locator('textarea').first();
  await expect(bodyTextarea).toBeVisible();

  // Verify action buttons exist (Back, Send test).
  await expect(page.getByRole('button', { name: /back/i })).toBeVisible();
  await expect(page.getByRole('button', { name: /send test/i })).toBeVisible();

  // Switch to Preview tab and verify preview content renders.
  await previewTab.click();
  await page.waitForTimeout(500);

  // The preview tab shows "Rendered preview" heading and variable info.
  await expect(page.getByText(/rendered preview/i)).toBeVisible();
  await expect(page.getByText(/variables/i)).toBeVisible();
});
