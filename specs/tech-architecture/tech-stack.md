# BigBase — Domain Context & Architecture

**type:** tech-stack
**context:** infra
**version:** 2.63.0
**supersedes:** `specs/plans/TECH_STACK_LATEST.md`, `specs/tech-architecture/TECH_STACK_ARCHIVE.md`
**verify:** `go test ./... && go build ./...`
**last-scan:** 2026-06-29 (cold read, no assumption carry-over)

## Overview

BigBase is a single-binary, open-source Backend-as-a-Service (BaaS) platform built
with Go. It uses the **Entity-Component-Construct (ECC)** pattern: a lightweight
kernel discovers, starts, and connects independent components via an event bus.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26.3 (go.mod) |
| Database | SQLite (`modernc.org/sqlite` v1.50, zero-CGO) · PostgreSQL (`pgx/v5 stdlib`) |
| Auth | bcrypt + HS256 JWT (`golang-jwt/jwt/v5`) · Google OAuth 2.0 · API keys (X-API-Key) · OTP · magic-link · phone |
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
| Observability | `slog` (structured JSON logs) · New Relic APM (`go-agent/v3`) · SSE health events |

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                           Kernel                                  │
│  • Component discovery & registration                             │
│  • Lifecycle: Init → Start → Stop (topo-sorted DFS)              │
│  • Event bus: hook-based pub/sub, priority-ordered dispatch      │
│  • Config: CLI flags → env vars (BIGBASE_*) → defaults.jsonc     │
└────────────────────────┬─────────────────────────────────────────┘
                         │ dispatches events
    ┌────────────────────┼────────────────────┐
    ▼                    ▼                    ▼
┌─────────┐       ┌──────────┐        ┌──────────────┐
│ proxy   │       │   auth   │        │   api/db     │
│ HTTP    │◄─────►│ JWT      │◄──────►│ CRUD + SQL   │
│ router  │       │ bcrypt   │        │ filter/sort  │
│ landing │       │ OAuth    │        │ org scoping  │
│ ACME    │       │ API keys │        └──────┬───────┘
│ custom  │       │ OTP/link │               │
│ domains │       │ orgs     │               ▼
└─────────┘       │ members  │   ┌──────────────────────┐
                  └──────────┘   │   storage · git       │
                                 │   forge · cici        │
                                 │   functions (goja JS) │
                                 └──────────────────────┘

┌──────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐
│realtime  │  │messaging │  │ deploy │  │monitoring│
│WebSocket │  │email/SMS │  │process │  │metrics   │
│mutation  │  │push      │  │mgmt    │  │logs      │
│events    │  │webhook   │  │domains │  │alerts    │
└──────────┘  │telegram  │  │delete  │  │hardware  │
              └──────────┘  └────────┘  │SSE events│
                                        └──────────┘

┌──────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐
│  admin   │  │  sites   │  │github  │  │ webhooks │
│ UI (SPA) │  │deploy GH │  │App auth│  │retry     │
│ 20 pages │  │logs UI   │  │repo    │  │backoff   │
│ dark mode│  │domains   │  │listing │  │delivery  │
└──────────┘  └──────────┘  └────────┘  └──────────┘
```

## Component Catalog

| # | Component | Path | Runtime | Purpose |
|---|-----------|------|---------|---------|
| — | Kernel | `kernel/` | Core | Discovery, lifecycle, event bus, config merge |
| 1 | Proxy | `components/proxy/` | HTTP | HTTP server, routing, landing, ACME TLS, custom domains |
| 2 | DB | `components/db/` | SQLite/PG | Database access, migrations, dual-driver (sqlite/postgres) |
| 3 | API | `components/api/` | HTTP | Auto CRUD, SQL endpoint, filter/sort, org scoping |
| 4 | Auth | `components/auth/` | HTTP | Register, login, JWT, Google OAuth, API keys, OTP, magic-link, phone, orgs, invites, rate-limit |
| 5 | Admin | `components/admin/` | HTTP | Embedded SPA server (React 19, dark mode, responsive) |
| 6 | Storage | `components/storage/` | HTTP | File upload/download/delete, MIME detection |
| 7 | Git | `components/git/` | HTTP | Bare repo management, SSH clone |
| 8 | Forge | `components/forge/` | HTTP | Issues, labels, kanban, wiki |
| 9 | CICI | `components/cici/` | HTTP | Pipeline/workflow engine, git push triggers |
| 10 | Functions | `components/functions/` | HTTP | JS runtime (goja), fetch+db+env+request+cron injections |
| 11 | Realtime | `components/realtime/` | WS | WebSocket subscriptions, mutation event broadcast |
| 12 | Messaging | `components/messaging/` | HTTP | Email/SMS/Push/Webhook/Telegram providers |
| 13 | Deploy | `components/deploy/` | Process | Build/run web apps, state machine, ACME domains, cache, health, supervisor |
| 14 | Monitoring | `components/monitoring/` | HTTP | Metrics, logs, alerts, health, SSE events, hardware |
| 15 | Sites | `components/sites/` | HTTP | Deploy from GitHub, custom domains CRUD, build/request logs |
| 16 | GitHub | `components/github/` | HTTP | GitHub App auth, repo listing |
| 17 | Webhooks | `components/webhooks/` | HTTP | Outbound webhook delivery, retry, exponential backoff |
| 18 | Backup | `components/backup/` | CLI | Backup/restore, migration tooling |
| 19 | MCP | `components/mcp/` | HTTP | Model Context Protocol server (SSE + stdio transport) |

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
- Project scoping foundation: `kernel.WithProjectID(ctx, project_id)` / `kernel.ProjectIDFromContext(ctx)` — typed, unexported context key provides type-safe project ID injection at the kernel level. Standardizes the pattern currently duplicated across `auth` and `api` packages.

### Type safety
- `any` appears in event bus (`Event.Data`) and `writeJSON` — two deliberate seams, not sprawl
- Every component defines its own `DBer = kernel.DBer` type alias locally — keeps components decoupled
- Concrete structs with json tags throughout; no generic containers in business logic

### Migrations
- Inline SQL strings inside `Init()` — no migration framework, no versioned files
- Pattern: `CREATE TABLE IF NOT EXISTS` (idempotent) + `ALTER TABLE ... ADD COLUMN` (ignored on error)
- Drawback: no rollback, no ordering guarantee across components, schema state not queryable

### Logging
- `slog.Logger` wired at startup → injected via `kernel.Context.Logger` (interface-typed)
- JSON mode in production, text mode in CLI sub-commands
- New Relic nrslog integration wraps the base handler when `--newrelic-enabled`

### Testing
- 98 test files, 533 test functions
- No mocking frameworks — custom lightweight interfaces + `httptest.NewRecorder`
- In-memory SQLite (`:memory:`) for all component tests — no mocks, real DB
- Contract tests in `tests/contract/`, benchmarks in `tests/bench/`
- Each component's tests live alongside source files

### UI
- React 19 + React Router v7, Vite, TypeScript
- All custom components — no third-party UI library
- Design token system: `ui/src/tokens/tokens.ts` (TS constants) + `ui/src/styles/tokens.css` (CSS vars)
- Dark mode via `data-theme="dark"` on `<html>` + CSS custom properties
- `ui/src/styles/theme.css` overrides semantic tokens for dark mode
- Built artifact embedded into Go binary via `//go:embed dist`

## Signals / Active Debt

| Signal | Location | Risk |
|--------|----------|------|
| `writeJSON` duplicated 14× | `components/*/...go` | DRY violation; shared helper blocked by no cross-component imports rule |
| `deploy.go` — 1819 lines | `components/deploy/deploy.go` | ADR 0005 decomposition (Engine/Gateway/Orchestrator) **designed but not yet implemented** — spec vocabulary exists, code split does not |
| `auth.go` — 1647 lines | `components/auth/auth.go` | Split started (separate files: otp.go, ratelimit.go, jwt.go, etc.) — manageable but watch for more growth |
| Inline migrations scattered | All component `Init()` | Schema state not observable; migrations run on every boot; ALTER failures silently ignored |
| Event bus barely wired | `components/api/api.go`, `deploy/`, `github/` | Only 3 components emit events; monitoring subscribed to dead hook `"deploy"` (deploy emits `deploy.state_changed`); e72 fixes contracts per ADR 0007 |
| Project scoping — foundation laid, callers pending | `kernel/scope.go` | `WithProjectID`/`ProjectIDFromContext` added in e57s01. No callers yet — auth injection (e57s04) and query scoping (e57s03) are follow-up stories. |
| `DBer` aliased in every component | ~14 component files | Working pattern; most use `type DBer = kernel.DBer` alias. Three components (mcp, webhooks, backup) keep local interfaces for test compatibility. Logger standardized to `kernel.Logger` in all components (e57s01). |

## Deploy Architecture (vocabulary — ADR 0005, designed, not yet implemented)

BigBase replicates Docker supervision — restart-on-crash, health checks — in a lightweight
in-process Go module (ADR 0004). ADR 0005 defines the decomposition:

```
Deploy (composition root)
  ├── Gateway (HTTP handlers only, delegates to Engine)
  ├── Engine  (per-deployment lifecycle: clone→build→start→health→register)
  └── Orchestrator (fleet: resume-on-boot, drain, rollback, delete)
```

**Current code reality:** `deploy.go` is still monolithic at 1819 lines. The state machine,
supervisor, health, cache, and domain routing have been extracted into separate files
(`state_machine.go`, `supervisor.go`, `health.go`, `cache.go`, `domain_routing.go`)
but the main `Deploy` struct and its HTTP handlers still coexist in `deploy.go`.
ADR 0005 vocabulary is the spec; the full split is e52+ work.

### Canonical terms (for planning)

- **Engine** — `Run(spec) → Result` behind a single seam; hides clone/build/start/health/cache
- **Gateway** — HTTP handlers only; `FakeEngine` makes tests zero-exec
- **Orchestrator** — fleet management (resume, drain, rollback, delete-site)
- **Supervisor** — crash-loop detect + exponential backoff restart (`supervisor.go` — implemented)
- **Instance** — one-shot live run (subprocess or static server), unified behind `Runner`
- **Spec** — immutable deploy descriptor; only state that survives restarts
- **DeploymentHostRegistry** — proxy seam mapping host→port (owns drain + connection count)
- **LogStore** — build log adapter: `Append` (engine), `Get` (gateway), `Subscribe` (WS)

## Observability (vocabulary — ADR 0007, epic e72)

Deploy debugging and alert response use explicit domain entities and deep internal modules:

```
Composition root (main.go)
  ├── Deploy Gateway — owns /api/deploy/:id/* routes
  │     └── injects DeployDiagnosisReader, DeployRelatedEventsReader (optional)
  ├── Monitoring — AlertRule checker, EvidenceGatherer, SSE fan-out
  └── internal/eventrecorder — Record + Query (FIFO-capped SQLite)
  └── internal/llm — OpenAI-compatible Complete (optional, no-op without key)
```

### Canonical terms

- **DeploymentState** — coarse lifecycle status on the `deployments` row (`pending` → `building` → `deploying` → `running` / `failed`). Distinct from pipeline timing.
- **PipelineTimeline** — per-stage timestamps (`clone`, `build`, `start`, `health`) stored as `pipeline_timeline` JSON on `deployments`. Orthogonal to DeploymentState.
- **DeployFailure** — derived domain moment: `TransitionState` reaches `failed` **and** `deploy.failed` bus event fires exactly once with diagnostic payload.
- **RecordedEvent** — persisted bus emission in `monitoring_events` (hook, data, timestamp, site_id). FIFO cap 5,000 rows.
- **AlertRule** — threshold configuration in `monitoring_alerts`. Not an incident.
- **AlertIncident** — one open breach episode per rule; deduplicated until resolved. Investigations attach here.
- **InvestigationReport** — evidence bundle (metrics, deployments, events, logs) plus optional LLM summary for an AlertIncident.
- **DeployDiagnosis** — one-shot LLM interpretation of a DeployFailure build log; stored in `deploy_diagnoses`.

### Invariants

- `deploy.failed` emits at most once per deployment ID.
- `alert.triggered` emits at most once per open AlertIncident (not every 30s ticker tick).
- Event bus hook names must match `Event.Name` exactly (`deploy.state_changed`, not `"deploy"`).
- Deploy Gateway never imports monitoring; cross-component behavior uses injected reader interfaces at the composition root.

### Concurrency (e72)

| Shared state | Readers | Writers | Sync |
|--------------|---------|---------|------|
| `eventrecorder` FIFO table | EvidenceGatherer, related-events query | bus subscriber handlers (goroutine + timeout) | SQLite single-writer; insert + trim in one transaction |
| `breachStart` map (alert checker) | alert checker ticker | same goroutine | single goroutine — no lock needed |
| `monitoring_alert_incidents` open row | investigation API | emitAlertTriggered | UNIQUE(rule_id) WHERE resolved_at IS NULL |
| LLM HTTP client | diagnosis + investigation goroutines | concurrent goroutines | stateless; 30s ctx timeout per call |

## Release History

| Version | Epics | Highlights |
|---------|-------|------------|
| v1.0 | e01—e16 | Foundation: CLI, proxy, DB, auth, admin UI, all 16 components |
| v2.7.0 | e17—e30 | Multi-tenancy, orgs, API keys, webhooks, backup, observability, functions runtime 2.0 |
| v2.30–v2.45 | e31—e45 | SPA auth, passwordless, phone, SDK adapters, auth UI, delete site, SvelteKit, MCP, streaming logs, manifest, env vars, build cache, health, rollback, drain |
| v2.46–v2.55 | e46—e55 | Custom domains + ACME (e46), JWT lifecycle (e50), design system (e51), project scoping backend (e52), deploy supervisor (e53), New Relic (e54), PR-Agent (e55) |
| v2.62.0 | e56+ | OTP hardening, project scoping UI, native port (SQL-over-HTTP, Better Auth, MCP Tools), secrets, CSP headers, usage dashboard planned |

## Event Flow

Components communicate through the kernel event bus. Most cross-component wiring
uses direct DB reads; event bus is used for async side-effects:

```
proxy.onRequest ──► auth validates token/API key ──► api routes (org-scoped)

api.onMutation ──► realtime broadcasts ──► webhooks deliver
                ──► functions triggers (schedule or event)

git.onPush ──► cici runs workflow ──► deploy builds & serves
                                   ──► github.scaffold_repo

db.onBackup ──► monitoring logs

mcp.onToolCall ──► deploy deploys, db queries, api routes
```

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
| DeepSeek LLM (optional) | OpenAI-compatible chat completions | `BIGBASE_LLM_API_KEY` or `DEEPSEEK_API_KEY`; default base `https://api.deepseek.com`, model `deepseek-chat` (e72) |

## Key Design Decisions

| ADR | Decision | Status |
|-----|----------|--------|
| 0001 | SQLite + JSON Blob API | Accepted |
| 0002 | JWT + bcrypt Auth | Accepted |
| 0003 | GitHub App for Sites | Accepted |
| 0004 | No containers (Docker/K8s) — in-process Go Supervisor + systemd isolation | Accepted |
| 0005 | Deploy decomposition: Engine + Gateway + Orchestrator | Accepted (spec); **code split pending** |
| — | ECC pattern (no direct cross-component imports) | Accepted |
| — | PostgreSQL dual-driver (modernc.org/sqlite + pgx/v5) | Accepted (e18) |
| — | Multi-tenant org isolation at API/DB layer | Accepted (e23) |
| — | Functions runtime injections: fetch+db+env+request+cron | Accepted (e30) |
| — | Provider interface for pluggable messaging backends | Accepted (e12, e30) |
| 0007 | E72 observability seams: deploy.failed, EventRecorder, AlertIncident, composition-root readers | Accepted (e72) |

## CLI Reference

```bash
# Serve
go run . serve [--port PORT] [--db PATH] [--db-driver DRIVER] [--db-dsn DSN]
               [--google-client-id ID] [--google-client-secret SECRET]
               [--github-app-id ID] [--github-app-slug SLUG]
               [--github-app-private-key-path PATH] [--github-webhook-secret SECRET]
               [--sites-domain DOMAIN] [--log-level LEVEL]
               [--cors-allowed-origins ORIGINS]
               [--auth-post-login-redirect URL] [--auth-spa-origin-allowlist ORIGINS]
               [--jwt-access-expiry DURATION] [--jwt-refresh-expiry DURATION]
               [--rate-limit-enabled] [--rate-limit-ip-max N] [--rate-limit-user-max N]
               [--mcp-disabled] [--mcp-port PORT] [--mcp-transport TRANSPORT]
               [--newrelic-license-key KEY] [--newrelic-app-name NAME] [--newrelic-enabled]
               [--acme-email EMAIL]

# Key env vars (BIGBASE_* prefix):
#   BIGBASE_JWT_SECRET         — shared secret for HS256 signing (min 32 chars)
#   BIGBASE_JWT_ACCESS_EXPIRY  — access token TTL, e.g. "24h", "30m" (default: 24h)
#   BIGBASE_JWT_REFRESH_EXPIRY — refresh token TTL (default: 720h / 30 days)
#   BIGBASE_DB_DRIVER, BIGBASE_DB_DSN
#   BIGBASE_RATE_LIMIT_ENABLED, BIGBASE_RATE_LIMIT_IP_MAX, BIGBASE_RATE_LIMIT_USER_MAX
#   BIGBASE_PUBLIC_URL, BIGBASE_SITES_DOMAIN

# Info
go run . version
go run . status
go run . components list

# Data
go run . init [--repo PATH]
go run . deploy --repo ID [--branch main] [--server URL] [--api-key KEY] [--wait]
go run . backup --db FILE --output FILE
go run . restore --input FILE --db FILE
go run . migrate up|down|status [--db PATH]
```

## Verification

```bash
# Architecture integrity: no direct component imports
grep -r "components/" kernel/ | grep -v "_test.go" | grep -v '//' || echo "No cross-component imports"

# All 19 components implement kernel.Component
grep -l "kernel.Component" components/*/*.go | wc -l

# Event bus wiring
grep -rn "Emit\|Subscribe" components/*/*.go | grep -v "_test.go"

# Test suite
go test ./... && go vet ./...
```

→ **verify:** `go test -count=1 ./... && go build ./... && go vet ./...`
