# Test Design: e78 — Browser E2E Test Coverage (All UI Routes)

> **Scope:** 22 hash-based UI routes across the BigBase admin SPA. Every page must be reachable, render its content, and expose its interactive controls (tabs, forms, buttons) through a real browser.
>
> **Boundary:** e76 covers token-generation/revocation API journeys; e77 covers all HTTP API endpoints. e78 focuses exclusively on **browser-rendered UI interactions**: page loads, navigation, tab switching, form submission through the DOM, loading/empty/error states visible on screen, and cross-page state transitions.
>
> **Risk level:** P0 — currently 0 browser E2E tests exist for any UI route. Every UI regression is caught by users, not tests.

---

## 1. Current State

| Metric | Value |
|--------|-------|
| Existing Playwright E2E files | 3 (`login.spec.ts`, `sites.spec.ts`, `health.spec.ts`) |
| Existing Playwright tests | ~9 |
| API-only tests (they hit endpoints, not pages) | All 9 |
| Browser page-level tests | **0** |
| Total UI routes | 22 |

The 3 existing E2E files test backend API endpoints via `request.post('/api/...')`. None use `page.goto()` or browser interaction. This epic closes the gap by adding true browser-level page tests.

---

## 2. Risk Matrix & Scenarios

### 2.1 — e78s01: Login & Authentication UI (4 BCPs)

| Scenario ID | Behavior Description | Risk | Test Level | Page |
|-------------|----------------------|------|------------|------|
| `SC-e78s01-P0-01` | **Login page loads and renders form** — `page.goto('/#/login')` renders email input, password input, and "Sign in" button without console errors. | P0 | E2E | `LoginPage` |
| `SC-e78s01-P0-02` | **Successful login navigates to dashboard** — fill credentials, click submit, verify redirect to `#/`, dashboard renders with sidebar visible and user greeting. | P0 | E2E | `LoginPage` → `DashboardPage` |
| `SC-e78s01-P0-03` | **Failed login shows error toast** — submit invalid credentials, verify error message appears in the UI and URL remains at `#/login`. | P0 | E2E | `LoginPage` |
| `SC-e78s01-P0-04` | **Logout clears session and redirects** — click logout in sidebar, verify redirect to `#/login`, verify protected routes return to login. | P0 | E2E | `Layout` → `LoginPage` |
| `SC-e78s01-P1-01` | **Loading state during login** — verify submit button shows spinner or is disabled while request is in flight. | P1 | E2E | `LoginPage` |
| `SC-e78s01-P1-02` | **Register link navigates to registration** — click registration link/button, verify registration form renders. | P1 | E2E | `LoginPage` |
| `SC-e78s01-P2-01` | **Password reset link renders** — verify "Forgot password" link is present and clickable. | P2 | E2E | `LoginPage` |
| `SC-e78s01-P2-02` | **Email validation — empty field blocks submit** — submit with empty email, verify inline validation error. | P2 | E2E | `LoginPage` |

---

### 2.2 — e78s02: Dashboard & Navigation (4 BCPs)

| Scenario ID | Behavior Description | Risk | Test Level | Page |
|-------------|----------------------|------|------------|------|
| `SC-e78s02-P0-01` | **Dashboard renders after login** — verify system status cards, onboarding checklist, and recent activity sections are visible. | P0 | E2E | `DashboardPage` |
| `SC-e78s02-P0-02` | **Sidebar renders all sections** — verify "Overview", "Build", "Data", "Auth", "Engage", "DevOps" sections and "Settings" footer link are present. | P0 | E2E | `Layout` |
| `SC-e78s02-P0-03` | **Sidebar navigation — every link navigates** — click each sidebar link, verify the correct page component renders (not just URL change). | P0 | E2E | `Layout` → all pages |
| `SC-e78s02-P0-04` | **Protected route redirect** — navigate to `#/deploy` without auth, verify redirect to `#/login`. | P0 | E2E | Guard |
| `SC-e78s02-P1-01` | **Active sidebar link highlight** — verify the current page's sidebar link has active/highlighted styling. | P1 | E2E | `Layout` |
| `SC-e78s02-P1-02` | **Dashboard metrics load from API** — verify CPU, memory, uptime, and goroutine counts render with real values from `/api/monitoring/metrics`. | P1 | E2E | `DashboardPage` |
| `SC-e78s02-P2-01` | **Onboarding checklist interaction** — verify checklist items can be checked/unchecked (if interactive) or display correct state. | P2 | E2E | `DashboardPage` |
| `SC-e78s02-P3-01` | **NotFound page for bad hash routes** — navigate to `#/nonexistent`, verify 404 page renders with link back to dashboard. | P3 | E2E | `NotFoundPage` |

---

### 2.3 — e78s03: Sites & Deploy UI (6 BCPs)

| Scenario ID | Behavior Description | Risk | Test Level | Page |
|-------------|----------------------|------|------------|------|
| `SC-e78s03-P0-01` | **Deploy page lists sites** — navigate to `#/deploy`, verify site cards render (or empty state if no sites). | P0 | E2E | `DeployPage` |
| `SC-e78s03-P0-02` | **Create Site wizard — Step 1 (Source)** — click "New Site", verify repo selection UI renders (GitHub App or local repo picker). | P0 | E2E | `CreateSitePage` |
| `SC-e78s03-P0-03` | **Create Site wizard — full flow** — select repo → configure name/branch/path → deploy → verify redirect to site detail with deployment in progress. | P0 | E2E | `CreateSitePage` → `SiteDetailPage` |
| `SC-e78s03-P0-04` | **Site detail — all 8 tabs render** — navigate to `#/deploy/:siteId`, click each tab (Deployments, Build Logs, Request Logs, Env Vars, Domains, Deploy Keys, Cache, Manifest), verify each tab's content panel renders. | P0 | E2E | `SiteDetailPage` |
| `SC-e78s03-P0-05` | **Deploy Key generation lifecycle** — open Deploy Keys tab → click "Generate New" → fill name → submit → verify raw `bb_dep_` token displayed in modal → close modal → verify key appears in list → click revoke → confirm → verify key removed from list. | P0 | E2E | `SiteDetailPage` → `SiteDeployKeysTab` |
| `SC-e78s03-P0-06` | **Build Logs tab renders TerminalLogViewer** — navigate to Build Logs tab, verify log content streams or displays placeholder. | P0 | E2E | `SiteDetailPage` |
| `SC-e78s03-P1-01` | **Deploy page filter/search** — type in search input, verify site cards filter in real time. | P1 | E2E | `DeployPage` |
| `SC-e78s03-P1-02` | **Site detail — Env Vars tab CRUD** — add env var → verify appears in list → edit → verify updated → delete → verify removed. | P1 | E2E | `SiteDetailPage` → `SiteEnvVarsTab` |
| `SC-e78s03-P1-03` | **Site detail — Domains tab** — add custom domain → verify DNS verification prompt → verify domain appears in list. | P1 | E2E | `SiteDetailPage` → `SiteDomainsTab` |
| `SC-e78s03-P1-04` | **Site detail — Cache tab** — verify cache stats, hit/miss counts, and prune button render. | P1 | E2E | `SiteDetailPage` → `SiteCacheTab` |
| `SC-e78s03-P1-05` | **Site detail — Manifest tab** — verify bigbase.yaml editor renders with save button. | P1 | E2E | `SiteDetailPage` → Manifest tab |
| `SC-e78s03-P2-01` | **Request Logs tab with filters** — apply status class and path filters, verify log list updates. | P2 | E2E | `SiteDetailPage` → Request Logs |
| `SC-e78s03-P2-02` | **Delete site flow** — click delete on site detail, confirm dialog, verify redirect to deploy list and site absent. | P2 | E2E | `SiteDetailPage` → `DeployPage` |
| `SC-e78s03-P3-01` | **Empty states — no sites** — verify friendly empty state renders when deploy list is empty. | P3 | E2E | `DeployPage` |
| `SC-e78s03-P3-02` | **Loading skeletons** — verify SkeletonCard components render while deploy page fetches site list. | P3 | E2E | `DeployPage` |

---

### 2.4 — e78s04: Data, SQL & Storage (6 BCPs)

| Scenario ID | Behavior Description | Risk | Test Level | Page |
|-------------|----------------------|------|------------|------|
| `SC-e78s04-P0-01` | **Data Studio — collection list** — navigate to `#/data`, verify left sidebar lists available collections (or empty state). | P0 | E2E | `DataStudioPage` |
| `SC-e78s04-P0-02` | **Data Studio — record browser** — click collection, verify record table renders with columns and rows (or empty state). | P0 | E2E | `DataStudioPage` |
| `SC-e78s04-P0-03` | **Data Studio — create record** — click "Add Record", fill form, submit, verify new record appears in table. | P0 | E2E | `DataStudioPage` |
| `SC-e78s04-P0-04` | **SQL Editor — query execution** — navigate to `#/sql`, type `SELECT 1`, click "Run", verify results table renders with the row. | P0 | E2E | `SqlEditorPage` |
| `SC-e78s04-P0-05` | **Storage — file upload** — navigate to `#/storage`, upload a small PNG, verify file appears in list/grid view. | P0 | E2E | `StoragePage` |
| `SC-e78s04-P0-06` | **Storage — file download/delete** — click download on a file, verify browser triggers download; click delete, verify file removed from list. | P0 | E2E | `StoragePage` |
| `SC-e78s04-P1-01` | **Data Studio — filter and sort** — type filter string, verify table rows narrow; change sort column, verify order changes. | P1 | E2E | `DataStudioPage` |
| `SC-e78s04-P1-02` | **Data Studio — Schema mode** — toggle to Schema view, verify column definitions render. | P1 | E2E | `DataStudioPage` |
| `SC-e78s04-P1-03` | **SQL Editor — error handling** — type invalid SQL, run, verify error message renders (not blank screen). | P1 | E2E | `SqlEditorPage` |
| `SC-e78s04-P1-04` | **Storage — view mode toggle** — switch between List and Grid views, verify layout changes. | P1 | E2E | `StoragePage` |
| `SC-e78s04-P2-01` | **Data Studio — "Query this" link** — verify "Query this" button on collection navigates to SQL Editor with pre-filled hint. | P2 | E2E | `DataStudioPage` → `SqlEditorPage` |
| `SC-e78s04-P2-02` | **Storage — oversized file rejection** — attempt upload of file > max size, verify error toast appears. | P2 | E2E | `StoragePage` |
| `SC-e78s04-P3-01` | **Storage — thumbnail preview** — upload an image, verify thumbnail renders in Grid view. | P3 | E2E | `StoragePage` |
| `SC-e78s04-P3-02` | **SQL Editor — empty state** — verify placeholder text when no query has been run. | P3 | E2E | `SqlEditorPage` |

---

### 2.5 — e78s05: Secondary Pages — Functions, Messaging, Users, Settings (6 BCPs)

| Scenario ID | Behavior Description | Risk | Test Level | Page |
|-------------|----------------------|------|------------|------|
| `SC-e78s05-P0-01` | **Functions — list and create** — navigate to `#/functions`, verify function list renders, click "Create", fill form with JS source, submit, verify new function appears. | P0 | E2E | `FunctionsPage` |
| `SC-e78s05-P0-02` | **Function detail — all 4 tabs** — navigate to `#/functions/:id`, verify Code, Triggers, Variables, and Logs tabs all render their content. | P0 | E2E | `FunctionDetailPage` |
| `SC-e78s05-P0-03` | **Function execution** — on FunctionDetailPage Code tab, click "Run", verify execution result/logs appear. | P0 | E2E | `FunctionDetailPage` |
| `SC-e78s05-P0-04` | **Messaging — all 3 tabs** — navigate to `#/messaging`, verify Templates, Send test, and History tabs render with their content. | P0 | E2E | `MessagingPage` |
| `SC-e78s05-P0-05` | **Messaging — Send test form** — fill email "To", "Subject", "Body" fields, verify submit is possible (UI validation passes). | P0 | E2E | `MessagingPage` |
| `SC-e78s05-P0-06` | **Users page renders** — navigate to `#/users`, verify user table renders with at least the logged-in user visible. | P0 | E2E | `UsersPage` |
| `SC-e78s05-P0-07` | **Settings — all 3 tabs** — navigate to `#/settings`, verify Account, Workspace, and Billing tabs render with real data. | P0 | E2E | `SettingsPage` |
| `SC-e78s05-P1-01` | **Messaging — channel sub-tabs** — on Send test tab, switch between EMAIL, SMS, and PUSH channels, verify form fields change. | P1 | E2E | `MessagingPage` |
| `SC-e78s05-P1-02` | **Messaging detail — Editor/Preview tabs** — navigate to a template detail, verify Editor renders source, Preview renders variable substitution. | P1 | E2E | `MessagingDetailPage` |
| `SC-e78s05-P1-03` | **Settings — change password** — fill current + new password fields, submit, verify success toast. | P1 | E2E | `SettingsPage` |
| `SC-e78s05-P1-04` | **Function detail — Variables tab** — edit environment JSON, save, verify success. | P1 | E2E | `FunctionDetailPage` |
| `SC-e78s05-P2-01` | **Settings — Workspace member list** — verify member table renders with roles. | P2 | E2E | `SettingsPage` |
| `SC-e78s05-P2-02` | **Settings — Billing usage stats** — verify functions count, storage MB, and site count render non-zero values. | P2 | E2E | `SettingsPage` |
| `SC-e78s05-P3-01` | **Functions — delete confirmation** — click delete on a function, verify confirmation dialog, confirm, verify function removed. | P3 | E2E | `FunctionsPage` |
| `SC-e78s05-P3-02` | **Messaging — History tab** — verify sent message history table renders status, channel, and timestamp columns. | P3 | E2E | `MessagingPage` |

---

### 2.6 — e78s06: DevOps Pages — CI/CD, Monitoring, Forge, Realtime, Events, Git (10 BCPs)

| Scenario ID | Behavior Description | Risk | Test Level | Page |
|-------------|----------------------|------|------------|------|
| `SC-e78s06-P0-01` | **Git Repos — list, create, delete** — navigate to `#/repos`, verify repo list, click create, fill name, submit, verify appears, delete, verify removed. | P0 | E2E | `GitReposPage` |
| `SC-e78s06-P0-02` | **CI/CD — Workflows tab** — navigate to `#/cici`, verify Workflows tab renders with workflow list (or empty state), YAML editor, and Run button. | P0 | E2E | `CiciPage` |
| `SC-e78s06-P0-03` | **CI/CD — Runs tab** — switch to Runs tab, verify run history table renders with status badges and expandable log rows. | P0 | E2E | `CiciPage` |
| `SC-e78s06-P0-04` | **Monitoring — Overview tab** — navigate to `#/monitoring`, verify CPU, heap, goroutines, uptime stats render with real values, and endpoint-by-status table populates. | P0 | E2E | `MonitoringPage` |
| `SC-e78s06-P0-05` | **Monitoring — Host tab with live SSE** — switch to Host tab, verify CPU donut, Memory donut, Disk bar, and Network sparklines render and update via SSE stream. | P0 | E2E | `MonitoringPage` |
| `SC-e78s06-P0-06` | **Monitoring — Logs and Alerts tabs** — switch to Logs tab (verify searchable log table), switch to Alerts tab (verify alert list + create form). | P0 | E2E | `MonitoringPage` |
| `SC-e78s06-P0-07` | **Forge — Issues tab** — navigate to `#/forge`, select a repo, verify issue table renders (or empty state), click create, fill title + description, submit, verify appears. | P0 | E2E | `ForgePage` |
| `SC-e78s06-P0-08` | **Forge — Board tab** — switch to Board tab, verify 4 Kanban columns render (Open, In Progress, Review, Closed). | P0 | E2E | `ForgePage` |
| `SC-e78s06-P0-09` | **Realtime page** — navigate to `#/realtime`, verify WebSocket connection status, rooms list, and per-user subscription table render. | P0 | E2E | `RealtimePage` |
| `SC-e78s06-P0-10` | **Events page** — navigate to `#/events`, verify live SSE event stream renders events in reverse-chronological order. | P0 | E2E | `EventsPage` |
| `SC-e78s06-P1-01` | **CI/CD — create workflow** — paste YAML, save, verify success toast and workflow appears in list. | P1 | E2E | `CiciPage` |
| `SC-e78s06-P1-02` | **Monitoring — Alerts CRUD** — create alert rule → verify appears in list → delete → verify removed. | P1 | E2E | `MonitoringPage` |
| `SC-e78s06-P1-03` | **Forge — issue status change** — change issue status via dropdown, verify it moves on the Board tab. | P1 | E2E | `ForgePage` |
| `SC-e78s06-P2-01` | **Forge — labels** — create label with name + color, verify appears in issue label picker. | P2 | E2E | `ForgePage` |
| `SC-e78s06-P2-02` | **Monitoring — log search** — type a level filter, verify log table narrows. | P2 | E2E | `MonitoringPage` |
| `SC-e78s06-P3-01` | **Realtime — empty subscriptions** — verify empty state renders when no subscriptions exist. | P3 | E2E | `RealtimePage` |
| `SC-e78s06-P3-02` | **Events — empty stream** — verify empty state renders when no recent events exist. | P3 | E2E | `EventsPage` |

---

## 3. Scenario Count Summary

| Story | P0 | P1 | P2 | P3 | Total |
|-------|----|----|----|----|-------|
| e78s01 — Login & Auth UI | 4 | 2 | 2 | 0 | 8 |
| e78s02 — Dashboard & Nav | 4 | 2 | 1 | 1 | 8 |
| e78s03 — Sites & Deploy UI | 6 | 5 | 2 | 2 | 15 |
| e78s04 — Data, SQL & Storage | 6 | 4 | 2 | 2 | 14 |
| e78s05 — Secondary Pages | 7 | 4 | 2 | 2 | 15 |
| e78s06 — DevOps Pages | 10 | 3 | 2 | 2 | 17 |
| **Total** | **37** | **20** | **11** | **9** | **77** |

---

## 4. Fixture Architecture & Isolation

### 4.1 — Auth State

All protected routes require a logged-in session. Use a global `setup` project in Playwright config:

```typescript
// tests/e2e/auth.setup.ts
import { test as setup, expect } from '@playwright/test';
import path from 'path';

const AUTH_FILE = path.join(__dirname, '.auth/user.json');

setup('authenticate', async ({ request, page }) => {
  const email = `e2e-browser-${Date.now()}@test.com`;
  const password = 'TestPass123!';

  // Register
  const reg = await request.post('/api/auth/register', {
    data: { email, password },
  });
  expect(reg.status()).toBe(201);
  const body = await reg.json();

  // Set auth cookie via page context
  await page.goto('/');
  await page.evaluate(
    ({ token }) => {
      document.cookie = `token=${token}; path=/;`;
    },
    { token: body.token }
  );

  // Store state for dependent projects
  await page.context().storageState({ path: AUTH_FILE });
});
```

All browser test files use `storageState: AUTH_FILE` so auth happens once.

### 4.2 — Database Isolation

```typescript
// playwright.config.ts — webServer section
webServer: {
  command: 'BIGBASE_DB=/tmp/bigbase-e2e-browser.db ./bigbase serve --port 3901',
  url: 'http://localhost:3901/health',
  reuseExistingServer: false,  // always fresh DB
  timeout: 30000,
}
```

Each test run starts with a fresh SQLite database. Auth setup registers a new user each run. Site/repo fixtures are created within each test's `beforeEach`/`beforeAll` using API calls.

### 4.3 — Test File Organization

```
tests/e2e/
├── auth.setup.ts              # Global auth fixture
├── playwright.config.ts       # Updated with storageState + webServer
├── login.spec.ts              # (existing) API-level login tests
├── health.spec.ts             # (existing) health check
├── sites.spec.ts              # (existing) API-level site tests
├── pages/                     # NEW: browser-level page tests
│   ├── e78s01-login-ui.spec.ts
│   ├── e78s02-dashboard-nav.spec.ts
│   ├── e78s03-sites-deploy-ui.spec.ts
│   ├── e78s04-data-sql-storage.spec.ts
│   ├── e78s05-secondary-pages.spec.ts
│   └── e78s06-devops-pages.spec.ts
├── fixtures/
│   ├── site-factory.ts        # Creates site + repo via API
│   ├── function-factory.ts    # Creates JS function via API
│   └── collection-factory.ts  # Creates collection + records via API
└── .auth/
    └── user.json              # Generated by auth.setup.ts
```

### 4.4 — Fixture Factories

```typescript
// tests/e2e/fixtures/site-factory.ts
import { APIRequestContext } from '@playwright/test';

export async function createTestSite(
  request: APIRequestContext,
  token: string
): Promise<{ siteId: string; repoId: string }> {
  // Create bare repo
  const repoRes = await request.post('/api/git/repos', {
    headers: { Authorization: `Bearer ${token}` },
    data: { name: `e2e-repo-${Date.now()}` },
  });
  const repo = await repoRes.json();

  // Create site from repo
  const siteRes = await request.post('/api/sites', {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      name: `e2e-site-${Date.now()}`,
      git_repo_id: repo.id,
      production_branch: 'main',
      root_path: './',
    },
  });
  const site = await siteRes.json();

  return { siteId: site.id, repoId: repo.id };
}
```

### 4.5 — Network Intercepts (limited)

Browser E2E tests run against the real server — no mocks. The only exception is:

- **WebSocket / SSE subscriptions**: Tests verify the WS/SSE connection establishes and the UI renders the initial data; they do not wait for real-time updates. Use `page.waitForResponse()` for API calls and `page.waitForSelector()` for DOM elements.

### 4.6 — Data Cleanup

Tests that create resources (sites, repos, functions, collections) MUST clean up in `afterEach`:

```typescript
test.afterEach(async ({ request }) => {
  if (siteId) {
    await request.delete(`/api/sites/${siteId}`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
  }
});
```

---

## 5. NFR Verification

| NFR Type | Requirement | Verification Command |
|----------|-------------|----------------------|
| **Build** | All 6 spec files compile and Playwright config is valid | `npx playwright test --list 2>&1 \| grep -c 'e78'` |
| **Run** | Full suite completes in < 5 minutes (77 tests, chromium only) | `time npx playwright test tests/e2e/pages/ --project=chromium` |
| **Stability** | Zero flaky tests after 3 consecutive runs | `npx playwright test tests/e2e/pages/ --repeat-each=3 --retries=0` |
| **Coverage gate** | All 22 UI routes are navigated at least once | `rg 'page\.goto\(' tests/e2e/pages/ \| rg -o '/#/[^'"'"']+' \| sort -u \| wc -l` (expect ≥ 20) |
| **Regression** | Existing API-level E2E tests still pass | `npx playwright test tests/e2e/login.spec.ts tests/e2e/sites.spec.ts tests/e2e/health.spec.ts` |
| **Perf** | Each page renders within 3 seconds of navigation | Configured via `test.slow()` threshold + Playwright `timeout` |

---

## 6. Implementation Order (Dependency Graph)

```
e78s01 (Login UI) ─────────────────────────────────────┐
                                                        │
e78s02 (Dashboard & Nav) ── depends on s01 ────────────┤
                                                        │
e78s03 (Sites & Deploy) ── depends on s01, s02 ────────┤
                                                        │
e78s04 (Data, SQL, Storage) ── depends on s01 ─────────┤
                                                        │
e78s05 (Secondary Pages) ── depends on s01 ────────────┤
                                                        │
e78s06 (DevOps Pages) ── depends on s01 ───────────────┘
```

- s01 must ship first (auth is prerequisite for all other pages).
- s02 ships second (nav verification validates s01's auth + sidebar).
- s03–s06 can run in parallel after s01 is done.
- s03 has the most scenarios (15) and the highest risk (deploy key lifecycle is P0 security).

---

## 7. Out of Scope

- **Mobile/tablet viewport tests**: Desktop Chromium only (1280×720 default).
- **Visual regression / pixel-diff testing**: Covered by developer review; not automated.
- **MCP server browser E2E**: MCP is a JSON-RPC protocol tested in e76/e77; no browser UI exists for it.
- **GitHub OAuth callback in browser**: Requires a real GitHub App installation; tested via API-level E2E in e77.
- **Realtime WebSocket full subscription lifecycle**: Verifies page renders and WS connects; does not test mutation event propagation end-to-end (covered by Go integration tests).
- **Third-party integrations (GitHub, SMTP, Telegram)**: Mocked/disabled at the server level during E2E runs.
- **Non-Chromium browsers (Firefox, WebKit)**: Chromium-only to keep CI runtime under 5 minutes.
