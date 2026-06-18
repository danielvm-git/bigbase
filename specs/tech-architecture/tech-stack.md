# BigBase — Domain Context & Architecture

**type:** tech-stack
**context:** infra
**version:** 2.12.0
**supersedes:** `specs/plans/TECH_STACK_LATEST.md`, `specs/tech-architecture/TECH_STACK_ARCHIVE.md`
**verify:** `go test ./... && go build ./...`

## Overview

BigBase is a single-binary, open-source Backend-as-a-Service (BaaS) platform built
with Go. It uses the **Entity-Component-Construct (ECC)** pattern: a lightweight
kernel discovers, starts, and connects independent components via an event bus.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| Database | SQLite (`modernc.org/sqlite`, zero-CGO) · PostgreSQL (`lib/pq`) |
| Auth | bcrypt + HS256 JWT · Google OAuth 2.0 relay · API keys (X-API-Key) |
| Functions runtime | goja (pure Go JavaScript) with fetch, db binding, request context, env, cron schedule |
| Admin UI | React 19 + Vite + TypeScript, embedded via `//go:embed` |
| Deployment | Child processes on host, port-allocation, custom domains with TLS |
| CI/CD | GitHub Actions + semantic-release |
| Hosting | Contabo Cloud VPS (current), Terraform-defined |
| Messaging | Email (SMTP) · SMS · Push · Webhook/Telegram providers |
| Real-time | WebSocket subscriptions, mutation event broadcasting |
| Scheduling | robfig/cron v3 for Functions trigger=schedule |

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                           Kernel                                  │
│  • Component discovery & registration                             │
│  • Lifecycle: Init → Start → Stop (topo-sorted)                  │
│  • Event bus: hook-based pub/sub, priority-ordered dispatch      │
│  • Config merge: CLI flags → defaults → user overrides           │
└────────────────────────┬─────────────────────────────────────────┘
                         │ dispatches events
    ┌────────────────────┼────────────────────┐
    ▼                    ▼                    ▼
┌─────────┐       ┌──────────┐        ┌──────────────┐
│ proxy   │       │   auth   │        │   api/db     │
│ HTTP    │◄─────►│ JWT      │◄──────►│ CRUD + SQL   │
│ router  │       │ bcrypt   │        │ filter/sort  │
│ landing │       │ OAuth    │        │ org scoping  │
└─────────┘       │ API keys │        └──────┬───────┘
                  │ orgs     │               │
                  │ multi-   │               ▼
                  │ tenant   │   ┌──────────────────────┐
                  └──────────┘   │   storage · git       │
                                 │   forge · cici        │
                                 │   functions (goja JS) │
                                 │   · fetch (allowlist) │
                                 │   · db.collection()   │
                                 │   · request context   │
                                 │   · env · cron        │
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
│ 8 pages  │  │logs UI   │  │repo    │  │backoff   │
│ dark mode│  │domains   │  │listing │  │delivery  │
└──────────┘  └──────────┘  └────────┘  └──────────┘
```

## Component Catalog

| # | Component | Path | Runtime | Purpose |
|---|-----------|------|---------|---------|
| — | Kernel | `kernel/` | Core | Discovery, lifecycle, event bus, config merge |
| 1 | Proxy | `components/proxy/` | HTTP | HTTP server, routing, landing page |
| 2 | DB | `components/db/` | SQLite/PG | Database access, migrations, dual-driver |
| 3 | API | `components/api/` | HTTP | Auto CRUD, SQL endpoint, filter/sort, org scoping |
| 4 | Auth | `components/auth/` | HTTP | Register, login, JWT, Google OAuth, API keys, orgs, invites |
| 5 | Admin | `components/admin/` | HTTP | Embedded SPA server (8 pages, dark mode, responsive) |
| 6 | Storage | `components/storage/` | HTTP | File upload/download/delete, MIME detection |
| 7 | Git | `components/git/` | HTTP | Bare repo management, SSH clone |
| 8 | Forge | `components/forge/` | HTTP | Issues, labels, kanban, wiki |
| 9 | CICI | `components/cici/` | HTTP | Pipeline/workflow engine, git push triggers |
| 10 | Functions | `components/functions/` | HTTP | JS runtime (goja), fetch+db+env+request+cron injections |
| 11 | Realtime | `components/realtime/` | WS | WebSocket subscriptions, mutation event broadcast |
| 12 | Messaging | `components/messaging/` | HTTP | Email/SMS/Push/Webhook/Telegram providers |
| 13 | Deploy | `components/deploy/` | Process | Build/run/delete web apps, custom domains, TLS |
| 14 | Monitoring | `components/monitoring/` | HTTP | Metrics, logs, alerts, health, SSE events, processes |
| 15 | Sites | `components/sites/` | HTTP | Deploy from GitHub, build/request logs UI |
| 16 | GitHub | `components/github/` | HTTP | GitHub App auth, repo listing |
| 17 | Webhooks | `components/webhooks/` | HTTP | Outbound webhook delivery, retry, exponential backoff |
| 18 | Backup | `components/backup/` | CLI | Backup/restore, migration tooling |

## Release History

| Version | Epics | Highlights |
|---------|-------|------------|
| v1.0 | e01—e16 | Foundation: CLI, proxy, DB, auth, admin UI, all 16 components |
| v2.7.0 | e17—e30 | Multi-tenancy, orgs, API keys, webhooks, backup, observability, functions runtime 2.0, webhook/telegram, collections filter/sort |

## Event Flow

Components communicate exclusively through the kernel event bus. No direct imports between components.

```
proxy.onRequest ──► auth validates token/API key ──► api routes (org-scoped)

api.onMutation ──► realtime broadcasts ──► webhooks deliver
                ──► functions triggers (schedule or event)

git.onPush ──► cici runs workflow ──► deploy builds & serves

db.onBackup ──► monitoring logs
```

## External Services

| Service | Integration | Config |
|---------|------------|--------|
| Google OAuth | OAuth 2.0 relay | `--google-client-id`, `--google-client-secret` |
| GitHub | GitHub App (optional) | `--github-app-id`, `--github-app-private-key-path` |
| SMTP | Email sending | Messaging API, `--messaging-webhook-url` |
| PostgreSQL | Dual-driver DB | `--db-driver postgres`, `--db-dsn` |
| Telegram | Bot messaging | Messenger WebhookProvider, `--messaging-webhook-token` |
| Cron | Function scheduling | `--functions-schedule` flag, robfig/cron v3 |

## Key Design Decisions

| ADR | Decision | Status |
|-----|----------|--------|
| 0001 | SQLite + JSON Blob API | Accepted |
| 0002 | JWT + bcrypt Auth | Accepted |
| 0003 | GitHub App for Sites | Accepted |
| — | ECC pattern (no direct component imports) | Accepted |
| — | PostgreSQL dual-driver (modernc.org/sqlite + lib/pq) | Accepted (e18) |
| — | Multi-tenant org isolation at API/DB layer | Accepted (e23) |
| — | Functions runtime injections: fetch+db+env+request+cron | Accepted (e30) |
| — | Provider interface for pluggable messaging backends | Accepted (e12, e30) |

## CLI Reference

```bash
go run . serve [--port PORT] [--db PATH] [--db-driver DRIVER] [--db-dsn DSN]
               [--google-client-id ID] [--google-client-secret SECRET]
               [--messaging-webhook-url URL] [--messaging-webhook-token TOKEN]
               [--functions-schedule]

go run . version
go run . status
go run . components list
go run . backup --db FILE --output FILE
go run . migrate up|down|status
```

## Verification

```bash
# Architecture integrity: no direct component imports
grep -r "components/" kernel/ | grep -v "_test.go" | grep -v '//' || echo "No cross-component imports"

# All components implement kernel.Component
grep -l "kernel.Component" components/*/**.go | wc -l

# Event bus is sole communication channel
grep -c "EventBus" components/*/**.go

# Test suite
go test ./... && go vet ./...
```

→ **verify:** `go test -count=1 ./... && go build ./... && go vet ./...`
