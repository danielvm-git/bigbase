# BigBase — Domain Context & Architecture

**type:** tech-stack
**context:** infra
**version:** 2.76.7
**supersedes:** v2.71.0 snapshot
**verify:** `go test ./... && go build ./... && go vet ./...`
**last-scan:** 2026-07-10 (bulk fix session, 17 bugs fixed across deploy/auth/proxy/monitoring/cici/test)

## Overview

BigBase is a single-binary, open-source Backend-as-a-Service (BaaS) platform built
with Go. It uses the **Entity-Component-Construct (ECC)** pattern: a lightweight
kernel discovers, starts, and connects independent components via an event bus.

**~196 Go source files · ~105 test files · ~600 test functions · 20 component directories**
**(Deploy component decomposed per ADR 0005: engine.go, gateway.go, orchestratorator.go — deploy.go reduced from 1875→342 lines)**

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26.3 (`go.mod`) |
| Database | SQLite (`modernc.org/sqlite` v1.50, zero-CGO) · PostgreSQL (`pgx/v5 stdlib`) |
| Auth | bcrypt + HS256 JWT (`golang-jwt/jwt/v5`) · Google OAuth 2.0 · API keys (`X-API-Key`) · OTP · magic-link · phone · **site deploy keys (`bb_dep_*`)** · **MCP org keys with scoped `mcp:provision`** |
| Auth hardening | DB-backed OTP store · rate limiting · audit logging (`components/auth/store.go`, `otp.go`) |
| Functions runtime | goja (pure Go JS) with fetch, db, request context, env, cron bindings |
| Admin UI | React 19 + Vite + TypeScript, embedded via `//go:embed` |
| Design system | Custom token-based system in `ui/src/tokens/` + CSS custom properties |
| SDK | `packages/auth` + adapters (react/vue/svelte/astro/next) + `auth-ui-svelte` |
| Deployment | Child processes (no containers), ACME auto-SSL, custom domains via autocert |
| CI/CD | GitHub Actions + semantic-release |
| Hosting | Contabo VPS, Terraform-defined infra |
| Messaging | Email (SMTP) · SMS · Push · Webhook · Telegram |
| Real-time | WebSocket subscriptions, mutation event broadcasting |
| Scheduling | robfig/cron v3 for Functions `trigger=schedule` |
| Observability | `slog` · New Relic APM · SSE event stream · alert checker · **`eventrecorder` (FIFO 5000)** · **`deploy_diagnoses` + LLM diagnosis (`internal/llm`)** · **`pipeline_timeline`** · **alert incidents + investigations** |

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                           Kernel                                  │
│  • Component discovery & registration                             │
│  • Lifecycle: Init → Start → Stop (topo-sorted DFS)              │
│  • Event bus: hook-based pub/sub, priority-ordered dispatch      │
│  • Config: CLI flags → env vars (BIGBASE_*) → defaults.jsonc     │
│  • Scope context: WithProjectID/WithSiteID (typed, unexported)   │
└────────────────────────┬─────────────────────────────────────────┘
                         │ dispatches events
    ┌────────────────────┼────────────────────┐
    ▼                    ▼                    ▼
┌─────────┐       ┌──────────┐        ┌──────────────┐
│ proxy   │       │   auth   │        │   api/db     │
│ HTTP    │◄─────►│ JWT      │◄──────►│ CRUD + SQL   │
│ router  │       │ bcrypt   │        │ filter/sort  │
│ landing │       │ OAuth    │        │ org scoping  │
│ ACME    │       │ API keys │        │ env vars API │
│ custom  │       │ OTP/link │        └──────┬───────┘
│ domains │       │ site keys│               │
│ auth    │       │ org keys │               ▼
│ policy  │       │ audit log│   ┌──────────────────────┐
│ enforce │       │ orgs     │   │   storage · git       │
│ passtr◄─┤       │ members  │   │   forge · cici        │
│ headers │       └──────────┘   │   functions (goja JS) │
└─────────┘                      └──────────────────────┘

┌──────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐
│realtime  │  │messaging │  │ deploy │  │monitoring│
│WebSocket │  │email/SMS │  │process │  │metrics   │
│mutation  │  │push      │  │mgmt    │  │logs      │
│events    │  │webhook   │  │domains │  │alerts    │
└──────────┘  │telegram  │  │delete  │  │SSE stream│
              └──────────┘  │native  │  │hardware  │
                            │DB env  │  │alert     │
                            │pipeline│  │checker   │
                            │timeline│  └──────────┘
                            └────────┘

┌──────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐
│  admin   │  │  sites   │  │github  │  │ webhooks │
│ UI (SPA) │  │deploy GH │  │App auth│  │retry     │
│ 20 pages │  │logs UI   │  │repo    │  │backoff   │
│ dark mode│  │domains   │  │listing │  │delivery  │
└──────────┘  │auth pol. │  └────────┘  └──────────┘
              └──────────┘
```

## Component Catalog

| # | Component | Path | Runtime | Purpose |
|---|-----------|------|---------|---------|
| — | Kernel | `kernel/` | Core | Discovery, lifecycle, event bus, config merge, scope context |
| 1 | Proxy | `components/proxy/` | HTTP | HTTP server, routing, landing, ACME TLS, custom domains, **route auth policy enforcement** |
| 2 | DB | `components/db/` | SQLite/PG | Database access, migrations, dual-driver (sqlite/postgres) |
| 3 | API | `components/api/` | HTTP | Auto CRUD, SQL endpoint, filter/sort, org scoping, env vars |
| 4 | Auth | `components/auth/` | HTTP | Register, login, JWT, Google OAuth, API keys, OTP, magic-link, phone, orgs, invites, rate-limit, **site deploy keys, org API keys, audit logging** |
| 5 | Admin | `components/admin/` | HTTP | Embedded SPA server (React 19, dark mode, responsive) |
| 6 | Storage | `components/storage/` | HTTP | File upload/download/delete, MIME detection |
| 7 | Git | `components/git/` | HTTP | Bare repo management, SSH clone |
| 8 | Forge | `components/forge/` | HTTP | Issues, labels, kanban, wiki |
| 9 | CICI | `components/cici/` | HTTP | Pipeline/workflow engine, git push triggers |
| 10 | Functions | `components/functions/` | HTTP | JS runtime (goja), fetch+db+env+request+cron injections |
| 11 | Realtime | `components/realtime/` | WS | WebSocket subscriptions, mutation event broadcast |
| 12 | Messaging | `components/messaging/` | HTTP | Email/SMS/Push/Webhook/Telegram providers |
| 13 | Deploy | `components/deploy/` | Process | Build/run web apps, state machine, ACME domains, cache, health, supervisor, **native DB env injection, pipeline timeline** |
| 14 | Monitoring | `components/monitoring/` | HTTP | Metrics, logs, alerts, health, SSE events, hardware, **alert checker (breach tracking)** |
| 15 | Sites | `components/sites/` | HTTP | Deploy from GitHub, custom domains CRUD, build/request logs, **auth policy CRUD** |
| 16 | GitHub | `components/github/` | HTTP | GitHub App auth, repo listing |
| 17 | Webhooks | `components/webhooks/` | HTTP | Outbound webhook delivery, retry, exponential backoff |
| 18 | Backup | `components/backup/` | CLI | Backup/restore, migration tooling |
| 19 | MCP | `components/mcp/` | HTTP | Model Context Protocol server (SSE + stdio transport), **Bearer auth with tiered scopes, site discovery, deploy key lifecycle, CI templates** |

**Internal packages:** `components/internal/envcrypto` — AES-256-GCM for site env vars (optional key; no-op when key absent).

## Conventions (Observed from Code)

### Error handling
- All API errors returned as `{"error": "message"}` JSON via a local `writeJSON` helper
- Internal errors are never leaked to API clients (opaque "internal error" strings)
- `fmt.Errorf("context: %w", err)` wrapping throughout — consistent sentinel propagation
- DB errors treated as transient unless they are schema-level (no-such-table → graceful skip)

### API shapes
- REST, snake_case JSON, standard HTTP status codes
- Responses: data directly or `{"error":"..."}` — no envelope wrapper
- Org isolation: `org_id` in every user-data table, injected via middleware context key
- Project scoping: `kernel.WithProjectID(ctx, projectID)` / `kernel.ProjectIDFromContext(ctx)` — typed context keys, **still no callers** (foundation laid, adoption pending)
- Site scoping: `kernel.WithSiteID(ctx, siteID)` / `kernel.SiteIDFromContext(ctx)` — used by proxy and deploy site-key validation flows

### Type safety
- `any` appears in event bus (`Event.Data`) and `writeJSON` — two deliberate seams, not sprawl
- Every component defines its own DB access interface (`type DBer = kernel.DBer` alias or local interface)
- Concrete structs with json tags throughout; no generic containers in business logic
- Context keys use unexported types to prevent external spoofing (`projectIDKeyType`, `siteIDKeyType`)

### Migrations
- Inline SQL strings inside `Init()` — no migration framework, no versioned files
- Pattern: `CREATE TABLE IF NOT EXISTS` (idempotent) + `ALTER TABLE ... ADD COLUMN` (ignored on duplicate)
- Drawback: no rollback, no ordering guarantee across components, schema state not queryable

### Logging
- `slog.Logger` wired at startup → injected via `kernel.Context.Logger` (interface-typed)
- JSON mode in production, text mode in CLI sub-commands
- New Relic nrslog integration wraps the base handler when `--newrelic-enabled`
- Auth audit: structured logs for OTP rate-limit hits, key resolution failures

### Testing
- 101 test files, 584 test functions
- No mocking frameworks — custom lightweight interfaces + `httptest.NewRecorder`
- In-memory SQLite (`:memory:`) for all component tests — no mocks, real DB
- Contract tests in `tests/contract/`, benchmarks in `tests/bench/`
- Each component's tests live alongside source files

### UI
- React 19 + React Router v7, Vite, TypeScript
- All custom components — no third-party UI library
- Design token system: `ui/src/tokens/tokens.ts` (TS constants) + `ui/src/styles/tokens.css` (CSS vars)
- Dark mode via `data-theme="dark"` on `<html>` + CSS custom properties
- Built artifact embedded into Go binary via `//go:embed dist`

## Signals / Active Debt

| Signal | Location | Risk |
|--------|----------|------|
| `auth.go` — 1756 lines | `components/auth/auth.go` | Split started (separate files: otp.go, store.go, apikeys.go, etc.) — manageable but watch for continued growth |
| `proxy/hosts.go` — 488 lines | `components/proxy/hosts.go` | Auth policy enforcement + metadata injection + deployment routing in one middleware; e71 added significant logic here |
| Inline migrations scattered | All component `Init()` | Schema state not observable; migrations run on every boot; ALTER failures silently ignored. 3 new columns added since v2.63: `auth_policy`, `pipeline_timeline`, `status_history` |
| Event bus covers 15 files | `deploy/`, `monitoring/`, `api/`, `github/`, `realtime/`, `proxy/` | Growing adoption; monitoring fix (e72) will align hook names with actual emissions (`deploy.state_changed` vs `"deploy"`) |
| Project scoping — foundation only | `kernel/scope.go` (40 lines) | `WithProjectID`/`ProjectIDFromContext` defined; **zero callers** after 2 release cycles. Auth injection and query scoping stories pending. |
| `DBer` aliased in 13 components | 3 keep local interfaces (backup, mcp, webhooks) | Pattern is stable and understood |
| Scope keys: `SiteID` adopted, `ProjectID` stalled | `kernel/scope.go` | `WithSiteID` called by proxy+deploy flows; `WithProjectID` has no consumers since e57s01 |
| ADR 0007 (e72) landed | `deploy/pipeline_timeline.go`, `internal/eventrecorder/`, `internal/llm/`, `monitoring/observability.go`, `deploy/observability.go` | Pipeline timeline, persistent event recorder (FIFO 5000), `deploy.failed` + `deploy_diagnoses`, related-events snapshot, alert incidents + investigations |
| Deploy log eviction non-deterministic | `deploy/logs.go:14-26` | `initDeployLogs` uses Go map random iteration for victim selection — log loss is unpredictable, not FIFO |

## Recent Features Landed (v2.67 → v2.71)

| Version | Epic | Feature |
|---------|------|---------|
| v2.71.0 | e71 | **Host-Level Route Auth Policy** — `AuthPolicy` on site records, proxy JWT + site-key validation, passthrough identity headers (`X-BigBase-User-ID`, `X-BigBase-Site-ID`), `set_site_auth_policy` MCP tool |
| v2.70.0 | e56 | **OTP Hardening** — DB-backed OTP store (`store.go`), rate limiting, audit logging |
| v2.69.0 | e72 | **MCP Platform Authentication** — Bearer token auth (`auth.go`, 245 lines), tiered access (public/read/write), `mcp:provision` scope enforcement, `OrgKeyAuthenticator` interface |
| v2.68.0 | e69 | **MCP Site Discovery** — site discovery and deploy key lifecycle tools |
| v2.67.0 | e68 | **Native DB Connection String** — `NativeDBEnv()` injects `DATABASE_URL`/`DB_PATH` into deployed app environment |

## Route Auth Policy (e71 — v2.71.0)

Sites can declare authentication rules at the host/proxy layer, eliminating per-app JWT middleware:

```yaml
auth:
  default: public            # or "protected"
  protected_paths:
    - "/books/*"
    - "/pagefind/*"
    - "/mcp"
  public_paths:
    - "/login"
    - "/assets/*"
  accept:
    - jwt                    # BigBase session tokens
    - site_key               # bb_dep_* for CI/service accounts
```

**Flow:**
1. Site record stores `AuthPolicy` as JSON in `sites.auth_policy` column
2. Proxy caches policy in-memory on `RegisterDeploymentHost` (reads DB at registration)
3. `deploymentHostMiddleware` evaluates every request: check `protected_paths` → validate Bearer/Cookie → enforce (401 if unauthorized)
4. For passthrough backends, injects `X-BigBase-User-ID` and `X-BigBase-Site-ID` headers (strips any spoofed incoming headers first)
5. MCP `set_site_auth_policy` tool allows agent-driven configuration; Sites HTTP API (`GET/PUT /api/sites/:id/auth-policy`) mirrors it

## MCP Authentication (e38 / e67 — v2.69.0)

The MCP server supports Bearer token authentication with three tiers:

| Tier | Example tools | Auth required? |
|------|--------------|----------------|
| Public | `ping`, `list_services`, `get_ci_template` | No |
| Read | `get_site`, `list_sites` | Bearer token (any org key) |
| Write | `create_site`, `deploy_site`, `provision_ci_credentials` | Bearer token + `mcp:provision` scope |

**Architecture:**
- `OrgKeyAuthenticator` interface — defined in MCP, implemented by auth at the composition root (no MCP→auth import)
- `bearerAuthMiddleware` — HTTP middleware that reads+parses body for `tools/call`, resolves tool tier, and validates Bearer
- `enforceToolAuth` — SDK-level enforcement with context fallback for test harnesses
- `authenticatePost` — validates Bearer for non-public POST tool calls, writes HTTP errors directly

## Event Bus (expanded usage)

15 source files now emit or subscribe to events (up from ~9 at v2.63):

```
deploy → deploy.state_changed     (state machine transitions)
deploy → deploy.failed            (ADR 0007 — exactly once per deployment ID)
monitoring → alert.triggered      (breach threshold met + duration exceeded)
monitoring ← subscribes to all     (SSE fan-out to connected clients)
api → api.mutation                (realtime broadcast + webhook delivery)
github → github.scaffold_repo     (CI pipeline trigger)
realtime → mutation events         (WebSocket fan-out)
```

## Deploy Architecture (ADR 0005 — implemented)

BigBase replicates Docker supervision — restart-on-crash, health checks — in a lightweight
in-process Go module (ADR 0004). ADR 0005 defines the decomposition, which has been
implemented in the e45 cycle:

```
Deploy (composition root — deploy.go, 342 lines)
  ├── Gateway (gateway.go — HTTP handlers only, delegates to Engine)
  ├── Engine  (engine.go — per-deployment lifecycle: clone→build→start→health→register)
  └── Orchestrator (orchestrator.go — fleet: resume-on-boot, drain, rollback, delete)
```

Supporting modules: `manifest.go` (builder manifest parsing), `supervisor.go` (process
supervision), `health.go` (health probing), `cache.go` (build caching), `state_machine.go`
(status transitions), `runner.go` (Run/RunWithLogs interface), `log_stream.go` (WebSocket
log streaming), `logs.go` (log retention), `request_logs.go` (request-level logging),
`rollback.go`, `schema.go`, `seams.go`, `dep_env.go` (native DB env injection),
`pipeline_timeline.go`, `python.go` (Python build support, e74).

### Domain Language for Security Patterns
These terms describe the deploy component's security posture and appear in reviews,
bug reports, and planning:

| Term | Definition |
|------|------------|
| **SQL-safety doctrine** | All SQL queries MUST use `?` parameter binding. Zero string interpolation allowed. Proven by review. |
| **No-shell subprocess** | All external commands use `exec.Command(argv...)`. No `shell=true`, no `os.system()`. |
| **Auth gate** | Every API endpoint behind a common middleware chain. No "skip auth" bypass. |
| **Unguessable ID** | `kernel.GenerateID()` from `crypto/rand` → 32-char hex, 128 bits. |
| **Site key scoping** | Deploy keys (`bb_dep_*`) grant access to exactly one site. Enforced via `kernel.SiteIDFromContext` at the gateway. |

### Canonical terms (for planning)
- **Engine** — `Run(spec) → Result` behind a single seam; hides clone/build/start/health/cache
- **Gateway** — HTTP handlers only; `FakeEngine` makes tests zero-exec
- **Orchestrator** — fleet management (resume, drain, rollback, delete-site)
- **Supervisor** — crash-loop detect + exponential backoff restart (implemented)
- **PipelineTimeline** — per-stage timestamps (clone, build, start, health) stored as JSON on `deployments` row
- **NativeDBEnv** — injects `DATABASE_URL` (Postgres) or `DB_PATH` (SQLite) into deployed app env

### Deploy Security Hardening (e45)

The deploy component follows four security patterns, proven by security review
(REVIEW_2026-07-10):

1. **SQL-safety doctrine (e45s41)** — every SQL query in the deploy component uses
   `?` parameter binding with `ExecContext`/`QueryRowContext`. Zero instances of
   string interpolation in SQL across all deploy files.
2. **No-shell subprocess pattern** — all `exec.Command` calls use argv form
   (`exec.Command("git", "checkout", branch)`). No `shell=True` or equivalent.
   Even user-controlled `branch` values are safe (argv never passes through a shell).
3. **Auth gate consistency** — all deploy API endpoints share the same middleware
   chain. Site deploy keys (`bb_dep_*`) are scoped to a single site via
   `kernel.SiteIDFromContext`.
4. **Unguessable IDs** — all deployment and repo IDs are generated by
   `kernel.GenerateID()` (16 bytes from `crypto/rand` → 32-char hex string,
   128 bits of entropy). Unguessable per cryptographic property, with no `.` or `/`
   characters that could escape path construction.
5. **Path traversal hardening** — `LoadManifestPath` applies `filepath.Clean` +
   `filepath.IsAbs` + `strings.HasPrefix` (CWE-22 fix).

These patterns are independently verified and tested; no security findings at
confidence ≥ 8 were identified in the July 2026 review.

## External Services

| Service | Integration | Config |
|---------|------------|--------|
| Google OAuth | OAuth 2.0 relay | `--google-client-id`, `--google-client-secret` |
| GitHub | GitHub App (optional) | `--github-app-id`, `--github-app-slug`, `--github-app-private-key-path` |
| SMTP | Email sending | Messaging API, `--messaging-smtp-*` |
| Let's Encrypt | ACME TLS (autocert) | `--acme-email`; DirCache in `certs/` |
| PostgreSQL | Dual-driver DB | `--db-driver postgres`, `--db-dsn` |
| Telegram | Bot messaging | `--messaging-webhook-token` |
| New Relic | APM monitoring | `--newrelic-license-key`, `NEW_RELIC_LICENSE_KEY` |
| DeepSeek LLM | OpenAI-compatible chat | `BIGBASE_LLM_API_KEY` or `DEEPSEEK_API_KEY`; `components/internal/llm` (e72 deploy diagnosis + alert investigation) |

## Key Design Decisions

| ADR | Decision | Status |
|-----|----------|--------|
| 0001 | SQLite + JSON Blob API | Accepted |
| 0002 | JWT + bcrypt Auth | Accepted |
| 0003 | GitHub App for Sites | Accepted |
| 0004 | No containers (Docker/K8s) — in-process Go Supervisor + systemd isolation | Accepted |
| 0005 | Deploy decomposition: Engine + Gateway + Orchestrator | **Implemented** (e45 cycle) |
| — | ECC pattern (no direct cross-component imports) | Accepted |
| — | PostgreSQL dual-driver (modernc.org/sqlite + pgx/v5) | Accepted (e18) |
| — | Multi-tenant org isolation at API/DB layer | Accepted (e23) |
| — | Functions runtime injections: fetch+db+env+request+cron | Accepted (e30) |
| — | Provider interface for pluggable messaging backends | Accepted (e12, e30) |
| 0006 | Site route auth policy at proxy layer (e71) | **Implemented** (v2.71.0) |
| 0007 | E72 observability seams: PipelineTimeline, alert.triggered, EventRecorder, LLM | **Implemented** (e72) — pipeline timeline, eventrecorder, deploy diagnosis, alert incidents + investigations |

## Release History

| Version | Epics | Highlights |
|---------|-------|------------|
| v1.0 | e01—e16 | Foundation: CLI, proxy, DB, auth, admin UI, all 16 components |
| v2.7.0 | e17—e30 | Multi-tenancy, orgs, API keys, webhooks, backup, observability, functions runtime 2.0 |
| v2.30–v2.45 | e31—e45 | SPA auth, passwordless, phone, SDK adapters, auth UI, delete site, SvelteKit, MCP, streaming logs, manifest, env vars, build cache, health, rollback, drain |
| v2.46–v2.55 | e46—e55 | Custom domains + ACME, JWT lifecycle, design system, project scoping backend, deploy supervisor, New Relic, PR-Agent |
| v2.67.0 | e68 | Native DB connection string injection |
| v2.68.0 | e69 | MCP site discovery + deploy key lifecycle |
| v2.69.0 | e38/e67 | MCP Bearer auth, tiered scopes, org key resolution |
| v2.70.0 | e56 | DB-backed OTP store, rate limiting, audit logging |
| v2.71.0 | e71 | Host-level route auth policy, JWT + site-key enforcement, passthrough identity headers |
| v2.72.0 | e72 | Pipeline timeline, correlated event recorder, AI deploy diagnosis, alert investigations |
| v2.73.0 | e74 | Python runtime support (poetry/pip build, app type detection, native DB env) |
| v2.74.0 | e45 | Deploy decomposition (ADR 0005), deploy key prefix + security hardening, SQL-safety doctrine proof |
| v2.75.0 | e26 | Compose deploy, log streaming WebSocket, manifest parser, Node/Python build caching |
| v2.76.0+ | — | Bulk bug fix: 17 fixes across deploy keys+security, dep vulns (7 critical), DAST CSP/Cache-Control, test cleanup, comment stripping |

## Verification

```bash
# Architecture integrity: no direct component imports
grep -r "components/" kernel/ | grep -v "_test.go" | grep -v '//' || echo "No cross-component imports"

# All components implement kernel.Component
grep -l "kernel.Component" components/*/*.go | wc -l

# Event bus wiring
grep -rn "Emit\|Subscribe" components/*/*.go kernel/*.go | grep -v "_test.go"

# Test suite
go test -count=1 ./... && go vet ./...
```

→ **verify:** `go test -count=1 ./... && go build ./... && go vet ./...`
