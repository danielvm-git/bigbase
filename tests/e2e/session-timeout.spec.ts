// story: e87s05
// Session timeout warning + graceful re-authentication (WCAG 2.2.6 / 2.2.5).
//
// Requires the e87s05 UI (SessionTimeoutWarning + LoginPage route restore) to be
// present in the served SPA. The SPA is embedded from ui/dist at build time
// (ui/embed.go), so this suite only passes against a dist built after e87s05
// lands. baseURL / webserver come from playwright.config.ts
// (--jwt-access-expiry=30s), which is far below the 5-minute warning
// threshold — the warning dialog is therefore always visible while signed in,
// which is what these tests exercise.
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const password = 'TestPass123!';

async function registerViaApi(request: APIRequestContext, userEmail: string): Promise<void> {
  const res = await request.post('/api/auth/register', {
    data: { email: userEmail, password },
  });
  expect(res.status()).toBe(201);
}

async function loginViaUi(page: Page, userEmail: string): Promise<void> {
  await page.goto('/admin/#login');
  await page.getByPlaceholder('Email').fill(userEmail);
  await page.getByPlaceholder('Password').fill(password);
  await page.getByRole('button', { name: 'Sign In' }).click();
  await page.waitForURL('**/admin/#/');
}

test.describe('Session timeout warning + re-auth (e87s05)', () => {
  test.setTimeout(120_000);

  test('warning appears and "Stay signed in" extends the session past the original expiry', async ({
    page,
    request,
  }) => {
    const testEmail = `e2e-timeout-${Date.now()}@test.com`
    await registerViaApi(request, testEmail);
    await loginViaUi(page, testEmail);

    // 30s access expiry < 5 min threshold → the dialog is visible immediately.
    const dialog = page.getByRole('dialog', { name: /Session expiring soon/i });
    await expect(dialog).toBeVisible({ timeout: 10_000 });
    await expect(dialog.getByText(/expires in \d{1,2}:\d{2}/i)).toBeVisible();

    // Let most of the original 30s window elapse, then refresh.
    await page.waitForTimeout(8000);
    await dialog.getByRole('button', { name: 'Stay signed in' }).click();

    // Original expiry (~30s after login) passes without a redirect, so the
    // refresh genuinely extended the session.
    await page.waitForTimeout(24_000);
    await expect(page).toHaveURL(/\/admin\/#\/$/);
    await expect(dialog).toBeVisible();
  });

  test('expired session re-authenticates and restores the route', async ({
    page,
    request,
  }) => {
    const testEmail = `e2e-timeout-${Date.now()}@test.com`
    await registerViaApi(request, testEmail);
    await loginViaUi(page, testEmail);
    await expect(page.getByRole('dialog', { name: /Session expiring soon/i })).toBeVisible({
      timeout: 10_000,
    });

    // Move to a deep route; a reload re-mounts the app (fresh context).
    await page.goto('/admin/#/deploy');
    await expect(page).toHaveURL(/\/admin\/#\/deploy/);

    // Let the access token expire (~30s) while idle.
    await page.waitForTimeout(31_000);

    // Reload fires /api/auth/me with the expired cookie → 401 → graceful
    // re-auth: route saved, redirected to /login with the session-expired notice.
    await page.reload();
    await expect(page).toHaveURL(/\/admin\/#\/login/, { timeout: 15_000 });
    await expect(page.getByText(/Your session expired/i)).toBeVisible();
    const pending = await page.evaluate(() =>
      sessionStorage.getItem('bigbase.pendingRoute'),
    );
    expect(pending).toBe('/deploy');

    // Sign in again → the saved route is restored (and cleared).
    await page.getByPlaceholder('Email').fill(testEmail)
    await page.getByPlaceholder('Password').fill(password);
    await page.getByRole('button', { name: 'Sign In' }).click();
    await expect(page).toHaveURL(/\/admin\/#\/deploy/, { timeout: 15_000 });
    const pendingAfter = await page.evaluate(() =>
      sessionStorage.getItem('bigbase.pendingRoute'),
    );
    expect(pendingAfter).toBeNull();
  });
});
