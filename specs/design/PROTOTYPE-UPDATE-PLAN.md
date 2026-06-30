# Prototype Update Plan — e57 → e65

**Date:** 2026-06-30  
**Prototype:** `BigBase Console.html` in project `ec1480a1`  
**Protocol:** For each epic with UI impact → write UPDATE-BRIEF → post to claude.ai/design → prototype updated → implementation begins.

---

## Epic → Prototype Impact Matrix

| Epic | Name | Prototype Impact | Brief | Priority |
|------|------|-----------------|-------|----------|
| e57 | Project Scoping Backend | None — backend only | — | — |
| **e58** | Project Scoping UI | **MAJOR** — new sidebar zones, header bar, 5 new screens | UPDATE-BRIEF-e58.md | P0 |
| e59 | Neon Native Port | Minor — Branches screen update (already drafted in e58) | folded into e58 | P1 |
| **e60** | Route Rename /deploy → /sites | Trivial — update sidebar label + breadcrumbs in prototype | UPDATE-BRIEF-e60.md | P0 (quick) |
| **e61** | Secrets Manager | New screen: Secrets tab in Project Settings | UPDATE-BRIEF-e61.md | P1 |
| e62 | CSP Headers | None — pure backend | — | — |
| **e63** | Usage Dashboard | New screen: Usage page, sidebar entry | UPDATE-BRIEF-e63.md | P1 |
| **e64** | Schema Designer | New screen: Visual ERD in Data Studio | UPDATE-BRIEF-e64.md | P2 |
| **e65** | Preview Environments | New tab in Site Detail: Previews | UPDATE-BRIEF-e65.md | P2 |

---

## Sequencing

### Now (before e58 implementation)
1. **UPDATE-BRIEF-e60** — apply immediately, 5-minute prototype edit (route label only)
2. **UPDATE-BRIEF-e58** — the IA restructure. Must be prototyped and reviewed before any e58 code is written

### After e58 ships
3. **UPDATE-BRIEF-e61** — Secrets screen inside Project Settings
4. **UPDATE-BRIEF-e63** — Usage Dashboard

### Before e64 / e65 implementation
5. **UPDATE-BRIEF-e64** — Schema Designer tab
6. **UPDATE-BRIEF-e65** — Previews tab on Site Detail

---

## What Changes in e58 (Design C)

This is the only epic that restructures the global IA. Every other epic adds a single screen or tab.

### Sidebar: 2-zone layout
```
PROJECT ZONE (accent-colored header, visible when project selected)
  [project name]  ← project selector trigger
  ○ Dashboard
  ○ SQL Editor
  ○ Branches
  ○ Settings

GLOBAL ZONE (always visible, unchanged)
  Overview  →  Dashboard (global)
  Build     →  Sites, Functions
  Data      →  Data Studio, Storage
  Auth      →  Users
  Engage    →  Messaging
  DevOps    →  Git Repos, CI/CD, Monitoring, Forge, Realtime, Events
  ─────
  Appearance, Settings, User, Logout
```

### Header bar (new element between logo and content)
```
[BigBase logo]  [Project: my-db ▾]  [Branch: main ▾]  |  Page title        [Avatar]
```
Visible only when inside a project route (`/project/:id/*`). Hidden on global pages.

### New screens (5)
1. **Projects List** — `/projects` — global dashboard for all projects
2. **Project Dashboard** — `/project/:id` — stats, connection string, recent queries, branch cards
3. **SQL Editor** — `/project/:id/sql/:branch?` — 3-panel (schema tree / editor / results)
4. **Branches** — `/project/:id/branches` — branch cards, create/switch/delete
5. **Project Settings** — `/project/:id/settings` — 4 tabs: General / Database / Env Vars / Danger Zone

### Screens unchanged
All 20 screens from PROTOTYPE-SPEC-v1.md stay exactly as-is. The restructure adds new routes; it does not modify existing global pages.

---

## Brief Status

| Brief | Status |
|-------|--------|
| UPDATE-BRIEF-e60.md | ✅ Written — no prototype change needed (already correct) |
| UPDATE-BRIEF-e58.md | ✅ Written — 2-zone sidebar, header bar, 5 new screens |
| UPDATE-BRIEF-e61.md | ✅ Written — Secrets tab in Project Settings |
| UPDATE-BRIEF-e63.md | ✅ Written — Usage page + sidebar entry |
| UPDATE-BRIEF-e64.md | ✅ Written — Schema Diagram view + enhanced columns tab |
| UPDATE-BRIEF-e65.md | ✅ Written — Previews tab in Site Detail |
