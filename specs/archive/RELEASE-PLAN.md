# BigBase v2.0 — Release Plan

9 epics organized UI-first. Each epic is a vertical slice with
independently testable steps. Testing (Epic 021) runs in parallel with all
other epics — see `specs/TASKS.md` for the dependency graph and parallel
execution groups.

---

## Epic 017: Enhanced Admin UI

**type:** feat
**context:** ui
**WSJF:** 1 (start now — fully spec'd, zero backend dependencies)

**Context:** Admin UI has 7 pages but misses dedicated views for the full
console experience. Design system, 8 screens, dark mode, responsive layout,
toast notifications, and live metrics are all spec'd in
`specs/epics/017-enhanced-admin-ui/`. This is the first epic because it's
independent and produces immediate visible value.

**Design reference:** `specs/epics/017-enhanced-admin-ui/`
- `SYSTEM_DESIGN.md` — design tokens, color palette, typography, API endpoints
- `COMPONENT_INVENTORY.md` — 24 primitive components + 8 screen specs
- `IMPLEMENTATION_GUIDE.md` — 5-phase migration path, React/TypeScript stack

**Design tokens:**
- Brand: Indigo `#4F46E5` | Accent: Emerald `#10B981` | Warning: Amber `#F59E0B` | Error: Red `#EF4444`
- Sans: Inter (400/500/600/700) | Mono: Fira Code (400/500/600)
- Sidebar: 240px desktop → 64px icon-only mobile

**8 screens to deliver:**
1. Dashboard — health banner, component grid, request rate chart, activity feed, quick-actions
2. Sites management — list + create wizard (source → configure → deploy) + site detail (overview/deployments/domains/logs/settings)
3. SQL Editor — dark-themed query editor, table browser sidebar, result table
4. Storage browser — bucket navigation, file list, drag-drop upload, image preview
5. Users — team member list, invite flow, role badges, verification status
6. Git Repos — repository browser with language/visibility badges, "Create site" shortcut
7. CI/CD Pipelines — success rate tiles, run list with status/trigger/actor/duration
8. Monitoring — component health grid (16/16), CPU/memory graphs, activity feed

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **017-A** | Realtime inspector page at `#/realtime`. List active WebSocket connections, view live event feed, inspect channel subscriptions | `cd ui && npm run build` |
| 2 | **017-B** | Function logs viewer at `#/functions/:id/logs`. Execution history, console output capture, error traces. New `GET /api/functions/:id/logs` backend endpoint | `go test ./components/functions/ -run TestLogs -v` |
| 3 | **017-C** | Storage browser at `#/storage`. File grid/list toggle, preview modal (images, text), folder tree navigation, drag-drop upload. New thumbnail endpoint for image previews | `go test ./components/storage/ -run TestThumbnail -v` |
| 4 | **017-D** | Deploy detail page at `#/deploy/:id` with build log streaming, status timeline, environment variables editor. CICI pipeline viewer at `#/cici/:id` with workflow DAG | `go test ./components/deploy/ -run TestLogStream -v` |
| 5 | **017-E** | Dashboard overhaul: real-time request rate chart, error rate gauge, active user count, component health indicators, dark mode toggle with persistence, responsive sidebar (240px → 64px), toast notification system, quick-action buttons (Create Collection, Deploy Site, Run Function) | `cd ui && npm run build` — visual verification across all 8 screens |
| 6 | **017-F** | Storage page test coverage: `StoragePage.test.tsx` covering grid, list toggle, preview modals, drag-drop, empty state, folder tree | `cd ui && npm test -- StoragePage -- --coverage` |
| 7 | **017-G** | Deploy/CICI detail completion: SSE log stream backend + `StreamLog` component, extract `StatusTimeline`, `EnvVarEditor`, CICI workflow DAG viz, `DeployPage.test.tsx`, `CiciPage.test.tsx` | `go test ./components/deploy/ -run TestDeployLogStream -v && cd ui && npm test -- StreamLog EnvVarEditor DeployPage CiciPage -- --coverage` |
| 8 | **017-H** | Dashboard primitives: `MetricCard`, `RequestChart`, `ComponentHealthGrid`, `QuickActions` with full test coverage | `cd ui && npm test -- MetricCard RequestChart ComponentHealthGrid QuickActions -- --coverage` |
| 9 | **017-I** | Dashboard tests + polish: `DashboardPage.test.tsx`, wire `StatusTimeline` into DeployPage, close CSS var gap (22 → 30+) | `cd ui && npm test -- DashboardPage -- --coverage && grep -c 'var(--' ui/src/styles/tokens.css \| awk '{if($1>=30) print "PASS"; else print "FAIL"}'` |
| 10 | **017-J** | SiteDetail tabs + Users enhancements: 5-tab navigation (overview/deployments/domains/logs/settings), custom domain input, build settings editor, danger zone, Users invite flow, metric tiles | `cd ui && npm test -- SiteDetailPage UsersPage -- --coverage && npm run build` |

**Acceptance criteria:**
- All 8 screens render without console errors
- Dark mode toggles and persists across page reload
- Responsive layout works on mobile (≤768px)
- Form inputs validate (email, repo selection, env vars)
- Status badges and spinners animate correctly
- Toast notifications display and auto-dismiss
- Navigation persists with hash routing
- `npm run build` passes clean

### Story 017-F: Storage Page Test Coverage

**WSJF:** 6.0 (BV: 5 + TC: 6 + RR: 7) / JS: 3

**Context:** `StoragePage.tsx` exists but has zero test coverage. The plan calls for
tests covering the file grid, list toggle, preview modal (images + text), drag-drop
zone, and folder tree navigation.

**Gherkin AC:**
```gherkin
Scenario: Storage page renders file grid
  Given the storage page loads
  Then the file grid is visible
  And each file shows its name and type icon

Scenario: Toggle between grid and list view
  Given the storage page is in grid mode
  When the user clicks the list toggle
  Then files are displayed as table rows

Scenario: Image preview modal
  Given a file with type "image/png" exists
  When the user clicks the file
  Then a preview modal opens showing the image

Scenario: Text file preview modal
  Given a file with type "text/plain" exists
  When the user clicks the file
  Then a preview modal opens showing the file contents

Scenario: Drag-drop upload zone
  Given the storage page is open
  When a file is dragged onto the drop zone
  Then the upload zone highlights
  And the file is added to the upload queue

Scenario: Empty state
  Given the bucket has no files
  Then the storage page shows an empty state message
```

**Tasks:**
| # | Action | Verify |
|---|--------|--------|
| 1 | Write `StoragePage.test.tsx` — cover grid render, list toggle, image preview, text preview, drag-drop zone, empty state, folder tree navigation | `cd ui && npm test -- StoragePage` — **RED** then **GREEN** |
| 2 | Integrate with existing `npm run preflight:ui` | `npm run preflight:ui` passes |

**Verify:** `cd ui && npm test -- StoragePage -- --coverage && npm run build`

---

### Story 017-G: Deploy & CICI Detail Pages — Full Delivery

**WSJF:** 3.0 (BV: 8 + TC: 7 + RR: 6) / JS: 7

**Context:** `DeployPage.tsx` and `CiciPage.tsx` exist but are basic list views. The
plan calls for detail pages with SSE log streaming, a deploy status timeline, an env
variable editor, and a CICI workflow DAG visualization. `TestDeployLogs` exists but
tests only basic GET/POST — no SSE streaming behavior. All page test files are missing.

**Gherkin AC:**
```gherkin
Scenario: Deploy detail page shows status timeline
  Given a deployment with status "building"
  When the user navigates to /deploy/:id
  Then a status timeline shows: queued → building → deploying → live
  And the current step is highlighted

Scenario: Deploy log stream via SSE
  Given a deployment is in progress
  When the user views the deploy detail page
  Then build logs stream in real-time via SSE
  And each log line shows a timestamp and stream type (stdout/stderr)

Scenario: Environment variable editor
  Given the user is on the deploy detail page
  When they add a new env var KEY=VALUE
  Then the variable appears in the list
  And the save button is enabled

Scenario: CICI workflow DAG visualization
  Given a CICI workflow with 3 steps (build → test → deploy)
  When the user views the CICI detail page
  Then a DAG graph renders nodes for each step
  And edges connect steps in dependency order
  And step status badges update when a run completes
```

**Tasks:**
| # | Action | Verify |
|---|--------|--------|
| 1 | Implement SSE log stream in `components/deploy/deploy.go`: `GET /api/deploy/:id/logs/stream` emits `data: {line, stream, timestamp}` events. Use `http.Flusher`. Buffer last 500 lines for late-connecting clients | `go test ./components/deploy/ -run TestDeployLogStream -v` — **RED** then **GREEN** |
| 2 | Extract `StatusTimeline` from `SiteDetailPage.tsx` into `ui/src/components/StatusTimeline.tsx`. Accept `steps: {label, status}[]` prop. Use in DeployPage and SiteDetailPage | `cd ui && npm run build` — no import errors |
| 3 | Add `StreamLog` component (`ui/src/components/StreamLog.tsx`): connects to SSE, renders streaming log lines with auto-scroll, shows "connecting"/"live"/"disconnected" status | `cd ui && npm test -- StreamLog` — **RED** then **GREEN** |
| 4 | Add `EnvVarEditor` component (`ui/src/components/EnvVarEditor.tsx`): key-value pair list with add/remove/edit, validates KEY=VALUE format, save button with loading state | `cd ui && npm test -- EnvVarEditor` — **RED** then **GREEN** |
| 5 | Overhaul `DeployPage.tsx` with detail view: SSE stream log, status timeline, env var editor, deployment metadata. Write `DeployPage.test.tsx` covering detail view, empty state, error state | `cd ui && npm test -- DeployPage -- --coverage` — **GREEN** |
| 6 | Add `DAG` visualization to `CiciPage.tsx` detail view: render workflow steps as nodes with SVG/Canvas edges, step status badges. Write `CiciPage.test.tsx` covering DAG render, status updates, run history | `cd ui && npm test -- CiciPage -- --coverage` — **GREEN** |

**Verify:** `go test ./components/deploy/ -run TestDeployLogStream -v && cd ui && npm test -- StreamLog EnvVarEditor DeployPage CiciPage -- --coverage && npm run build`

---

### Story 017-H: Dashboard Primitive Components

**WSJF:** 3.4 (BV: 7 + TC: 6 + RR: 4) / JS: 5

**Context:** `DashboardPage.tsx` uses only `DashboardMetrics` (component health count).
The plan calls for four additional primitives: `MetricCard` (icon + label + value + trend),
`RequestChart` (line chart from monitoring API), `ComponentHealthGrid` (16 component
status cards), and `QuickActions` (Create Collection, Deploy Site, Run Function buttons).

**Gherkin AC:**
```gherkin
Scenario: MetricCard displays value with trend indicator
  Given a MetricCard with label "Requests/min", value 1423, and trend +12%
  When the card renders
  Then the value is prominently displayed
  And the trend shows a green up-arrow with "+12%"

Scenario: RequestChart renders line chart from API data
  Given the monitoring API returns request rate data points
  When RequestChart mounts
  Then a line chart renders with time on X-axis and request count on Y-axis
  And the chart polls for new data every 5 seconds

Scenario: ComponentHealthGrid shows all 16 components
  Given the component status API returns 16 component entries
  When ComponentHealthGrid renders
  Then 16 cards are displayed
  And each card shows the component name and status indicator (green/red)
  And clicking a card navigates to that component's monitoring section

Scenario: QuickActions triggers navigation or modal
  Given the dashboard page is loaded
  When the user clicks "Create Collection"
  Then they navigate to the SQL editor
  When the user clicks "Deploy Site"
  Then the site creation wizard opens
  When the user clicks "Run Function"
  Then a function execution modal appears
```

**Tasks:**
| # | Action | Verify |
|---|--------|--------|
| 1 | Create `MetricCard.tsx` (`ui/src/components/MetricCard.tsx`): `icon`, `label`, `value`, `trend?`, `trendUp?` props. Renders card with trend indicator. Write `MetricCard.test.tsx` | `cd ui && npm test -- MetricCard` — **RED** then **GREEN** |
| 2 | Create `RequestChart.tsx` (`ui/src/components/RequestChart.tsx`): fetches from `GET /api/monitoring/metrics`, renders SVG/Canvas line chart with axes. 5s polling interval. Handles empty/no-data states. Write `RequestChart.test.tsx` | `cd ui && npm test -- RequestChart` — **RED** then **GREEN** |
| 3 | Create `ComponentHealthGrid.tsx` (`ui/src/components/ComponentHealthGrid.tsx`): fetches component status, renders 16-card grid. Each card: name, status dot, click → route. Handles loading + error states. Write `ComponentHealthGrid.test.tsx` | `cd ui && npm test -- ComponentHealthGrid` — **RED** then **GREEN** |
| 4 | Create `QuickActions.tsx` (`ui/src/components/QuickActions.tsx`): 3 action buttons — Create Collection (→ `/sql`), Deploy Site (→ `/sites?wizard=1`), Run Function (opens modal). Uses `useNavigate`. Uses `useToast` for feedback. Write `QuickActions.test.tsx` | `cd ui && npm test -- QuickActions` — **RED** then **GREEN** |
| 5 | Integrate all 4 primitives into `DashboardPage.tsx`. Replace placeholder stats with MetricCards, add RequestChart section, replace component count with ComponentHealthGrid, add QuickActions row | `cd ui && npm run build` — dashboard renders with all primitives |

**Verify:** `cd ui && npm test -- MetricCard RequestChart ComponentHealthGrid QuickActions -- --coverage && npm run build`

---

### Story 017-I: Dashboard Test Coverage + Polish

**WSJF:** 5.3 (BV: 5 + TC: 5 + RR: 6) / JS: 3

**Context:** `DashboardPage.tsx` has no test file. `StatusTimeline` was extracted in 017-G
but needs usage in DeployPage. CSS custom property `var(--` usage in tokens.css is 22,
short of the 30 minimum. Close both gaps.

**Gherkin AC:**
```gherkin
Scenario: Dashboard page renders all sections
  Given the user is authenticated
  When the dashboard page loads
  Then the health banner is visible
  And the component health grid shows 16 cards
  And the request rate chart renders
  And the quick-action buttons are present

Scenario: Dashboard handles loading state
  Given API responses are pending
  When the dashboard page is loading
  Then skeleton placeholders are shown

Scenario: Dashboard handles error state
  Given the health API returns a 500 error
  When the dashboard page loads
  Then an error banner is displayed
  And the remaining sections still render

Scenario: Dark mode applies correctly to dashboard
  Given dark mode is active
  When the dashboard page renders
  Then all MetricCards use dark theme colors
  And the RequestChart background is dark

Scenario: CSS token coverage meets threshold
  Given the tokens.css file
  Then at least 30 unique var(-- references exist across the UI
```

**Tasks:**
| # | Action | Verify |
|---|--------|--------|
| 1 | Write `DashboardPage.test.tsx`: renders all sections (health banner, component grid, chart, quick actions), handles loading spinner, handles API error states, verifies dark mode CSS class application, tests toast on quick-action click | `cd ui && npm test -- DashboardPage` — **RED** then **GREEN** |
| 2 | Ensure `StatusTimeline` (extracted in 017-G) is wired into `DeployPage.tsx` detail view | `cd ui && npm run build` — DeployPage uses StatusTimeline |
| 3 | Close CSS var gap: add 8+ new `var(--` usages across page and component styles. Target areas: MetricCard (4), RequestChart (2), ComponentHealthGrid (2), QuickActions (2) | `grep -c 'var(--' ui/src/styles/tokens.css` ≥ 30 |

**Verify:** `cd ui && npm test -- DashboardPage -- --coverage && grep -c 'var(--' ui/src/styles/tokens.css | awk '{if($1>=30) print "PASS: "$1; else print "FAIL: "$1" < 30"}'`

---

### Story 017-J: SiteDetail Tabs + Users Enhancements

**WSJF:** 2.8 (BV: 7 + TC: 6 + RR: 4) / JS: 6

**Context:** The Console HTML prototype defines `SiteDetail` with 5 tabs (overview,
deployments, domains, logs, settings) and a danger zone for site deletion. `SiteDetailPage.tsx`
currently renders a single-scroll page with overview + deployments — no tab navigation,
no domains tab, no settings tab, no danger zone. The Users page exists at 2.4 KB but
lacks the invite flow and metric tiles shown in the prototype.

**Design reference:** `specs/BigBase Console.html` lines 1324–1437 (SiteDetail with tabs),
1592–1630 (Users with metric tiles + invite), 1397–1434 (domains + settings + danger zone).

**Gherkin AC:**
```gherkin
Scenario: SiteDetail renders 5-tab navigation
  Given a site detail page is loaded
  Then 5 tabs are visible: Overview, Deployments, Domains, Logs, Settings
  And the Overview tab is selected by default

Scenario: SiteDetail overview tab shows production deployment
  Given the Overview tab is active
  Then the site name, URL, framework, and latest commit are displayed
  And a "Redeploy" and "Visit" button are visible

Scenario: SiteDetail deployments tab shows history table
  Given the Deployments tab is active
  Then a table lists deployments with status, commit, branch, duration, and timestamp

Scenario: SiteDetail domains tab allows custom domain entry
  Given the Domains tab is active
  Then the BigBase subdomain is shown with Active badge
  And an input field accepts a custom domain name
  And CNAME setup instructions are displayed

Scenario: SiteDetail logs tab shows build log
  Given the Logs tab is active
  Then the build log is displayed in a dark code block
  And each line shows a timestamp and log message

Scenario: SiteDetail settings tab shows build configuration
  Given the Settings tab is active
  Then a "Build settings" card shows framework preset, build command, and output directory
  And a "Save changes" button is enabled after edits

Scenario: SiteDetail danger zone allows site deletion
  Given the Settings tab is active
  When the user clicks "Delete site"
  Then a confirmation dialog appears
  And confirming deletes the site and redirects to the sites list

Scenario: Users page shows metric tiles
  Given the Users page is loaded
  Then three metric tiles display: Total users, Verified, and Admins
  And each tile shows a count value

Scenario: Users page invite flow triggers modal
  Given the Users page is loaded
  When the user clicks "Invite user"
  Then an invite modal opens with email input
  And submitting shows a toast confirmation
```

**Tasks:**
| # | Action | Verify |
|---|--------|--------|
| 1 | Add `Tabs` container to `SiteDetailPage.tsx`: import existing `Tabs` component, render 5 tab buttons (Overview, Deployments, Domains, Logs, Settings). Move existing single-page content into Overview + Deployments tabs | `cd ui && npm run build` — SiteDetail loads with tabs |
| 2 | Build Domains tab: BigBase subdomain display with Active badge, custom domain input with "Add domain" button, CNAME setup instructions. Mock-only — no backend changes | `cd ui && npm run build` — Domains tab renders |
| 3 | Build Logs tab: re-export `BUILD_LOG` mock data from `specs/BigBase Console.html` (or use `window.DATA` equivalent). Render dark code-output block with log lines (timestamp + message). Use existing `.log-line` + `.code-output` CSS classes | `cd ui && npm run build` — Logs tab renders |
| 4 | Build Settings tab: Build settings card (framework preset dropdown, build command input, output directory input, Save button). Danger zone card (red border, "Delete this site" description, "Delete site" button with confirmation dialog) | `cd ui && npm run build` — Settings tab renders, danger zone styled |
| 5 | Write `SiteDetailPage.test.tsx`: test all 5 tabs render, test tab switching, test domains input, test settings save, test danger zone confirmation flow, test loading and error states | `cd ui && npm test -- SiteDetailPage` — **RED** then **GREEN** |
| 6 | Enhance `UsersPage.tsx`: add 3 metric tiles (Total users / Verified / Admins) above the table using prototype's `.metric-grid` + `.metric-tile` patterns. Add "Invite user" button with email modal and toast on submit | `cd ui && npm run build` — Users page shows tiles + invite button |
| 7 | Write `UsersPage.test.tsx`: test metric tiles render with correct counts, test invite modal opens/closes, test email validation in invite form, test toast on successful invite | `cd ui && npm test -- UsersPage` — **RED** then **GREEN** |

**Verify:** `cd ui && npm test -- SiteDetailPage UsersPage -- --coverage && npm run build`

---

## Epic 018: Multi-DB Support (PostgreSQL)

**type:** feat
**context:** infra
**WSJF:** 2 (foundation — unblocks production PostgreSQL deployments)

**Context:** Currently supports SQLite only via `modernc.org/sqlite`. Add PostgreSQL
driver support with a generalized `DBer` interface, config-based driver selection,
and a versioned migration system. Dual-driver CI matrix ensures both paths work.

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **018-A** | Extract shared `DBer` interface into `kernel/dber.go` — consolidate 6 duplicate `DBer` definitions across monitoring, storage, git, forge, cici, functions | `go build ./...` — compiles clean |
| 2 | **018-B** | Implement PostgreSQL driver in `components/db/postgres.go` using `lib/pq`. Connection pool, JSONB support, migration runner | `go test ./components/db/ -run TestPostgres -v` |
| 3 | **018-C** | Add `--db-driver sqlite\|postgres` and `--db-dsn` flags. `db.New()` selects driver based on config. Default remains SQLite | `go run . serve --db-driver postgres --db-dsn "postgres://..." && curl /health` |
| 4 | **018-D** | Dual-driver GitHub Actions matrix: run full `go test ./...` against SQLite (default) and PostgreSQL (with `PG_DSN` env) | CI matrix: 2x parallel, all green |
| 5 | **018-E** | Versioned migration system: `db.Migrate([]Migration)` with up/down, version tracking table, rollback support. Replace ad-hoc `CREATE TABLE IF NOT EXISTS` patterns | `go test ./components/db/ -run TestMigration -v` |

---

## Epic 019: Security Hardening

**type:** feat
**context:** security
**WSJF:** 3

**Context:** Platform needs rate limiting, email verification, password reset,
refresh token rotation, and security headers. These were deferred in the original
ADR 002 and are now blocking production hardening. Depends on 018-E (migration system).

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **019-A** | Token-bucket rate limiter in `components/auth/ratelimit.go`. Per-IP + per-user buckets. Configurable window and max. Applied to login, register, all API endpoints | `go test ./components/auth/ -run TestRateLimit -v` |
| 2 | **019-B** | Email verification flow: `POST /api/auth/register` sends verification email (via messaging component). `GET /api/auth/verify-email?token=X` confirms. `users.verified` flag | `go test ./components/auth/ -run TestEmailVerify -v` |
| 3 | **019-C** | Password reset: `POST /api/auth/forgot-password` generates 1-hour reset token, emails link. `POST /api/auth/reset-password` with token + new password | `go test ./components/auth/ -run TestPasswordReset -v` |
| 4 | **019-D** | Refresh token rotation: `POST /api/auth/refresh` exchanges refresh token for new access + refresh pair. Old refresh token invalidated on use (rotation prevents replay) | `go test ./components/auth/ -run TestRefreshToken -v` |
| 5 | **019-E** | Security headers middleware: CSP, HSTS (`max-age=31536000`), X-Frame-Options (`DENY`), X-Content-Type-Options (`nosniff`), Referrer-Policy (`strict-origin-when-cross-origin`). Configurable via env vars | `curl -I http://localhost:9999/ \| grep -E "Content-Security\|Strict-Transport\|X-Frame\|X-Content"` |

---

## Epic 020: Platform Operations

**type:** feat
**context:** infra | cli
**WSJF:** 4

**Context:** Production operations require backup/restore, migration tooling,
env var management, custom domains, and an outbound webhook system.
Depends on 018-C (driver flags) and 019-E (security headers).

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **020-A** | Backup/restore CLI: `bigbase backup --db PATH --output FILE` (SQLite `.dump` / PG `pg_dump` compatible). `bigbase restore --db PATH --input FILE`. API: `POST /api/backup`, `POST /api/restore` | `go run . backup --db bigbase.db --output /tmp/backup.sql && go run . restore --db /tmp/restored.db --input /tmp/backup.sql` |
| 2 | **020-B** | Migration tooling CLI: `bigbase migrate [up\|down\|status]`. Versioned migration files in `migrations/`. Integrates with 018-E migration system | `go test ./components/db/ -run TestMigrationTool -v` |
| 3 | **020-C** | Environment variable management: `POST/GET/DELETE /api/env`. Per-deployment and per-function env vars. Encrypted at rest (AES-256-GCM). Admin UI editor page | `go test ./components/api/ -run TestEnvVars -v` |
| 4 | **020-D** | Custom domains for Sites: `POST /api/sites/:id/domains` — add custom domain. DNS verification (TXT record). Caddy/Certbot integration for automatic TLS | `go test ./components/sites/ -run TestCustomDomain -v` |
| 5 | **020-E** | Outbound webhook system: `POST /api/webhooks` — register URL + secret + event types. New `components/webhooks/` component. Deliver on DB mutation, deploy complete, auth events. Retry with exponential backoff (3 attempts) | `go test ./components/webhooks/ -run TestWebhookDelivery -v` |

---

## Epic 021: Testing & Quality Hardening

**type:** feat
**context:** testing | ci
**WSJF:** — (parallel track, woven into all epics)

**Context:** E2E tests, contract tests, benchmarks, coverage gates, and
race/fuzz hardening. Runs alongside other epics — each component's tests are
improved when that component is touched.

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **021-A** | Playwright E2E test suite in `tests/e2e/`. Scenarios: login → dashboard → create collection → add record → deploy → realtime notification. 10+ scenarios | `npx playwright test` — all pass |
| 2 | **021-B** | API contract tests in `tests/contract/`. Snapshot each endpoint's response schema. Run against both SQLite and PostgreSQL drivers. Fails on breaking contract changes | `go test ./tests/contract/...` — passes both drivers |
| 3 | **021-C** | Benchmark suite in `tests/bench/`. DB CRUD throughput, auth ops/sec, WS message rate, function execution time. Reports ops/sec per operation | `go test -bench=. ./...` — results printed |
| 4 | **021-D** | Coverage gates: `go test -coverprofile=coverage.out ./...` enforced at 80% minimum. CI pipeline fails if below threshold. Excludes `ui/`, `main.go`, vendor | CI coverage check gate passes |
| 5 | **021-E** | Race detector in CI (`go test -race ./...`). Fuzz targets for auth token parsing, DB query builder, function runtime, WS message parser. 30-second minimum fuzz time per target | `go test -race ./...` — zero races. `go test -fuzz=. -fuzztime=30s` passes |

---

## Epic 022: Developer Experience

**type:** feat
**context:** ui | telemetry | onboarding
**WSJF:** 5

**Context:** Blank-slate friction and invisible architecture. Onboarding
checklist, 1-click scaffolding, Event Bus visualizer, sample apps, and
interactive tutorials. Depends on 017-E (dashboard) and 018-E (migration system).

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **022-A** | Onboarding checklist card on dashboard: "Create your first collection", "Deploy a site", "Run a function", "Connect GitHub". Progress bar, completion tracking | `cd ui && npm run build` — checklist renders with steps and progress |
| 2 | **022-B** | 1-click scaffolding API: `POST /api/scaffold/db` (Todo schema with 3 tables), `POST /api/scaffold/repo` (React static site), `POST /api/scaffold/function` (hello-world JS). Returns created entity IDs | `go test ./components/api/ -run TestScaffold -v` |
| 3 | **022-C** | Event Bus live stream: `GET /api/monitoring/events` SSE endpoint. Streams kernel event bus dispatches as `data: {"event":"onMutation","source":"db",...}`. Visualizer canvas on `#/events` page with draggable component nodes and animated connections | `curl -N http://localhost:9999/api/monitoring/events` — SSE stream of events |
| 4 | **022-D** | Sample apps: 3 pre-built repos (Todo React, Blog Markdown, Chat WebSocket). Each has a "Deploy" button in Admin UI. `POST /api/samples/:name/deploy` clones and deploys | `go test ./components/deploy/ -run TestSampleDeploy -v` |
| 5 | **022-E** | Interactive tutorial: step-by-step overlay on Admin UI. "Build a Todo app in 5 minutes" — walks through scaffolding, data creation, function writing, deployment. Uses existing scaffolding API + sample apps | Visual verification: `/admin/#/tutorial` steps through correctly |

---

## Epic 023: Multi-Tenancy & Organizations

**type:** feat
**context:** auth | infra
**WSJF:** 6

**Context:** Teams need isolated workspaces. Organizations with member
management, resource scoping, API key authentication, and usage tracking.
Depends on 018-E (migrations) and 019 (security hardening).

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **023-A** | Organization CRUD: `orgs` table, `POST/GET/PATCH/DELETE /api/orgs`. User-org association table. Default org created for existing users on migration | `go test ./components/auth/ -run TestOrganization -v` |
| 2 | **023-B** | Team membership: `POST /api/orgs/:id/members` — invite by email with role (owner/admin/member). `POST /api/orgs/:id/accept-invite` with token. `GET /api/orgs/:id/members` lists team | `go test ./components/auth/ -run TestMembership -v` |
| 3 | **023-C** | Resource isolation: All database collections, storage files, functions, deployments, and sites scoped by `org_id`. Middleware validates org membership on every request. Cross-org access returns 403 | `go test ./components/api/ -run TestOrgIsolation -v` |
| 4 | **023-D** | API key management: `POST /api/orgs/:id/keys` — create scoped key with permissions + expiry. Auth via `X-API-Key` header. Key revocation, listing, last-used tracking | `go test ./components/auth/ -run TestAPIKeys -v` |
| 5 | **023-E** | Usage tracking: Per-org metrics — API calls, storage used (bytes), function invocations, deployments active. `GET /api/orgs/:id/usage`. Org admin only. Charts in Admin UI settings page | `go test ./components/monitoring/ -run TestOrgUsage -v` |

---

## Epic 024: Wire Observability

**type:** feat
**context:** observability | logging | telemetry
**WSJF:** 5.0 (after Platform Ops, before DX)

**Context:** The platform currently uses structured JSON logging (`slog.JSONHandler`)
only in serve mode, and component logging is inconsistent. Health checks exist at
`/health` and `/api/monitoring/health` but don't cover per-component status.
There are no distributed request IDs, no log level controls, and no metrics
export endpoint for external monitoring (Prometheus/Grafana). This epic adds
production-grade observability: structured logging across all components,
distributed tracing via request IDs, per-component health reporting, a
`/metrics` endpoint, and idempotent setup scripts.

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **024-A** | Standardize structured logging: ensure all 16 components use `slog` via injected `Logger` interface. Remove any remaining `fmt.Println` or `log.Printf`. Add `--log-level debug\|info\|warn\|error` flag to `serve` command | `go run . serve --log-level debug 2>&1 \| head -5` shows JSON log entries |
| 2 | **024-B** | Distributed request ID: add `X-Request-ID` header middleware in proxy component. Generate UUID if missing. Propagate through context. Log request ID in every component handler | `curl -I http://localhost:9999/ \| grep -i x-request-id` returns a UUID |
| 3 | **024-C** | Per-component health: extend `/health` to return `{"status":"ok","components":{"proxy":"ok","db":"ok",...}}`. Add `/api/monitoring/health/components` with per-component status + uptime | `curl http://localhost:9999/health \| jq '.components'` shows all 16 components |
| 4 | **024-D** | Metrics export: add `GET /api/monitoring/metrics/prometheus` endpoint with Prometheus text format (counter, gauge, histogram). Export request count, latency p50/p95/p99, error rate, goroutine count, memory | `curl http://localhost:9999/api/monitoring/metrics/prometheus \| grep 'bigbase_'` shows metrics |
| 5 | **024-E** | Idempotent setup: ensure `scripts/setup.sh` handles all dependencies (Go, Node, Caddy, SQLite). Add `scripts/health-check.sh` that curls `/health` and `/metrics` and exits non-zero on failure | `bash scripts/setup.sh` runs twice with no errors. `bash scripts/health-check.sh` returns 0 |

**Acceptance criteria:**
- All 16 components log in structured JSON format
- Every HTTP response includes `X-Request-ID` header
- `/health` returns per-component status (not just binary ok/fail)
- `/api/monitoring/metrics/prometheus` serves Prometheus-compatible metrics
- `scripts/setup.sh` is idempotent (safe to run multiple times)
- `scripts/health-check.sh` validates the running system

---

## Dependency Summary

```
 017 (Admin UI) ── independent, start now ─────────────────┐
                                                             │
 018 (Multi-DB) ──────────────────────────────────────────┐ │
     │                                                      │ │
     ├── 019 (Security) ── needs 018-E (migrations)         │ │
     │       │                                              │ │
     │       ├── 020 (Platform Ops) ── needs 018-C, 019-E   │ │
     │       │       │                                       │ │
     │       │       └── 023 (Multi-tenancy) ── needs 018-E  │ │
     │       │                                                │ │
      │       └── 024 (Observability) ── needs 018-E+019-E    │ │
      │               │                                        │ │
      │               └── 025 (HW Monitoring) ── depends 024   │ │
      │                                                        │ │
      └── 022 (DX) ── needs 017-E + 018-E ────────────────────┘ │

 021 (Testing) ── parallel track, no epic dependencies ─────┘
```

---

## Epic 025: Hardware Monitoring — Connect to Real System

**type:** feat
**context:** monitoring | observability | ui
**WSJF:** 4.25 (after Observability, before DX)

**Context:** The Monitoring page (`#/monitoring`) and monitoring component already
collect Go-level metrics (goroutines, memory, request counts, latencies) via
`/api/monitoring/metrics`, but exposes no host-level hardware data. Operators
need visibility into disk usage (risk of filling up), network I/O (traffic
patterns), per-process resource consumption, and real-time graphs on the Admin
UI. This epic connects the Monitoring page to the real hardware behind the
system.

### Steps

| # | Slice | Action | Verify |
|---|-------|--------|--------|
| 1 | **025-A** | Host metrics collection: add disk usage (`/`, `/opt/bigbase/data`), network I/O (bytes in/out), and system load to `MetricsCollector` in `components/monitoring/`. Use `gopsutil` or `syscall` for cross-platform collection | `curl /api/monitoring/metrics \| jq '.host'` shows `disk_used_pct`, `net_rx_bytes`, `net_tx_bytes`, `load_1m` |
| 2 | **025-B** | Real-time metrics API: add `GET /api/monitoring/metrics/stream` SSE endpoint for live metrics push. Emit CPU, memory, disk, request rate every 5s | `curl -N /api/monitoring/metrics/stream` shows SSE stream of `data: {"cpu":23.5,"mem":512,...}` |
| 3 | **025-C** | Monitoring page charts: add real-time CPU/memory graph (line chart), disk usage gauge, network traffic sparkline, and request rate counter to `MonitoringPage.tsx`. Use polling or SSE for live updates | `cd ui && npm run build` — monitoring page shows live charts |
| 4 | **025-D** | Process-level monitoring: add `/api/monitoring/processes` endpoint showing BigBase and its child processes (deployments, functions). PID, CPU%, memory, uptime per process | `curl /api/monitoring/processes \| jq '.processes'` lists BigBase + child processes |
| 5 | **025-E** | Alerting rules: add configurable alert rules (disk > 80%, memory > 90%, error rate spike) to monitoring component. Trigger webhook or log alert when threshold crossed. Admin UI alert configuration page | `go test ./components/monitoring/ -run TestAlertRules -v` |

**Acceptance criteria:**
- `/api/monitoring/metrics` returns host-level metrics (disk, network, load)
- Monitoring page shows real-time CPU, memory, and disk charts
- SSE endpoint pushes live metrics to connected clients
- Process list shows BigBase and its child deployments
- Alert rules fire when thresholds are crossed
- All new monitoring endpoints have test coverage

---

## Execution Order

| Phase | Epics | Gate |
|-------|-------|------|
| 1 | **017** (Admin UI) + 021 (Testing) | All 8 screens render, `npm run build` passes, dark mode + responsive. **Note:** 017-A–E shipped; 017-F–J are completion stories closing gaps found in audit against source + Console HTML prototype. |
| 2 | **018** (Multi-DB) | Both SQLite + PG drivers green, migration system in place |
| 3 | **019** (Security) | Rate limit, email verify, password reset, refresh tokens, security headers |
| 4 | **020** (Platform Ops) | Backup, migrations CLI, env vars, custom domains, webhooks all green |
| 5 | **024** (Observability) | Structured logging, request IDs, per-component health, Prometheus metrics, idempotent setup |
| 6 | **025** (Hardware Monitoring) | Host metrics (disk, network, load), real-time charts, SSE stream, process monitoring, alert rules |
| 7 | **022** (Developer Experience) | Onboarding flow complete, sample apps deployable, event bus visualizer live |
| 8 | **023** (Multi-Tenancy) | Full org isolation, API keys, usage tracking |

## Out of Scope (v2.0)

See `specs/SCOPE.md` for full listing. Key exclusions: SDK generation, Redis,
containers, enterprise SSO, read replicas, billing.
