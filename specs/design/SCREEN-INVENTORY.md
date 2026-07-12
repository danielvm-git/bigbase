# BigBase Screen Inventory

Living map of every screen: its route, owning epic, and prototype status.
Updated before each epic implementation starts.

**Last updated:** 2026-07-12
**Prototype project:** BigBase Prototype (`ec1480a1`) — BUILT. `BigBase Console.html` (modular `bb/*.jsx` runtime: 13 split source files concatenated into `bundle.jsx` per `bb/MANIFEST.md`, regenerated via `bb/make-bundle.sh`) + `BigBase Landing.html` (static marketing page mirroring bigbase.click root).
**Design System project:** BigBase Design System (`502492b2`) — tokens (`colors_and_type.css`: light/dark + 12 monthly accents + indigo default) + `components.css` + `_ds_bundle.js` component previews.
**Spec file:** `specs/design/PROTOTYPE-SPEC-v1.md` — spec for all 20 screens (updated 2026-07-12: 13 accents, v2.76.15 footer, 8 Site Detail tabs).
**Harmonization Phase 1 — done:** `pages-projects.jsx`'s e58 project routes + PROJECT sidebar zone + ContextBar are now merged directly into `app.jsx`/`shell.jsx` (the split-file sources); `EnhancedApp`/`EnhancedSidebar` and the second `ReactDOM.createRoot(...).render(...)` call are gone — there is exactly one render pass. `bundle.jsx` is a generated artifact (`bb/make-bundle.sh`, verified idempotent) covering all 13 sections that used to live only inside it (icons, functions, users-messaging, devops, monitoring, forge, misc now also have standalone split files). Deploy Keys tab (previously dead at runtime because it existed in `pages-sites.jsx` but not in the deployed `bundle.jsx`) now renders correctly — verified live in-browser.
**New finding for Phase 2 (P2 design-system project):** `_ds_bundle.js` contains a leftover mirrored copy of `source/bigbase-ui/main.tsx` that unconditionally self-mounts (`createRoot(document.getElementById('root')).render(<StrictMode><HashRouter><App/></HashRouter></StrictMode>)`) using the react-stubs demo `App`, not the console's real app. On any page that also loads `bb/bundle.jsx` into the same `#root` (i.e. `BigBase Console.html`), this fires first, throws (react-stubs demo references something the console page doesn't have), and logs a "createRoot called twice" + "type is invalid" console error before the real app's render call overwrites it. Harmless to the final rendered page — real console visually correct in every test — but pollutes the console on every load and should be gated (e.g. behind a `!document.querySelector('[data-bb-app]')` check or split into a truly separate preview-only bundle) as part of Phase 2's design-system cleanup.

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
| Site Detail (8 tabs) | `/sites/:id` | ✅ | ⚠️ table/modal classes | ✅ 8 tabs (Ph 1 fixed) | ⚠️ `/deploy/:id` | e60 |
| — Previews tab | (tab) | ✅ e65 brief | ⚠️ | ✅ | ❌ | e65 |
| — Deploy Keys tab | (tab) | ✅ | ⚠️ | ✅ (Ph 1 fixed — was dead in bundle) | ✅ | — |
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
| Projects List | `/projects` | ✅ e58 | ⚠️ | ✅ (single-app, Ph 1) | ❌ | e58 |
| Project Dashboard | `/project/:id` | ✅ e58 | ⚠️ | ✅ (single-app, Ph 1) | ❌ | e58 |
| Project SQL (3-panel) | `/project/:id/sql[/:branch]` | ✅ e58 | ⚠️ | ✅ (single-app, Ph 1) | ❌ | e58 |
| Project Branches | `/project/:id/branches` | ✅ e58 | ⚠️ | ✅ (single-app, Ph 1) | ❌ | e58/e59s02 |
| Project Settings (5 tabs incl. Secrets) | `/project/:id/settings` | ✅ e58+e61 | ⚠️ | ✅ (single-app, Ph 1) | ❌ | e58, e61 |
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
