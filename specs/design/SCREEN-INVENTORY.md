# BigBase Screen Inventory

Living map of every screen: its route, owning epic, and prototype status.
Updated before each epic implementation starts.

**Last updated:** 2026-06-29  
**Prototype project:** claude.ai/design (BigBase Console — Evolved v2 / Design C)  
**Design System project:** `502492b2-4dcc-4024-9e7a-26baa7943ca7`

---

## Current App (Shipped — 18 screens)

| Screen | Route | Page file | Epic | Prototype | Notes |
|--------|-------|-----------|------|-----------|-------|
| Global Dashboard | `/` | DashboardPage.tsx | e05/e17 | ✅ | Metrics, onboarding, health |
| Sites List | `/sites` | DeployPage.tsx | e15 | ⚠️ route drift | Code still uses `/deploy`; e60 renames |
| Site Detail | `/sites/:siteId` | SiteDetailPage.tsx | e15 | ⚠️ route drift | 5 tabs: Overview/Deploys/Domains/Logs/Settings |
| Create Site | `/sites/new` | CreateSitePage.tsx | e15 | ⚠️ route drift | GitHub OAuth wizard |
| SQL Editor | `/sql` | SqlEditorPage.tsx | e03 | ✅ | Will be replaced by scoped version in e58 |
| Data Studio | `/data` | DataStudioPage.tsx | e03 | ✅ | Collection browser, record editor |
| Storage | `/storage` | StoragePage.tsx | e06 | ✅ | Bucket browser, file list, upload |
| Functions List | `/functions` | FunctionsPage.tsx | e10 | ✅ | |
| Function Detail | `/functions/:id` | FunctionDetailPage.tsx | e10 | ✅ | Source, logs, triggers |
| Users | `/users` | UsersPage.tsx | e04 | ✅ | Auth users (not platform members) |
| Messaging | `/messaging` | MessagingPage.tsx | e12 | ✅ | |
| Messaging Detail | `/messaging/:id` | MessagingDetailPage.tsx | e12 | ✅ | Template editor |
| Git Repos | `/repos` | GitReposPage.tsx | e07 | ✅ | |
| CI/CD | `/cici` | CiciPage.tsx | e09 | ✅ | Pipeline runs |
| Monitoring | `/monitoring` | MonitoringPage.tsx | e14 | ✅ | Health grid, metrics, activity |
| Settings | `/settings` | SettingsPage.tsx | e17 | ✅ | Theme, accent, auth |
| **Forge** | `/forge` | ForgePage.tsx | e08 | ❌ MISSING | Issues, kanban, labels, wiki. Fix: Step 2 brief |
| **Realtime** | `/realtime` | RealtimePage.tsx | e11 | ❌ MISSING | WebSocket channel hub. Fix: Step 2 brief |
| **Events** | `/events` | EventsPage.tsx | e11 | ❌ MISSING | SSE event bus live stream. Fix: Step 2 brief |
| Login | `/login` | LoginPage.tsx | e04 | ✅ | Email/password + Google OAuth |

---

## Post-e57/e58 Screens (Design C — Plan Ready)

Source: `specs/DESIGN_C_HANDOFF.md`

| Screen | Route | Epic | Prototype | Notes |
|--------|-------|------|-----------|-------|
| Project Selector + Branch Selector | header overlay | e57s05 / e58 | ❌ needs brief | Dropdown in top bar. Prerequisite to all project-scoped screens |
| Project Dashboard | `/project/:id` | e58 | ❌ needs brief | Stats, connection string, recent queries, branch cards |
| Unified SQL + Data | `/project/:id/sql/:branch?` | e57/e58 | ❌ needs brief | 3-panel: schema tree / editor / results |
| Project Settings | `/project/:id/settings` | e58 | ❌ needs brief | 5 tabs: General / DB / Env Vars / Branches / Danger Zone |
| Branches | `/project/:id/branches` | e59s02 | ❌ future | Mark as "coming soon" in Design C prototype |
| Projects List (global) | `/projects` (or global dashboard) | e58 | ❌ needs brief | Shown when no project selected |

---

## Future Epic Screens (Planned — Not Yet Designed)

| Screen | Route | Epic | Prototype | Design Notes |
|--------|-------|------|-----------|--------------|
| Secrets Manager | `/project/:id/secrets` (or Settings tab) | e61 | — | Env var editor pattern, masking, rotation |
| Usage Dashboard | `/admin/usage` or sidebar item | e63 | — | Resource tracking: compute, storage, bandwidth, queries |
| Schema Designer | `/data/schema` or tab on Data Studio | e64 | — | Visual ERD + table CRUD editor |
| Preview Environments | `/sites/:id/previews` | e65 | — | PR-linked preview deployments |
| Platform Users | `/admin/platform/users` | e66 | — | Operators/admins (distinct from Auth users) |
| Platform Settings | `/admin/platform/settings` | e66 | — | Global instance config: SMTP, OAuth providers, rate limits |
| Team/Org Invites | `/admin/platform/invites` | e66 | — | Role assignment: admin/member/viewer |

---

## IA Change Summary (e57 → e58)

**Current sidebar (flat):**
```
Overview  →  Dashboard
Build     →  Sites, Functions
Data      →  Data Studio, SQL Editor, Storage
Auth      →  Users
Engage    →  Messaging
DevOps    →  Git Repos, CI/CD, Monitoring, Forge, Realtime
Footer    →  Settings
```

**Post-e58 sidebar (two-zone):**
```
PROJECT ZONE (visible when project selected, accent-colored)
  Dashboard
  SQL Editor       ← project/branch scoped
  Branches         ← future (e59s02)
  Settings         ← project settings only

GLOBAL ZONE (always visible)
  Sites, Functions
  Data Studio, Storage
  Users
  Messaging
  Git Repos, CI/CD, Monitoring, Forge, Realtime
Footer: Platform Settings (e66) | Account Settings
```

**Header bar (new in e58):**
```
[BigBase logo]  [Project ▾]  [Branch ▾]  |  Page Title       [Avatar] [Logout]
```

---

## Update Protocol

Before each epic implementation:

1. Claude Code generates a targeted update brief → saved to `specs/design/UPDATE-BRIEF-eXX.md`
2. User posts brief to claude.ai/design prototype project
3. Claude Design updates the affected screens
4. User exports updated HTML → stored in `specs/archive/bigbase-prototype-3/`
5. Claude Code generates gap analysis → `specs/design-feedback/eXX-VS-PROTOTYPE.md`
6. Implementation begins

**Design System sync** (after Track B component previews change):
```
DesignSync.finalize_plan → DesignSync.write_files
```
Run after any epic that ships new or changed components.
