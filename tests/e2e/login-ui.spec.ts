// story: e78s01
import { test, expect } from '@playwright/test';

test.use({ baseURL: 'http://localhost:9999' });

const PASSWORD = 'TestPass123!';
let registeredEmail: string;
let authToken: string;

// ═══════════════════════════════════════════════════════════════════
// API-based auth setup — register + login once to avoid rate limiting
// ═══════════════════════════════════════════════════════════════════
test.beforeAll(async ({ request }) => {
  registeredEmail = `e2e-login-ui-${Date.now()}@test.com`;
  const regRes = await request.post('/api/auth/register', {
    data: { email: registeredEmail, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);
  const regBody = await regRes.json();
  authToken = regBody.token;
  expect(typeof authToken).toBe('string');
});

// ═══════════════════════════════════════════════════════════════════
// Registration Form → Login via Form → Dashboard (serial because
// the registration step creates state consumed by the login step)
// ═══════════════════════════════════════════════════════════════════
test.describe.serial('Registration and Login', () => {
  test('Registration form creates account and redirects to dashboard', async ({ page }) => {
    const formEmail = `e2e-reg-form-${Date.now()}@test.com`;

    await page.goto('/admin/#login');

    // Wait for the login page to render (h1 brand heading)
    await expect(page.getByRole('heading', { name: 'BigBase' })).toBeVisible();

    // Toggle from sign-in mode to register mode
    await page.getByRole('button', { name: 'Register' }).click();

    // Fill in credentials
    await page.getByPlaceholder('Email').fill(formEmail);
    await page.getByPlaceholder('Password').fill(PASSWORD);

    // Submit the registration form
    await page.getByRole('button', { name: 'Register', exact: true }).click();

    // Should redirect to the dashboard (hash root)
    await page.waitForURL('**/admin/#/');

    // Verify dashboard is rendered
    await expect(page.getByText(/Welcome back/i)).toBeVisible();
  });

  test('Login with valid credentials sets session cookie and shows dashboard', async ({ page }) => {
    // Clear any existing session to start from a clean state
    await page.context().clearCookies();

    await page.goto('/admin/#login');
    await expect(page.getByRole('heading', { name: 'BigBase' })).toBeVisible();

    // Default mode is "Sign In" — fill in credentials (user was
    // registered in beforeAll, but we use the form for this UI test)
    await page.getByPlaceholder('Email').fill(registeredEmail);
    await page.getByPlaceholder('Password').fill(PASSWORD);

    // Submit the login form
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Should redirect to the dashboard
    await page.waitForURL('**/admin/#/');

    // Verify the session cookie was set
    const cookies = await page.context().cookies();
    const tokenCookie = cookies.find(c => c.name === 'token');
    expect(tokenCookie, 'Session cookie "token" should be set after login').toBeDefined();
    expect(tokenCookie?.value).toBeTruthy();

    // Verify dashboard content is visible
    await expect(page.getByText(/Welcome back/i)).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════
// Password Reset Flow (no auth needed — forgot-password is public)
// ═══════════════════════════════════════════════════════════════════
test('Password Reset flow shows confirmation message', async ({ page }) => {
  await page.goto('/admin/#login');
  await expect(page.getByRole('heading', { name: 'BigBase' })).toBeVisible();

  // Click the "Forgot password?" link
  await page.getByRole('button', { name: /Forgot password/i }).click();

  // The view switches to the password reset form
  await expect(page.getByRole('heading', { name: 'Reset password' })).toBeVisible();
  await page.getByPlaceholder('Email').fill(`e2e-reset-${Date.now()}@test.com`);

  // Submit the reset request
  await page.getByRole('button', { name: /Send reset link/i }).click();

  // Verify the success confirmation message appears
  await expect(page.getByText(/If an account exists/i)).toBeVisible();
});

// ═══════════════════════════════════════════════════════════════════
// Auth Framework Component Previews (requires auth token cookie)
// ═══════════════════════════════════════════════════════════════════
test('Auth Framework Component Preview renders on /admin/auth', async ({ page }) => {
  // Inject auth token cookie before navigating to auth-protected page
  await page.context().addCookies([
    { name: 'token', value: authToken, domain: 'localhost', path: '/' },
  ]);

  await page.goto('/admin/#auth');

  // Verify we land on the auth preview page
  await expect(page).toHaveURL(/\/admin\/#auth/);

  // The page should contain an h1 heading (framework preview title)
  await expect(page.locator('h1')).toBeVisible();
});
