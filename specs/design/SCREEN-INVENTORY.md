# BigBase Screen Inventory

Living map of every screen: its route, owning epic, and prototype status.
Updated before each epic implementation starts.

**Last updated:** 2026-07-12
**Prototype project:** BigBase Prototype (`ec1480a1`) — BUILT. `BigBase Console.html` (modular `bb/*.jsx` runtime: split source files concatenated into `bundle.jsx`) + `BigBase Landing.html` (static marketing page mirroring bigbase.click root).
**Design System project:** BigBase Design System (`502492b2`) — tokens (`colors_and_type.css`: light/dark + 12 monthly accents + indigo default) + `components.css` + `_ds_bundle.js` component previews.
**Spec file:** `specs/design/PROTOTYPE-SPEC-v1.md` — spec for all 20 screens (updated 2026-07-12: 13 accents, v2.76.15 footer, 8 Site Detail tabs).
**Known debt (harmonization Phase 1):** `bundle.jsx` drifted from split files (Deploy Keys tab dead at runtime); `pages-projects.jsx` monkey-patches a second app over the first render. Fix: split files become the only editable source, bundle regenerated via documented manifest, EnhancedApp dissolved.

---

## Surface Parity Matrix (harmonization scoreboard)

Legend — **Spec**: described in PROTOTYPE-SPEC/UPDATE-BRIEF · **P2**: design-system classes cover it · **P1**: renders in prototype · **Live**: shipped in `ui/src` on bigbase.click. ✅ aligned · ⚠️ partial/drift · ❌ missing · `eXX` = open epic that closes it.

| Screen | Route | Spec | P2 | P1 | Live | Gap owner |
|--------|-------|------|----|----|------|-----------|
| Landing (marketing) | `/` (public) | ✅ | ✅ | ✅ | ✅ | — |
| Login | `/login` | ✅ | ✅ | ✅ | ✅ | — |
| Global Dashboard | `/` | ✅ | ✅ | ✅ | ✅ | — |
| Sites List | `/sites` | ✅ | ✅ | ✅ | ⚠️ `/deploy` | e60 rename |
| Create Site | `/sites/new` | ✅ | ✅ | ✅ | ⚠️ `/deploy/new` | e60 |
| Site Detail (8 tabs) | `/sites/:id` | ✅ | ⚠️ table/modal classes | ⚠️ bundle drift (7 tabs live) | ⚠️ `/deploy/:id` | Ph 1 + e60 |
| — Previews tab | (tab) | ✅ e65 brief | ⚠️ | ✅ | ❌ | e65 |
| — Deploy Keys tab | (tab) | ✅ | ⚠️ | ⚠️ dead in bundle | ✅ | Ph 1 |
| Data Studio (+Schema Designer) | `/data` | ✅ e64 brief | ⚠️ | ✅ | ⚠️ no designer | e64 |
| SQL Editor | `/sql` | ✅ | ✅ | ✅ | ✅ | e58 replaces w/ scoped |
| Storage | `/storage` | ✅ | ✅ | ✅ | ✅ | — |
| Functions List / Detail | `/functions[/:id]` | ✅ | ✅ | ✅ | ✅ | — |
| Users | `/users` | ✅ | ✅ | ✅ | ✅ | — |
| Messaging / Detail | `/messaging[/:id]` | ✅ | ✅ | ✅ | ✅ | — |
| Git Repos | `/repos` | ✅ | ✅ | ✅ | ✅ | — |
| CI/CD | `/cici` | ✅ | ⚠️ terminal class | ✅ | ✅ | Ph 2 |
| Monitoring | `/monitoring` | ✅ | ⚠️ donut/sparkline | ✅ | ✅ | Ph 2 |
| Forge | `/forge` | ✅ | ⚠️ kanban class | ✅ | ✅ | Ph 2 |
| Realtime | `/realtime` | ✅ | ✅ | ✅ | ✅ | — |
| Events | `/events` | ✅ | ✅ | ✅ | ✅ | — |
| Settings | `/settings` | ✅ | ✅ | ✅ | ✅ | — |
| 404 | `*` | ⚠️ | ✅ | ❌ | ✅ | Ph 4 story |
| Projects List | `/projects` | ✅ e58 | ⚠️ | ✅ | ❌ | e58 |
| Project Dashboard | `/project/:id` | ✅ e58 | ⚠️ | ✅ | ❌ | e58 |
| Project SQL (3-panel) | `/project/:id/sql[/:branch]` | ✅ e58 | ⚠️ | ✅ | ❌ | e58 |
| Project Branches | `/project/:id/branches` | ✅ e58 | ⚠️ | ✅ | ❌ | e58/e59s02 |
| Project Settings (5 tabs incl. Secrets) | `/project/:id/settings` | ✅ e58+e61 | ⚠️ | ✅ | ❌ | e58, e61 |
| Usage Dashboard | `/usage` | ✅ e63 brief | ⚠️ | ❌ (intentionally held until ship) | ❌ | e63 |
| Platform Users / Settings / Invites | `/admin/platform/*` | ❌ not designed | ❌ | ❌ | ❌ | e66 (brief first) |

**P2 ⚠️ column note:** design system lacks classes for modal, app-footer, bare `.table`, segmented, switch, donut/sparkline, timeline, kanban, terminal — currently `bb-*` in prototype `custom.css` or inline. Closed by harmonization Phase 2.

---

## IA Change Summary (e57 → e58)

**Current sidebar (flat):**
```
Overview  →  Dashboard
Build     →  Sites, Functions
Data      →  Data Studio, SQL Editor, Storage
Auth      →  Users
Engage    →  Messaging
DevOps    →  Git Repos, CI/CD, Monitoring, Forge, Realtime, Events
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
  Git Repos, CI/CD, Monitoring, Forge, Realtime, Events
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
2. Prototype edit happens in the **split source files** (`bb/*.jsx`) — never directly in `bundle.jsx`; then `bundle.jsx` is regenerated per `bb/MANIFEST.md` concat order and pushed with `DesignSync.finalize_plan → DesignSync.write_files` to the BigBase Prototype project (`ec1480a1`). (Posting the brief to claude.ai/design for Claude Design remains a valid alternative path; export then replaces the split files.)
3. Claude Code generates gap analysis → `specs/design-feedback/eXX-VS-PROTOTYPE.md`
4. Implementation begins
5. On phase/epic exit: update the **Surface Parity Matrix** above — it is the harmonization scoreboard.

**Rules:**
- There is always exactly ONE prototype app: `BigBase Console.html` loading the generated `bundle.jsx` (plus the standalone `BigBase Landing.html` for the public marketing page)
- New screens go into the appropriate split file (or a new `bb/pages-*.jsx` added to the manifest) — never a second HTML console
- Solo Component Spec explorations are temporary: merge into the prototype within the same work session and delete the solo file

**Design System sync** (after any epic that ships new or changed components):
```
DesignSync.finalize_plan → DesignSync.write_files
```
Target project: `BigBase Design System` (`502492b2-4dcc-4024-9e7a-26baa7943ca7`)
This project holds component previews, tokens, and brand assets only — no prototype screens.
