# Design Feedback: Evolved v2 vs Codebase Gap Analysis

**Prototype file:** `BigBase Console - Evolved v2.html` (claude.ai/design project `ec1480a1`)
**Secondary file:** `BigBase Console - Project Scoped.html` (same project)
**IA spec:** `Component Spec - Information Architecture.html`
**Codebase:** `ui/src/` (React 19 + Vite + TS)
**Date:** 2026-06-29
**Purpose:** Pre-e52 alignment — catch divergence between Evolved v2 design and shipped code before implementation begins

---

## TL;DR

Three categories of misalignment:

1. **Prototype forgot existing features** — Forge, Realtime, Events are fully shipped in code but absent from the prototype's IA
2. **Prototype introduces new concepts not yet in code** — Project selector, Branches, two-zone nav, platform-level pages; these are the e52 intent
3. **Route/naming drift** — code uses `/deploy` for Sites; prototype uses `/sites`. Both IA specs use `/sites`.

---

## 1. Features in Codebase NOT in Prototype IA

These are fully shipped, working pages that the Evolved v2 and Project Scoped prototypes have silently dropped from the navigation. A user of the new design would lose access to them.

| Feature | Code route | Code page | Prototype status |
|---|---|---|---|
| **Forge** (Issues, Labels, Kanban, Wiki) | `/forge` | `ForgePage.tsx` | ❌ Not in any nav or IA spec |
| **Realtime** (WebSocket hub status) | `/realtime` | `RealtimePage.tsx` | ❌ Not in any nav or IA spec |
| **Events** (SSE event bus live stream) | `/events` | `EventsPage.tsx` | ❌ Not in any nav or IA spec |

**Forge** is the most significant gap. It's a full product surface (git-backed issue tracker + kanban board + wiki) that currently lives in DevOps in the sidebar. Both prototypes drop it entirely — not even listed in the IA's route table or cross-links.

**Recommendation for design:**
- Add Forge back under DevOps: `{ id: 'forge', label: 'Forge', icon: 'hammer' }` in the DevOps section
- Decide explicitly whether Realtime and Events belong in the DevOps/Monitoring section or are merged into Monitoring. Current code has them as separate routes; they could be tabs inside MonitoringPage.

---

## 2. Prototype Introduces New Concepts Not in Code (e52 scope)

These are intentional future features that the prototype previews but the codebase doesn't have yet. Listed so e52 work can be scoped properly.

### 2a. Two-Zone Navigation (Evolved v2 only)

The `BigBase Console - Evolved v2.html` introduces a **radical new sidebar model**:

```
PLATFORM ZONE (global)
  System monitoring
  Platform users
  Platform settings

PROJECT ZONE (per-project, accent-colored)
  Dashboard
  SQL Editor
  Branches       ← NEW concept
  Deployments
  Auth
  Functions
  Storage
  Settings
```

Current code has a **single flat nav** with named sections (Overview/Build/Data/Auth/Engage/DevOps). The two-zone model requires e52 (project isolation) before it can be built.

The `BigBase Console - Project Scoped.html` is the transitional state — same current sections but with a **Project selector dropdown** in the header. This is what e52 should target, not the full two-zone redesign.

**Recommendation for design:** Clarify which file is the e52 target:
- `Project Scoped.html` → transitional (project selector in header, existing nav sections) — buildable in e52
- `Evolved v2.html` → post-e52 long-term vision — NOT buildable until project isolation is complete

### 2b. Branches Page (Evolved v2 only)

The Evolved v2 prototype has a `Branches` page in the project zone showing:
- Branch list (name, base, status, commits ahead, last updated)
- Create branch form (name + base selector)
- Protected branch badge
- Merge / Delete actions

**Current codebase has NO branches concept.** Database branching is planned for e58 (Native Feature Port — SQL-over-HTTP, Better Auth, MCP Tools, Neon native port).

**Recommendation for design:** Flag the Branches page as e58 scope, not e52. e52 should only add the project selector + project-scoped data isolation, not full branching.

### 2c. Platform-Level Pages (Evolved v2 only)

The Evolved v2 global zone introduces:
- `Platform users` — all users across all projects (roles: superadmin, admin, member, viewer; shows `projects` count column)
- `Platform settings` — global BigBase instance config (SMTP, OAuth providers, rate limits, storage)

Current code has:
- `UsersPage` (`/users`) — **app-level users** (people who signed up via Auth), not platform admins
- `SettingsPage` (`/settings`) — mix of account + workspace + billing

These are different concepts that the design conflates:
- **Platform users** = operators/admins of the BigBase instance (e64 scope)
- **Auth users** = end-users who registered through the Auth API (existing `/users`)

**Recommendation for design:** Add a note distinguishing "platform members" (e64) from "auth users" (existing). The current `/users` page should stay as-is for auth user management. A new `Platform users` surface comes with e64.

### 2d. Project Selector in Header

Both `Project Scoped.html` and `Evolved v2.html` show a header dropdown:
```
Project ▾
  my-db
  staging
  analytics-db
  + Create project
```

Current code has **no project concept** — everything is single-tenant org-scoped. The project selector is the core deliverable of e52 (Project Scoping — Backend + UI).

**Recommendation for design:** The project selector is correctly in scope for e52 + e57 (e57 = Project Scoping Admin UI). Confirm the header placement is final before e57 implementation starts.

---

## 3. Route / Naming Drift

| Surface | Current code route | Prototype route | IA spec route |
|---|---|---|---|
| Sites list | `/deploy` | `sites` | `sites` |
| New site | `/deploy/new` | `sites/new` | `sites/new` |
| Site detail | `/deploy/:siteId` | `sites/:id` | `sites/:id` |

**The codebase uses `/deploy` but every spec and prototype says `/sites`.** This is a pure naming inconsistency from when the feature was built (epic e13 named it "Deploy"). The page is literally called `DeployPage.tsx` and `SiteDetailPage.tsx`.

**Recommendation for code (not design):** Rename routes from `/deploy` to `/sites` as part of e52 cleanup. This is a routing change only — no functional impact. Page files can stay named `DeployPage.tsx` temporarily.

---

## 4. IA Spec vs Current Sidebar — Full Diff

The `Component Spec - Information Architecture.html` defines the confirmed navigation tree. Current code status:

| IA item | Route | Code status | Notes |
|---|---|---|---|
| Overview / Dashboard | `/` | ✅ Shipped | |
| Build / Sites | `/deploy` | ✅ Shipped | Route name wrong (see §3) |
| Build / Functions | `/functions` | ✅ Shipped | |
| Data / Data Studio | `/data` | ✅ Shipped | |
| Data / SQL Editor | `/sql` | ✅ Shipped | |
| Data / Storage | `/storage` | ✅ Shipped | |
| Auth / Users | `/users` | ✅ Shipped | |
| Engage / Messaging | `/messaging` | ✅ Shipped | |
| DevOps / Git Repos | `/repos` | ✅ Shipped | |
| DevOps / CI/CD | `/cici` | ✅ Shipped | |
| DevOps / Monitoring | `/monitoring` | ✅ Shipped | |
| Footer / Settings | `/settings` | ✅ Shipped | |
| — | `/forge` | ✅ Shipped | **MISSING from IA** |
| — | `/realtime` | ✅ Shipped | **MISSING from IA** |
| — | `/events` | ✅ Shipped | **MISSING from IA** |

---

## 5. Recommended Feedback to Claude.ai/Design

These are the specific changes the design should incorporate before e52 implementation starts:

### Must Fix (blocks accurate implementation)

1. **Add Forge back to DevOps section** in all prototype files. Route: `/forge`, icon: `hammer`, label: `Forge`. Forge has issues, kanban, labels, wiki — it's fully shipped and used.

2. **Decide Realtime + Events fate** — either add them as DevOps nav items or document that they merge into Monitoring as tabs. Currently invisible in the design but present in code.

3. **Rename `/deploy` → `/sites`** in the prototype-to-code mapping document so e52 implementation starts with the right routes.

4. **Label Branches page as e58 / future** — not e52. Add a "future" badge or separate it into a second prototype file so e52 implementors don't accidentally try to build branching.

5. **Clarify two-zone nav target version** — is `Evolved v2.html` the e52 target or the long-term vision? `Project Scoped.html` is more realistic for e52. This ambiguity will cause scope creep.

### Should Fix (design quality)

6. **Distinguish auth users vs platform members** — the `Platform users` page in Evolved v2 shows operator roles (superadmin, admin, member, viewer with project counts). The existing `/users` page shows end-users registered via Auth API. They are different data sets. Add a callout in the design explaining the distinction.

7. **Settings split** — the current single `/settings` has account + workspace + billing. The two-zone nav splits this into `Settings (project)` in the project zone and `Platform settings` in the global zone. Confirm what each contains so Settings doesn't get split incorrectly in e52.

### Won't Fix / Code Adapts

8. **IA spec routes already use `/sites`** — code needs to catch up, not design.

---

## 6. What's Solid (No Changes Needed)

- Design tokens in `ui/src/tokens/` and `ui/src/styles/` are already in sync with both prototypes (e51 done)
- Component library (Button, Badge, Card, Input, AppShell, Sidebar, etc.) matches the prototype's `react-stubs/`
- Dark mode via `data-theme="dark"` is consistent across prototype and code
- Sidebar section order (Overview → Build → Data → Auth → Engage → DevOps) matches the IA spec and current code
- Theme/accent system (monthly accent colors) matches `ui/src/tokens/tokens.ts`
- Monitoring page is in DevOps in both prototype and code ✅
