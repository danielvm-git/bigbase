# Test Design: e75 — E2E UI Test Suite

> **Scope:** Cross-cutting E2E browser test coverage across all BigBase UI routes.
> **Rationale:** 0 browser E2E tests exist today. All 10 Playwright tests are API-only.
> Every UI bug tracked in the registry (deploy keys broken, dashboard/metrics drift, sidebar nav missing, etc.) would have been caught by a browser E2E test.
>
> **Risk level:** P0 (no browser coverage for critical user flows).

---

## 1. Risk Matrix & Scenarios

### P0 — Critical (core UX, release-blocking)

| Scenario ID | Behavior Description | Risk | Test Level | Target File |
|-------------|---------------------|------|------------|-------------|
| SC-e75-P0-01 | **Login Page renders** — email/password fields, submit button, toggle to register mode, Google OAuth button (when configured) | P0 | E2E | `tests/e2e/login-ui.spec.ts` |
| SC-e75-P0-02 | **Login with valid credentials** — fill form, submit, redirect to `/`, cookie set | P0 | E2E | `tests/e2e/login-ui.spec.ts` |
| SC-e75-P0-03 | **Login with invalid credentials** — show error message, no redirect | P0 | E2E | `tests/e2e/login-ui.spec.ts` |
| SC-e75-P0-04 | **Register new user** — fill register form, create account, auto-login, redirect to `/` | P0 | E2E | `tests/e2e/login-ui.spec.ts` |
| SC-e75-P0-05 | **Logout** — click logout sidebar button, redirect to `/login`, cannot access protected routes | P0 | E2E | `tests/e2e/login-ui.spec.ts` |
| SC-e75-P0-06 | **Dashboard renders** — system status panel, site count, navigation present, footer visible | P0 | E2E | `tests/e2e/dashboard.spec.ts` |
| SC-e75-P0-07 | **Site Detail Page loads** — tabs render (Deploy Keys, Env Vars, Cache, Domains), deployment history visible | P0 | E2E | `tests/e2e/sites-ui.spec.ts` |

### P1 — High (common user flows, regression-prone)

| Scenario ID | Behavior Description | Risk | Test Level | Target File |
|-------------|---------------------|------|------------|-------------|
| SC-e75-P1-01 | **Deploy Keys tab — generate key** — open generate modal, fill name, submit, key shown with CopyButton | P1 | E2E | `tests/e2e/deploy-keys-ui.spec.ts` |
| SC-e75-P1-02 | **Deploy Keys tab — copy ID** — click CopyButton on a key row, clipboard has key_id value | P1 | E2E | `tests/e2e/deploy-keys-ui.spec.ts` |
| SC-e75-P1-03 | **Deploy Keys tab — revoke key** — click Revoke, confirm in dialog, key disappears from list | P1 | E2E | `tests/e2e/deploy-keys-ui.spec.ts` |
| SC-e75-P1-04 | **Deploy Keys tab — revoke error displayed** — API failure shows error in modal, list not changed | P1 | E2E | `tests/e2e/deploy-keys-ui.spec.ts` |
| SC-e75-P1-05 | **Create Site wizard — step through** — repo selection, config, deploy, success state | P1 | E2E | `tests/e2e/sites-ui.spec.ts` |
| SC-e75-P1-06 | **Sidebar navigation — all routes accessible** — click each nav item, verify page renders without 404 | P1 | E2E | `tests/e2e/navigation.spec.ts` |
| SC-e75-P1-07 | **Sites list page** — shows sites cards, click navigates to detail | P1 | E2E | `tests/e2e/sites-ui.spec.ts` |
| SC-e75-P1-08 | **Password reset flow** — "Forgot password?" link, enter email, "Email sent" confirmation | P1 | E2E | `tests/e2e/login-ui.spec.ts` |
| SC-e75-P1-09 | **Theme toggle** — click dark/light toggle in sidebar footer, theme persists across navigation | P1 | E2E | `tests/e2e/navigation.spec.ts` |
| SC-e75-P1-10 | **Auth guard** — unauthenticated user redirected to `/login` when accessing protected route | P1 | Integration | `ui/src/components/AppShell.test.tsx` |

### P2 — Medium (secondary features)

| Scenario ID | Behavior Description | Risk | Test Level | Target File |
|-------------|---------------------|------|------------|-------------|
| SC-e75-P2-01 | **Storage page — upload file** | P2 | E2E | `tests/e2e/storage-ui.spec.ts` |
| SC-e75-P2-02 | **Storage page — delete file** | P2 | E2E | `tests/e2e/storage-ui.spec.ts` |
| SC-e75-P2-03 | **Functions page — list and filter** | P2 | E2E | `tests/e2e/functions-ui.spec.ts` |
| SC-e75-P2-04 | **Function Detail page — view logs** | P2 | E2E | `tests/e2e/functions-ui.spec.ts` |
| SC-e75-P2-05 | **Messaging page — create template** | P2 | E2E | `tests/e2e/messaging-ui.spec.ts` |
| SC-e75-P2-06 | **Settings page — renders correctly** | P2 | E2E | `tests/e2e/settings-ui.spec.ts` |
| SC-e75-P2-07 | **CI/CD page — pipeline list renders** | P2 | E2E | `tests/e2e/cici-ui.spec.ts` |
| SC-e75-P2-08 | **Monitoring page — metrics load** | P2 | E2E | `tests/e2e/monitoring-ui.spec.ts` |
| SC-e75-P2-09 | **404 page — unknown route** — navigating to `/nonexistent` shows NotFoundPage | P2 | E2E | `tests/e2e/navigation.spec.ts` |
| SC-e75-P2-10 | **Users page — list renders with data** | P2 | E2E | `tests/e2e/users-ui.spec.ts` |

### P3 — Low (edge cases, states)

| Scenario ID | Behavior Description | Risk | Test Level | Target File |
|-------------|---------------------|------|------------|-------------|
| SC-e75-P3-01 | **Empty states** — pages with no data display EmptyState component (e.g., no sites, no deploy keys) | P3 | Integration | `ui/src/components/EmptyState.test.tsx` |
| SC-e75-P3-02 | **API error displays error message** — network failure shows error UI, not blank page | P3 | Integration | `ui/src/components/Alert.test.tsx` |
| SC-e75-P3-03 | **Loading skeleton** — pages show skeleton/loading state while data loads | P3 | Integration | `ui/src/components/SkeletonCard.test.tsx` |
| SC-e75-P3-04 | **Realtime page — connection status** | P3 | E2E | `tests/e2e/realtime-ui.spec.ts` |
| SC-e75-P3-05 | **Events page — event list renders** | P3 | E2E | `tests/e2e/events-ui.spec.ts` |
| SC-e75-P3-06 | **Git Repos page — list renders** | P3 | E2E | `tests/e2e/repos-ui.spec.ts` |
| SC-e75-P3-07 | **Forge page — issues/kanban renders** | P3 | E2E | `tests/e2e/forge-ui.spec.ts` |

---

## 2. Fixture Architecture & Isolation

### Shared E2E fixture (`tests/e2e/fixtures/auth.ts`)
```typescript
interface AuthFixture {
  authToken: string
  email: string
  password: string
  siteId?: string  // created during setup
}

// Test fixture: registers user, logs in, optionally creates a site
export async function createAuthenticatedContext(request: APIRequestContext): Promise<AuthFixture>
export async function createSiteWithKeys(request: APIRequestContext, token: string): Promise<{ siteId: string }>
```

### Database state
- Playwright config already starts a fresh BigBase instance:
  `go run . serve --port 9999 --db /tmp/bigbase-e2e.db`
- Each spec file uses unique email (`e2e-${Date.now()}@test.com`) to avoid collisions
- **Isolation:** No shared state between test files — each file registers its own user

### Network intercepts (for integration tests)
- MSW handlers in `ui/src/mocks/` for vitest integration tests
- E2E tests hit the real BigBase API (no mocking at the network layer — this is the point)

### Playwright config additions needed
```typescript
// Add to existing playwright.config.ts:
projects: [
  {
    name: 'chromium',
    use: {
      ...devices['Desktop Chrome'],
      baseURL: 'http://localhost:9999',
    },
    testMatch: '**/*.spec.ts',
  },
],
```

### Page Object Model
Create shared page objects for reuse across tests:
| Page Object | File | Methods |
|-------------|------|---------|
| `LoginPage` | `tests/e2e/pages/login.page.ts` | `goto()`, `login(email, password)`, `register(email, password)`, `logout()` |
| `DashboardPage` | `tests/e2e/pages/dashboard.page.ts` | `goto()`, `waitForMetrics()` |
| `SitesListPage` | `tests/e2e/pages/sites.page.ts` | `goto()`, `clickSite(name)`, `getSiteCount()` |
| `SiteDetailPage` | `tests/e2e/pages/site-detail.page.ts` | `goto(siteId)`, `activateTab(name)`, `getActiveTab()` |
| `DeployKeysTab` | `tests/e2e/pages/deploy-keys-tab.page.ts` | `generateKey(name)`, `copyKeyId(index)`, `revokeKey(index)`, `getKeys()` |

---

## 3. NFR Verification

| NFR Type | Requirement | Verification Command |
|----------|-------------|----------------------|
| **Perf** | E2E suite completes in < 5 min (30 spec files × 10s avg) | `time npx playwright test tests/e2e/` |
| **Reliability** | < 5% flake rate across 3 CI runs | `npx playwright test --repeat-each=3 --workers=1 tests/e2e/critical.spec.ts 2>&1` |
| **Coverage** | All 20 UI routes exercised by at least one E2E scenario | `scripts/verify-e2e-route-coverage.sh` |
| **Isolation** | No test depends on state from another test | `npx playwright test --workers=4 --grep-invert @deprecated` (verify `--workers>1` doesn't fail) |

---

## 4. Out of Scope

- Mobile/responsive testing (webkit, mobile viewports) — deferred to a follow-up epic
- Visual regression / screenshot diffing — requires baseline images not yet committed
- Accessibility audit automation — currently covered by `ui/src/__tests__/a11y/axe-*.test.tsx`
- Performance / Lighthouse scoring — deferred
- Admin-only pages (no admin role exists in E2E setup)
- WebSocket / Realtime event testing — deferred (requires event emission fixture)

---

## 5. Implementation Order (Stories)

| Story | Scenarios | Effort | Dependencies |
|-------|-----------|--------|-------------|
| **e75s01** — Login E2E | SC-P0-01–05, SC-P1-08 | 3 BCPs | Auth fixture |
| **e75s02** — Navigation & Shell | SC-P0-06, SC-P1-06/09, SC-P2-09 | 2 BCPs | Auth fixture |
| **e75s03** — Sites & Deploy Keys | SC-P0-07, SC-P1-01–05/07 | 4 BCPs | Site fixture |
| **e75s04** — Secondary pages | SC-P2-01–10 | 4 BCPs | Auth fixture |
| **e75s05** — Edge cases & hardening | SC-P3-01–07 | 2 BCPs | All fixtures |

**Total:** 15 BCPs across 5 stories.
