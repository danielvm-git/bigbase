# 016 — Developer Onboarding Journey & Event Bus Visualizer

**status:** proposed  
**verify:** See slice verification blocks below

## Problem
When a developer first logs into the BigBase Admin UI, they are met with empty dashboard stats (blank slots for Git Repos, Deployments, Storage, and Messages) which creates initial friction. Furthermore, developers lack a clear visual representation of how the underlying Entity-Component-Construct (ECC) Event Bus binds components together (e.g., how a database write triggers a function or how a git push triggers a deployment).

This epic delivers an interactive developer onboarding journey, a live Event Bus Visualizer on the dashboard, and a 1-click template scaffolding system to quickly seed databases, repositories, and serverless functions, helping developers understand the system's capabilities immediately.

## Slices

| Slice | Goal | Backend |
|-------|------|---------|
| **A** | Onboarding Checklist & Layout UI | None (UI-first mockup) |
| **B** | Scaffolding API & Quick Seeders | `/api/scaffold` endpoints |
| **C** | Event Bus Live Streams | `/api/monitoring/events` SSE/WS |
| **D** | Event Bus Visualizer UI | React visual hook canvas |

## Routes (Admin UI & API)

### Admin UI
| Route | Page |
|-------|------|
| `#/` | Dashboard (renders Onboarding Checklist and Event Visualizer) |
| `#/data` | Data Studio (shows "Seed Sample Data" button on empty state) |
| `#/functions` | Functions (shows template selector cards) |

### API Endpoint Group
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/scaffold/db` | Seeds sample tables and records (e.g., Todo app schema) |
| POST | `/api/scaffold/repo` | Scaffolds a new git repository with static site or function templates |
| POST | `/api/scaffold/function` | Provisions a template runtime function (JS or Go) |
| GET | `/api/monitoring/events` | Stream of live Event Bus hook occurrences |

---

## Slice A Verification (UI Onboarding Checklist)
```bash
cd ui && npm run dev
# http://localhost:5173/admin/
```
**Acceptance:**
- New "Developer Onboarding Checklist" card appears on empty dashboard.
- Clicking checklist items redirects users to the correct workspace views (Data Studio, Repos, Functions).
- Smooth progress bar indicates checklist completion status.

---

## Slice B Verification (Backend Scaffolding & Seeders)
```bash
go test ./components/api/... -run TestScaffold -v
# Run the server and check seeder endpoint
curl -X POST http://localhost:9999/api/scaffold/db
```
**Acceptance:**
- Seeder successfully checks connection drivers (SQLite/Postgres).
- Creates target schema tables and generates random seed data.
- Returns JSON confirming seeded entities.

---

## Slice C & D Verification (Event Bus Visualizer)
```bash
go test ./components/monitoring/...
# Perform an action (e.g. trigger database write)
curl -X POST http://localhost:9999/api/collections/tasks -d '{"title":"Test task"}'
```
**Acceptance:**
- Visualizer renders component nodes (`proxy`, `db`, `git`, `cici`, `functions`, `messaging`, `deploy`).
- Event dispatch (e.g. `onMutation`) triggers visual signal/line connection animation between the `db` node and the handling consumer nodes (`realtime`, `functions`).
