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

**Acceptance criteria:**
- All 8 screens render without console errors
- Dark mode toggles and persists across page reload
- Responsive layout works on mobile (≤768px)
- Form inputs validate (email, repo selection, env vars)
- Status badges and spinners animate correctly
- Toast notifications display and auto-dismiss
- Navigation persists with hash routing
- `npm run build` passes clean

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
| 1 | **017** (Admin UI) + 021 (Testing) | All 8 screens render, `npm run build` passes, dark mode + responsive |
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
