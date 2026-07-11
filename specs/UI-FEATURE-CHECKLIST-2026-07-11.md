# UI Feature Checklist — BigBase Admin Console

Generated 2026-07-11 from completed epics (e01–e56, e67, e68, e71, e72 per `specs/execution-status.yaml`). Purpose: compare against the design prototype (`BigBase Design System` / `ui_kits/admin-console`) to verify coverage before finalizing.

Epics with no user-facing UI (pure backend/API/CLI/infra/security) are listed at the bottom for completeness — checked but out of scope for prototype comparison.

---

## Site Tree (current admin console — `ui/src/App.tsx`)

```
/                          → DashboardPage
/login                     → LoginPage
/data                      → DataStudioPage
/sql                       → SqlEditorPage
/users                     → UsersPage
/repos                     → GitReposPage
/deploy                    → DeployPage
  /deploy/new              → CreateSitePage
  /deploy/:siteId          → SiteDetailPage
/messaging                 → MessagingPage
  /messaging/:id           → MessagingDetailPage
/storage                   → StoragePage
/functions                 → FunctionsPage
  /functions/:id           → FunctionDetailPage
  /functions/:id/logs      → (redirect → FunctionDetailPage logs tab)
/forge                     → ForgePage
/cici                      → CiciPage
/monitoring                → MonitoringPage
/events                    → EventsPage
/realtime                  → RealtimePage
/settings                  → SettingsPage
*                          → NotFoundPage
```

**Not yet a routed page** (feature exists but no dedicated page found in `ui/src/pages/`):
- `/admin/auth` — live auth-component preview w/ framework tabs (from e34)
- Org usage/management screen (from e23 — org CRUD/usage charts)

---

## Checklist by Epic (UI-relevant only)

Check off each item after comparing it against the prototype.

### Foundational (e01–e16)

- [ ] **e02 — Landing page(s)** *(new/unmapped — public site, not admin console)*
  - [ ] Marketing/landing page at `/`
  - [ ] Second commercial landing page variant (from e07)
- [ ] **e04 — Auth: Login/Register** → `LoginPage`
  - [ ] Register form
  - [ ] Login form (JWT-based)
- [ ] **e05 — Admin dashboard shell** → `DashboardPage`, `DataStudioPage`, Sidebar/Layout
  - [ ] React SPA shell at `/admin/`
  - [ ] Collection list + record browser
- [ ] **e06 — Storage UI** → `StoragePage`
  - [ ] File upload/download/delete
- [ ] **e07 — Git repo UI** → `GitReposPage`
  - [ ] Repo management screen
- [ ] **e08 — Forge (issues/kanban/wiki)** → `ForgePage`
  - [ ] Issue tracker
  - [ ] Kanban board
  - [ ] Wiki UI
- [ ] **e09 — CI/CD pipelines** → `CiciPage`
  - [ ] Pipeline/workflow run history view
- [ ] **e10 — Functions runtime** → `FunctionsPage`
  - [ ] Function CRUD + run UI
- [ ] **e11 — Realtime inspector** → `RealtimePage`
  - [ ] Connections/subscriptions inspector
- [ ] **e12 — Messaging** → `MessagingPage`
  - [ ] Send message UI
  - [ ] Outbound log
- [ ] **e13 — Deploy (build & run)** → `DeployPage`
  - [ ] Deploy trigger UI
  - [ ] App-type detection surfaced in UI
- [ ] **e14 — Monitoring dashboard** → `MonitoringPage`
  - [ ] Metrics dashboard
- [ ] **e15 — Sites: deploy from GitHub** → `CreateSitePage`, `DeployPage`
  - [ ] GitHub connect flow
  - [ ] One-click deploy from repo picker

### e17 — Enhanced Admin UI (18 stories — large surface)

- [ ] `RealtimePage` — realtime inspector page
- [ ] `FunctionsPage` — function logs viewer
- [ ] `StoragePage` — grid/list toggle, preview modal, folder tree, drag-drop upload
- [ ] `DeployPage` / `CiciPage` — deploy detail page + CI/CD pipeline viewer with workflow DAG
- [ ] `DashboardPage` — charts, dark mode, sidebar, toasts, quick actions
- [ ] `DeployPage` / `CiciPage` — `StreamLog` and `EnvVarEditor` shared components
- [ ] `DashboardPage` — `MetricCard`, `RequestChart`, `ComponentHealthGrid`, `QuickActions` primitives
- [ ] `DashboardPage` — user card, health banner, quick actions, stat cards polish
- [ ] `SiteDetailPage` — status timeline + redeploy button
- [ ] `UsersPage` — users list w/ delete confirmation
- [ ] Sidebar/Layout — IA shell rebuild (sidebar groups: Overview/Build/Data/Auth/Engage/DevOps), Settings nav stub
- [ ] Sidebar/Layout — 12 accent themes + picker in sidebar footer
- [ ] Shared components — `Modal`, `Breadcrumb`, ghost Button variant, focus-visible states
- [ ] `FunctionDetailPage` — functions card grid + tabs (code/triggers/variables/logs)
- [ ] `MessagingPage` / `MessagingDetailPage` — template list + editor/preview
- [ ] `DataStudioPage` / `SqlEditorPage` — Data/Schema toggle, column Add/Edit/Delete, "Query this" link
- [ ] `SettingsPage` / `LoginPage` — Account/Workspace/Billing tab stubs; forgot-password UI
- [ ] Cross-cutting — responsive breakpoints (1024px/375px), dashboard cross-links, ARIA labels

### Platform & DX (e20–e30)

- [ ] **e20 — Env var editor (precursor) + custom domains** → `SettingsPage`, `SiteDetailPage`
  - [ ] Admin UI editor for env vars (early version)
- [ ] **e22 — Developer experience** → `DashboardPage`, `EventsPage`
  - [ ] Onboarding checklist card w/ progress
  - [ ] Event Bus visualizer canvas (`#/events`)
  - [ ] Sample-app "Deploy" buttons
  - [ ] Interactive tutorial overlay
- [ ] **e23 — Multi-tenancy / orgs** *(new/unmapped — no dedicated OrgsPage yet)*
  - [ ] Org usage tracking charts
- [ ] **e25 — Hardware monitoring** → `MonitoringPage`
  - [ ] CPU/memory charts
  - [ ] Disk gauge
  - [ ] Network sparkline
  - [ ] Alert rule config UI
- [ ] **e26 — Site build logs** → `SiteDetailPage`
  - [ ] "Logs" tab — terminal-style live/streaming build log viewer
- [ ] **e27 — Site request logs** → `SiteDetailPage`
  - [ ] "Request Logs" tab (filterable: method/path/status/duration)
  - [ ] "Runtime" sub-tab for stdout/stderr
- [ ] **e28 — Delete deployment** → `SiteDetailPage`
  - [ ] Per-row "Delete" button + confirmation on deployments table
- [ ] **e30 — Collection filter/sort** → `DataStudioPage`
  - [ ] Filter/sort inputs + sort dropdown on collection list

### Auth UI (e34)

- [ ] **e34 — Auth UI components (multi-framework)** *(new/unmapped — `/admin/auth` not yet routed)*
  - [ ] `<SignIn>`, `<SignUp>`, `<UserButton>`, `<AuthView>` components (React/Vue/Svelte)
  - [ ] `/admin/auth` live-preview page with framework tabs + copy-paste snippets

### Deploy lifecycle (e36–e45)

- [ ] **e36 — Delete deployed site** → `DeployPage` (SiteCard), `SiteDetailPage`
  - [ ] Delete action on Sites grid card
  - [ ] "Danger Zone" delete on site detail w/ confirmation
- [ ] **e39 — Streaming build logs** → `SiteDetailPage`
  - [ ] WebSocket live log streaming
  - [ ] Status timeline with animated dots / "Live" badge
  - [ ] `TerminalLogViewer` toolbar (search, copy, ANSI colors)
- [ ] **e40 — App manifest (bigbase.yaml)** → `SiteDetailPage`
  - [ ] "Manifest" tab — read-only + inline YAML editor with validation/save
- [ ] **e41 — Environment variables UI** → `SiteDetailPage`
  - [ ] "Environment Variables" tab — table (masked values, build/runtime toggles)
  - [ ] Add/edit/delete env vars
  - [ ] Bulk `.env` import/export
- [ ] **e42 — Build cache** → `SiteDetailPage`, `DeployPage`
  - [ ] Per-site "Cache" tab (entries, size, hit count, clear button)
  - [ ] Global "Build Cache" panel on Deploy page (usage, clear all, prune, max-size setting)
- [ ] **e44 — One-click rollback** → `SiteDetailPage`
  - [ ] "Rollback to previous version" button + confirmation
  - [ ] Rollback history timeline
- [ ] **e45 — Zero-downtime drain** → `SiteDetailPage`
  - [ ] `draining` / `stopped` states in status timeline (pulse animation)
  - [ ] New Badge variants for drain states

### e51 — UI Design System & Component Library v2 (7 stories)

- [ ] Design tokens as TS constants + `prefers-reduced-motion` support
- [ ] Core components: `Checkbox`, `Spinner`, `Switch`, `Select`, `Label`, `Table`
- [ ] Layout: `AppShell`, `Sidebar`, `SidebarSection`, `SidebarItem`, `AppFooter`, mobile slide-over drawer
- [ ] Page templates: `Page`, `ListPage`, `DetailPage`, `SettingsPage` template
- [ ] Accessibility & responsive audit (axe-core, breakpoint fixes)
- [ ] Extended components: `Tooltip`, `DropdownMenu`, `Dialog`, `FileUpload`, `CopyButton`, `TagInput`, `Alert`
- [ ] Data-viz/editor: `AreaChart`, `CodeBlock`, `JsonTree`, enhanced `BarGauge`

---

## Checked — No UI (backend/API/CLI/infra/security only)

For completeness; these were reviewed and confirmed to have no admin-console surface, so they're out of scope for the prototype comparison.

| Epic | Reason |
|---|---|
| e01 | CLI binary only |
| e03 | DB CRUD REST API |
| e16 | Onboarding journey test + docs, no distinct screen |
| e18 | DB driver is a config flag |
| e19 | Security hardening (rate limit, verify/reset flows — backend) |
| e21 | Test/quality hardening only |
| e24 | Observability wiring (logs, metrics export) |
| e29 | Security vuln fixes (OAuth CSRF, org isolation, admin gates) |
| e31 | SPA auth (CORS, redirect, logout) — backend |
| e32 | Passwordless auth endpoints (screen built in e17s17) |
| e33 | Client SDK packages, not admin console |
| e35 | Phone OTP / anonymous JWT / OAuth popup — API/SDK only |
| e37 | SvelteKit build-output detection logic |
| e38 | MCP server for AI agents, no admin UI |
| e43 | Absorbed into e53, no separate stories |
| e47 | Rate limiter wiring |
| e48 | Live surface hardening (headers, CI scanning) |
| e49 | Auth hardening (JWT claims, Host header, path traversal) |
| e50 | JWT secret & token lifecycle — backend config |
| e53 | Deploy process supervisor engine |
| e54 | New Relic integration |
| e55 | PR-agent code review CI workflow |
| e56 | OTP persistence & session audit (explicitly no admin UI) |
| e67 | MCP provisioning tools (explicitly no UI, future epic) |
| e68 | DB connection string env var injection |
| e71 | Route auth policy (explicitly "UI future epic") |
| e72 (mcp-platform-auth) | Bearer middleware, scope-gated provisioning |
| e72 (monitoring-enhancements) | AI incident diagnosis via REST only (timeline UI explicitly out of scope) |

---

## Notes for Prototype Comparison

1. **e34's `/admin/auth` page and e23's org usage screen** have no current route in `ui/src/App.tsx` — verify whether the prototype includes them as net-new screens or whether they were deprioritized.
2. **e72 has an ID collision** — two separate completed epics (`mcp-platform-auth` and `monitoring-enhancements`) both claim `e72`. Monitoring-enhancements produced data (correlated timeline, pipeline stage timing) that has no UI yet — worth flagging as a prototype opportunity even though the epic itself shipped API-only.
3. **e17 and e51 are the two largest UI epics** — most of the current admin console's visual language and shared components trace back to these two. Prioritize comparing prototype fidelity against these first.
