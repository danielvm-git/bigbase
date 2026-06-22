# BigBase — Domain Context & Architecture

> **⚠️ SUPERSEDED** — This document is no longer the authoritative architecture reference.
> The canonical source is [`specs/tech-architecture/tech-stack.md`](../tech-architecture/tech-stack.md)
> (version 2.13.0). This file is preserved for historical reference; do not edit.
>
> **Key drift**: TECH_STACK_LATEST.md lists 16 components; the actual codebase has 19
> (MCP, Webhooks, Backup were added). The canonical doc also covers PostgreSQL
> dual-driver, orgs/multi-tenant, Google OAuth, New Relic, and MCP server.

## Overview

BigBase is a single-binary, open-source Backend-as-a-Service (BaaS) platform built
with Go. It uses the **Entity-Component-Construct (ECC)** pattern: a lightweight
kernel discovers, starts, and connects independent components via an event bus.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| Database | SQLite (via `modernc.org/sqlite`, zero-CGO) |
| Auth | bcrypt + HS256 JWT |
| Functions runtime | goja (pure Go JavaScript) |
| Admin UI | React 19 + Vite + TypeScript, embedded via `//go:embed` |
| Deployment | Child processes on host, port-allocation |
| CI/CD | GitHub Actions + semantic-release |
| Hosting | Contabo Cloud VPS (current), Terraform-defined |

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        Kernel                                 │
│  • Component discovery & registration                         │
│  • Lifecycle: Init → Start → Stop (topo-sorted)              │
│  • Event bus: hook-based pub/sub, priority-ordered dispatch  │
│  • Config merge: defaults + user overrides                   │
└────────────────────────┬─────────────────────────────────────┘
                         │ dispatches events
    ┌────────────────────┼────────────────────┐
    ▼                    ▼                    ▼
┌─────────┐       ┌──────────┐        ┌──────────────┐
│ proxy   │       │   auth   │        │   api/db     │
│ HTTP    │◄─────►│ JWT      │◄──────►│ CRUD + SQL   │
│ router  │       │ bcrypt   │        │              │
└─────────┘       │ OAuth    │        └──────┬───────┘
                  └──────────┘               │
                                            ▼
┌─────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐
│storage  │  │  git     │  │ forge  │  │functions │
│files    │  │ bare repos│  │issues  │  │goja JS   │
└─────────┘  └──────────┘  │kanban  │  └──────────┘
                           └────────┘
┌──────────┐  ┌──────────┐  ┌────────┐  ┌──────────┐
│realtime  │  │messaging │  │ deploy │  │monitoring│
│WebSocket │  │email/SMS │  │process │  │metrics   │
└──────────┘  │push      │  │mgmt    │  │logs      │
              └──────────┘  └────────┘  │alerts    │
                                        └──────────┘
┌──────────┐  ┌──────────┐  ┌────────┐
│  admin   │  │ github   │  │ sites  │
│ UI (SPA) │  │ App auth │  │deploy  │
└──────────┘  └──────────┘  │from GH │
                            └────────┘
```

## Component Catalog

All 16 components implement the `kernel.Component` interface.

| Component | Path | Runtime | Tests | Purpose |
|-----------|------|---------|-------|---------|
| Kernel | `kernel/` | Core | ✅ | Discovery, lifecycle, event bus |
| Proxy | `components/proxy/` | HTTP | ✅ | HTTP server, routing, landing page |
| DB | `components/db/` | SQLite | ✅ | Database access, migrations |
| API | `components/api/` | HTTP | ✅ | Auto CRUD, SQL endpoint |
| Auth | `components/auth/` | HTTP | ✅ | Register, login, JWT, Google OAuth |
| Admin | `components/admin/` | HTTP | ✅ | Embedded SPA server |
| Storage | `components/storage/` | HTTP | ✅ | File upload/download/delete |
| Git | `components/git/` | HTTP | ✅ | Bare repo management |
| Forge | `components/forge/` | HTTP | ✅ | Issues, labels, kanban, wiki |
| CICI | `components/cici/` | HTTP | ✅ | Pipeline/workflow engine |
| Functions | `components/functions/` | HTTP | ✅ | JS runtime, CRUD, execution |
| Realtime | `components/realtime/` | WS | ✅ | WebSocket subscriptions, event broadcast |
| Messaging | `components/messaging/` | HTTP | ✅ | Email, SMS, push endpoints |
| Deploy | `components/deploy/` | Process | ✅ | Build/run web apps from repos |
| Monitoring | `components/monitoring/` | HTTP | ✅ | Metrics, logs, alerts, health |
| Sites | `components/sites/` | HTTP | ✅ | Deploy from GitHub wizard |
| GitHub | `components/github/` | HTTP | ✅ | GitHub App auth, repo listing |

## Event Flow

Components communicate through the kernel event bus. Events are dispatched
by priority order to all subscribers.

```
proxy.onRequest ──► auth validates token ──► api routes
                         │
db.onMutation ──► realtime broadcasts ──► functions triggers
                    │
                    └──► messaging sends notification

git.onPush ──► cici runs workflow ──► deploy previews
```

## External Services

| Service | Integration | Config |
|---------|------------|--------|
| Google OAuth | OAuth 2.0 relay | `--google-client-id`, `--google-client-secret` |
| GitHub | GitHub App (optional) | `--github-app-id`, `--github-app-private-key-path` |
| SMTP (Messaging) | Email sending | Configured via messaging API |
| Contabo VPS | Production hosting | Terraform + GitHub Actions |
| SQLite | Primary database | `--db` flag (default: `bigbase.db`) |

## Deployment Topology

```
GitHub ──push──► GitHub Actions
                    │
               semantic-release
                    │
               deploy to VPS
                    │
               /opt/bigbase/
               ├── bin/bigbase
               ├── data/bigbase.db
               ├── releases/
               └── backups/
```

## Key Design Decisions

See `specs/adr/` for full ADRs:

- **ADR 0001**: SQLite + JSON Blob API (accepted)
- **ADR 0002**: JWT + bcrypt Auth (accepted)
- **ADR 0003**: GitHub App for Sites (accepted)
