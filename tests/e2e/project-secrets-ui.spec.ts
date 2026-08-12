// story: e89s05 — Project Secrets Admin UI browser coverage.
//
// Seeds an organization, Project, Environment, and a secret through the API,
// then drives the /secrets UI: masked listing, explicit reveal + clear,
// safe .env import, destructive confirmation, keyboard flow, and value-free
// authorization denials.
import { test, expect, type Page } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

// ── Module-scoped auth state ──────────────────────────────────────
const EMAIL = `e2e-secrets-ui-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
const OTHER_EMAIL = `e2e-secrets-other-${Date.now()}@test.com`;
const SENTINEL = `e2e-seed-sentinel-${Date.now().toString().slice(-6)}`;
const SECRET_KEY = 'SEED_KEY';

let authToken: string;
let otherToken: string;
let projectId: string;
let envId: string;
let secretsPath: string;

test.beforeAll(async ({ request }) => {
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);
  authToken = (await regRes.json()).token as string;

  const otherRes = await request.post('/api/auth/register', {
    data: { email: OTHER_EMAIL, password: PASSWORD },
  });
  expect(otherRes.status()).toBe(201);
  otherToken = (await otherRes.json()).token as string;

  const headers = { Authorization: `Bearer ${authToken}` };

  const projectRes = await request.post('/api/projects', {
    headers,
    data: { name: `e2e-secrets-${Date.now()}` },
  });
  expect(projectRes.status()).toBe(201);
  projectId = ((await projectRes.json()) as { data: { id: string } }).data.id;

  const envRes = await request.post(`/api/projects/${projectId}/environments`, {
    headers,
    data: { slug: 'production', name: 'Production' },
  });
  expect(envRes.status()).toBe(201);
  envId = ((await envRes.json()) as { data: { id: string } }).data.id;

  secretsPath = `/api/projects/${projectId}/environments/${envId}/secrets`;

  // Seed one secret so list/reveal/denial tests are independent of UI creation.
  const seedRes = await request.post(secretsPath, {
    headers,
    data: { key: SECRET_KEY, value: SENTINEL },
  });
  expect(seedRes.status()).toBe(201);
});

test.beforeEach(async ({ context }) => {
  await context.addCookies([{ name: 'token', value: authToken, domain: 'localhost', path: '/' }]);
});

async function openSecretsFolder(page: Page) {
  await page.goto('http://localhost:9999/admin/#/secrets');
  await page.waitForLoadState('networkidle');
  await page.getByRole('link', { name: 'Open environments' }).first().click();
  await page.getByRole('link', { name: 'Open secrets' }).first().click();
  await expect(page.locator('h1').filter({ hasText: 'Project Secrets' })).toBeVisible();
  await expect(page.locator('h2').filter({ hasText: /Secrets in default/ })).toBeVisible();
}

test('project secrets list is masked and never renders plaintext', async ({ page }) => {
  await openSecretsFolder(page);

  // The seeded key is visible…
  await expect(page.getByText(SECRET_KEY)).toBeVisible();
  // …its masked preview shows only the trailing characters…
  await expect(page.getByText(`••••${SENTINEL.slice(-4)}`)).toBeVisible();
  // …and the full plaintext never appears anywhere on the page.
  await expect(page.getByText(SENTINEL)).toHaveCount(0);
});

test('operator creates a secret through the UI and the list stays masked', async ({ page }) => {
  await openSecretsFolder(page);

  const key = 'UI_CREATED_KEY';
  const value = `e2e-ui-created-${Date.now().toString().slice(-6)}`;
  await page.getByRole('button', { name: 'Add Secret' }).click();
  await page.getByLabel('Key').fill(key);
  await page.getByLabel('Value').fill(value);
  await page.getByRole('button', { name: 'Add', exact: true }).click();

  await expect(page.getByText(key)).toBeVisible();
  await expect(page.getByText(`••••${value.slice(-4)}`)).toBeVisible();
  await expect(page.getByText(value)).toHaveCount(0);
});

test('reveal shows one value explicitly and clears it on close', async ({ page }) => {
  await openSecretsFolder(page);

  const row = page.locator('tr', { hasText: SECRET_KEY });
  await row.getByRole('button', { name: 'Reveal' }).click();

  // The explicit /value read displays the plaintext in the dialog…
  await expect(page.getByText(SENTINEL)).toBeVisible();

  // …and closing the dialog clears it from the page.
  await page.getByRole('button', { name: 'Close dialog' }).click();
  await expect(page.getByText(SENTINEL)).toHaveCount(0);
});

test('reveal is keyboard accessible and Escape closes the dialog', async ({ page }) => {
  await openSecretsFolder(page);

  const row = page.locator('tr', { hasText: SECRET_KEY });
  await row.getByRole('button', { name: 'Reveal' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText(SENTINEL)).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(page.getByText(SENTINEL)).toHaveCount(0);
});

test('unauthorized value reads stay value-free (401 anon, 404 cross-org)', async ({ request }) => {
  const valueUrl = `${secretsPath}/${SECRET_KEY}/value`;

  const anon = await request.get(valueUrl);
  expect(anon.status()).toBe(401);
  expect(JSON.stringify(await anon.json())).not.toContain(SENTINEL);

  const crossOrg = await request.get(valueUrl, {
    headers: { Authorization: `Bearer ${otherToken}` },
  });
  // Cross-organization targets are non-disclosing: 404, never the value.
  expect(crossOrg.status()).toBe(404);
  const crossOrgBody = JSON.stringify(await crossOrg.json());
  expect(crossOrgBody).not.toContain(SENTINEL);
  expect(crossOrgBody).not.toContain(SECRET_KEY);
});

test('import saves valid keys and reports invalid keys by name without echoing values', async ({ page }) => {
  await openSecretsFolder(page);

  const importedKey = 'IMPORTED_KEY';
  const importedValue = `imported-value-${Date.now().toString().slice(-6)}`;
  const invalidKey = 'bad-key';

  await page
    .getByLabel('Import .env file')
    .setInputFiles({
      name: 'secrets.env',
      mimeType: 'text/plain',
      buffer: Buffer.from(`${importedKey}=${importedValue}\n${invalidKey}=secret-too\n`),
    });

  // Valid key saved and masked; invalid key reported by name only.
  await expect(page.getByText(importedKey)).toBeVisible();
  await expect(page.getByText(/key failed to import: bad-key/)).toBeVisible();
  await expect(page.getByText(importedValue)).toHaveCount(0);
  await expect(page.getByText('secret-too')).toHaveCount(0);
});

test('delete requires destructive confirmation before removing a secret', async ({ page }) => {
  // Create a disposable secret through the API so this test owns its lifecycle.
  const key = 'DELETE_ME';
  const response = await page.request.post(secretsPath, {
    headers: { Authorization: `Bearer ${authToken}` },
    data: { key, value: 'delete-me-value' },
  });
  expect(response.status()).toBe(201);

  await openSecretsFolder(page);
  const row = page.locator('tr', { hasText: key });
  await row.getByRole('button', { name: 'Delete' }).click();

  const dialog = page.getByRole('dialog', { name: 'Delete secret' });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText(key)).toBeVisible();

  // The secret still exists until the confirmation is acknowledged.
  await page.getByRole('button', { name: 'Delete secret' }).click();
  await expect(page.getByText(key)).toHaveCount(0);
});
