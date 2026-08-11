// story: e85s02 — cross-surface theme parity
// Proves a theme chosen in /admin/ (written to localStorage by ThemeProvider)
// is honored by the server-rendered landing page at "/". Both surfaces are
// same-origin, so the bigbase-theme / bigbase-accent keys are shared.
import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-theme-parity-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

test.beforeAll(async ({ request }) => {
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);

  const loginRes = await request.post('/api/auth/login', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(loginRes.status()).toBe(200);
  authToken = (await loginRes.json()).token;
});

test.beforeEach(async ({ context }) => {
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

// Normalize a CSS custom-property value for comparison: trim + collapse spaces.
// The landing script sets e.g. --brand-500 to "rgb(236, 72, 153)".
async function brand500(page: Page): Promise<string> {
  return page.evaluate(() => {
    const v = getComputedStyle(document.documentElement).getPropertyValue('--brand-500');
    return v.replace(/\s+/g, ' ').trim();
  });
}

test.describe('e85 — cross-surface theme parity', () => {
  // ─── Landing reads admin-written localStorage (dark + accent) ──────────
  test('landing reflects admin dark mode + October (pink) accent', async ({ page }) => {
    // Simulate what the admin ThemeProvider writes.
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('bigbase-theme', 'dark');
      localStorage.setItem('bigbase-accent', 'october');
    });
    await page.reload(); // re-run the inline head script with the keys present

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    const brand = await brand500(page);
    expect(brand).toContain('236, 72, 153'); // October — Pink rgb(236, 72, 153)
  });

  // ─── Landing reflects light + default indigo ───────────────────────────
  test('landing reflects light + default indigo accent', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('bigbase-theme', 'light');
      localStorage.setItem('bigbase-accent', 'default');
    });
    await page.reload();

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    const brand = await brand500(page);
    expect(brand).toContain('79, 70, 229'); // default indigo
    // Rainbow flag must NOT be set for a non-rainbow accent.
    await expect(page.locator('html')).not.toHaveAttribute('data-accent-rainbow');
  });

  // ─── Rainbow accent propagates ─────────────────────────────────────────
  test('landing reflects June (rainbow) accent', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('bigbase-theme', 'light');
      localStorage.setItem('bigbase-accent', 'june');
    });
    await page.reload();

    await expect(page.locator('html')).toHaveAttribute('data-accent-rainbow', 'true');
  });

  // ─── Unknown / malicious accent value is rejected (falls back) ─────────
  test('landing rejects an unknown accent value', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('bigbase-theme', 'light');
      localStorage.setItem('bigbase-accent', 'evil<script>'); // not a known id
    });
    await page.reload();

    // Falls back to default indigo — never writes the untrusted string anywhere.
    const brand = await brand500(page);
    expect(brand).toContain('79, 70, 229');
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  });

  // ─── Full chain: admin sidebar UI writes the keys the landing reads ─────
  test('admin sidebar writes theme keys that the landing page honors', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/admin/#/');
    await page.getByRole('navigation').waitFor({ state: 'visible', timeout: 15000 });

    // Toggle dark mode via the sidebar appearance button.
    const darkToggle = page.getByRole('button', { name: /Switch to dark mode/i });
    await darkToggle.click();

    // Pick the March (purple) accent via the ThemePicker popover.
    await page.getByRole('button', { name: /Accent theme/i }).click();
    await page.getByRole('menuitem', { name: /March/i }).click();

    // The admin ThemeProvider must have persisted both keys.
    const stored = await page.evaluate(() => ({
      theme: localStorage.getItem('bigbase-theme'),
      accent: localStorage.getItem('bigbase-accent'),
    }));
    expect(stored.theme).toBe('dark');
    expect(stored.accent).toBe('march');

    // Now navigate to the landing page (same origin → same localStorage).
    await page.goto('/');

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    const brand = await brand500(page);
    expect(brand).toContain('124, 58, 237'); // March — Purple rgb(124, 58, 237)
  });
});
