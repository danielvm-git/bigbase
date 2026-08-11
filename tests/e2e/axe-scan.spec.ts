// story: e78s06 (a11y)
// Machine-verified WCAG conformance scan using @axe-core/playwright.
// Scans every SPA route with wcag2a + wcag2aa + wcag2aaa rule tags and
// collects all violations. Each test FAILS on any violation so the run
// doubles as a conformance gate.
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.use({ baseURL: 'http://localhost:9999' });

const EMAIL = `e2e-axe-scan-${Date.now()}@test.com`;
const PASSWORD = 'TestPass123!';
let authToken: string;

// All SPA routes (hash router). Auth pages require the token cookie;
// /login is public.
const ROUTES = [
  { path: '/admin/#/', name: 'Dashboard' },
  { path: '/admin/#/data', name: 'Data Studio' },
  { path: '/admin/#/sql', name: 'SQL Editor' },
  { path: '/admin/#/users', name: 'Users' },
  { path: '/admin/#/repos', name: 'Git Repos' },
  { path: '/admin/#/deploy', name: 'Sites' },
  { path: '/admin/#/deploy/new', name: 'New Site' },
  { path: '/admin/#/messaging', name: 'Messaging' },
  { path: '/admin/#/storage', name: 'Storage' },
  { path: '/admin/#/functions', name: 'Functions' },
  { path: '/admin/#/forge', name: 'Forge' },
  { path: '/admin/#/cici', name: 'CI/CD' },
  { path: '/admin/#/monitoring', name: 'Monitoring' },
  { path: '/admin/#/events', name: 'Events' },
  { path: '/admin/#/realtime', name: 'Realtime' },
  { path: '/admin/#/settings', name: 'Settings' },
  { path: '/admin/#/login', name: 'Login (public)' },
  // Public (Go-served) pages — guard against the landing/docs style drift
  // (e89: token values duplicated in proxy.go/themes.go must stay in parity).
  // Accent-theme coverage is enforced by TestAccentRampParity (Go) + the
  // e88s01 brandLink contrast matrix, not per-route scans.
  { path: '/', name: 'Landing (public)' },
  { path: '/docs', name: 'Docs (public)' },
];

test.beforeAll(async ({ request }) => {
  const regRes = await request.post('/api/auth/register', {
    data: { email: EMAIL, password: PASSWORD },
  });
  expect(regRes.status()).toBe(201);
  authToken = (await regRes.json()).token as string;
});

test.beforeEach(async ({ context }) => {
  await context.addCookies([{ name: 'token', value: authToken, url: 'http://localhost:9999' }]);
});

for (const route of ROUTES) {
  test(`axe scan: ${route.name} (${route.path})`, async ({ page }) => {
    await page.goto(`http://localhost:9999${route.path}`);
    await page.waitForLoadState('networkidle');

    // Wait for the page shell to render (main landmark or h1).
    await page.waitForSelector('.page-header, main, .content', { timeout: 15000 }).catch(() => {});

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag2aaa'])
      .analyze();

    if (results.violations.length > 0) {
      // eslint-disable-next-line no-console
      console.log(`\n=== ${route.name} (${route.path}) — ${results.violations.length} violations ===`);
      for (const v of results.violations) {
        console.log(`  [${v.impact}] ${v.id} — ${v.help}`);
        for (const n of v.nodes.slice(0, 5)) {
          console.log(`      ${n.target.join(' ')} :: ${(n.failureSummary || '').split('\n')[0]}`);
        }
      }
    }

    expect(results.violations, `${route.name} axe violations`).toEqual([]);
  });
}
