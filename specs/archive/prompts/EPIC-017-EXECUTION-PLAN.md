---
type: execution-plan
epic: 017 — Enhanced Admin UI
context: UI-first vertical slices with per-slice quality-gate pipeline
date: 2026-06-01
---

# Epic 017 Execution Plan — Enhanced Admin UI

Zero-to-prod execution for the first epic of BigBase v2.0. 5 vertical slices, each
independently shippable through the full quality-gate pipeline:

**verify-work → audit-code → request-review → respond-review → (validate-fix) → commit-message → release-branch**

---

## Pre-Flight (Before Any Slice)

These two tasks unlock TDD and design consistency for all subsequent work.

### P0-A: Design Token Foundation

| Step | Action | Verify |
|------|--------|--------|
| 1 | Extract all design tokens from `specs/epics/017-enhanced-admin-ui/SYSTEM_DESIGN.md` and `specs/BigBase Console.html` (354KB prototype) | — |
| 2 | Create `ui/src/styles/tokens.css` with CSS custom properties: brand colors, semantic colors, typography (Inter + Fira Code with fallback stack), spacing scale (4/8/12/16/24/32/48/64), border radius, shadows, z-index layers | `grep -c 'var(--' ui/src/styles/tokens.css` ≥ 30 |
| 3 | Create `ui/src/styles/theme.css` with `[data-theme="dark"]` overrides. Dark colors per SYSTEM_DESIGN.md spec. No `prefers-color-scheme` coupling | `cd ui && npm run build` — dark mode CSS present in dist |
| 4 | Create `ui/src/styles/globals.css` that imports tokens.css + theme.css + reset + utility classes | Files referenced correctly |
| 5 | Update `ui/src/index.css` to import globals.css. Verify existing pages don't break visually | `cd ui && npm run build` passes, visual comparison against current main |

### P0-B: UI Test Infrastructure

| Step | Action | Verify |
|------|--------|--------|
| 1 | Install Vitest + @testing-library/react + @testing-library/jest-dom + jsdom: `cd ui && npm install -D vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom` | `npm ls vitest` shows version |
| 2 | Add test config to `ui/vite.config.ts`: `test: { environment: 'jsdom', globals: true, setupFiles: './src/test-setup.ts' }` | Config parses without error |
| 3 | Create `ui/src/test-setup.ts`: `import '@testing-library/jest-dom'` | File exists |
| 4 | Add `"test": "vitest run"` and `"test:watch": "vitest"` to `ui/package.json` scripts | `cd ui && npm test` runs (0 tests, no crash) |
| 5 | Write a trivial smoke test for an existing component (e.g., `Badge.test.tsx`: renders with correct text and variant class) | `cd ui && npm test -- --reporter=verbose` — 1 test passes |

**P0 Gate:** `cd ui && npm run build && npm test` — build + test both green.

---

## Slice 017-A: Realtime Inspector Page

**Scope:** New `#/realtime` page. List active WebSocket connections, view live event feed, inspect channel subscriptions. **UI-only** — no new backend endpoints (reads from monitoring component's existing data).

**Design reference:** `specs/epics/017-enhanced-admin-ui/SYSTEM_DESIGN.md` — look for WS inspector layout in prototype.

### A-1: TDD — Write Failing Tests

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `RealtimePage.test.tsx`: renders connection list table, shows channel subscriptions per connection, renders event feed section, shows "no connections" empty state | `cd ui && npm test -- RealtimePage` — **RED** (page doesn't exist yet) |
| 2 | Write component tests for any new primitives needed (e.g., `ConnectionRow`, `EventFeedItem`) | Tests exist but fail (component files don't exist yet) |

### A-2: TDD — Implement

| Step | Action | Verify |
|------|--------|--------|
| 1 | Create `ui/src/pages/RealtimePage.tsx` — connection table (ID, IP, connected at, channels subscribed), live event feed panel, channel subscription inspector | `cd ui && npm test -- RealtimePage` — **GREEN** |
| 2 | Add `#/realtime` route in `App.tsx` | Route resolves in browser |
| 3 | Add sidebar nav entry "Realtime" with appropriate icon | Nav item appears in sidebar |

### A-3: verify-work Gate

| Step | Action | Verify |
|------|--------|--------|
| 1 | Cold-start smoke: run server, navigate to `#/realtime` | Page renders without console errors |
| 2 | Build: `cd ui && npm run build` | Build passes clean |
| 3 | Typecheck: `cd ui && npx tsc --noEmit` | Zero type errors |
| 4 | Lint: `cd ui && npm run lint` | Zero lint errors |
| 5 | Tests: `cd ui && npm test -- --coverage` | All tests pass, coverage tracked |
| 6 | Visual: dark mode toggle works on Realtime page, responsive layout on mobile (≤768px) | Screenshot comparison acceptable |

### A-4: audit-code Gate

| Action | Verify |
|--------|--------|
| Run `audit-code` skill: check CONVENTIONS.md compliance (naming, immutability, error handling), Boy Scout Rule (leave files better), test coverage, type safety, SOLID | All checklist items pass |

### A-5: request-review Gate

| Action | Verify |
|--------|--------|
| Dispatch fresh reviewer agent with `request-review` skill. Reviewer critiques code quality, test coverage, API shape, security | Review report received |

### A-6: respond-review Gate

| Action | Verify |
|--------|--------|
| Act on review findings systematically. Categorize (must-fix / should-fix / nice-to-have). Apply fixes. Re-run tests | `cd ui && npm test` — still green after fixes |

### A-7: validate-fix Gate (Conditional)

| Condition | Action | Verify |
|-----------|--------|--------|
| Only if review findings required code changes | Re-run the failing test that triggered the fix, run full suite, typecheck, lint, harden against recurrence | All verification steps from A-3 re-pass |

### A-8: commit-message Gate

| Action | Verify |
|--------|--------|
| Run `commit-message` skill: draft Conventional Commits title + body, note semantic-release version bump | Commit message drafted and approved |

### A-9: release-branch Gate

| Action | Verify |
|--------|--------|
| Create PR with `gh`, verify CI passes, merge decision | PR created, CI green, branch merged |

**017-A verify command:** `cd ui && npm test -- RealtimePage -- --coverage && npm run build`

---

## Slice 017-B: Function Logs Viewer

**Scope:** New `#/functions/:id/logs` page + new backend endpoint `GET /api/functions/:id/logs`. Execution history, console output capture, error traces.

**Backend work:** `components/functions/` — new endpoint + test.

### B-1: TDD — Backend Endpoint

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `TestLogs` in `components/functions/functions_test.go`: creates a function, executes it, then calls `GET /api/functions/:id/logs`. Asserts response shape: `{logs: [{timestamp, level, message, ...}]}`. Tests: empty logs, many logs, pagination | `go test ./components/functions/ -run TestLogs -v` — **RED** |
| 2 | Implement `GET /api/functions/:id/logs` handler in `components/functions/handlers.go`. Query execution history from DB, return JSON array | `go test ./components/functions/ -run TestLogs -v` — **GREEN** |

### B-2: TDD — UI Page

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `FunctionLogsPage.test.tsx`: renders log entries chronologically, shows function name/ID header, handles empty state, handles error state, tests log level badges (info/warn/error) | `cd ui && npm test -- FunctionLogsPage` — **RED** |
| 2 | Create `ui/src/pages/FunctionLogsPage.tsx`. Fetch from `/api/functions/:id/logs`. Log table with timestamp, level badge, message. Auto-scroll to bottom | `cd ui && npm test -- FunctionLogsPage` — **GREEN** |
| 3 | Link from existing `FunctionsPage.tsx` to logs page per-function | Navigation works |

### B-3 through B-9: Quality Gates

Same pipeline as 017-A (verify-work → audit-code → request-review → respond-review → commit-message → release-branch). Apply to both backend and UI changes together (single PR).

**017-B verify command:** `go test ./components/functions/ -run TestLogs -v && cd ui && npm test -- FunctionLogsPage -- --coverage && npm run build`

---

## Slice 017-C: Storage Browser

**Scope:** Enhanced `#/storage` page + new thumbnail endpoint. File grid/list toggle, preview modal (images, text), folder tree navigation, drag-drop upload.

**Backend work:** `components/storage/` — thumbnail generation endpoint.

### C-1: TDD — Thumbnail Endpoint

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `TestThumbnail` in `components/storage/storage_test.go`: upload an image, call `GET /api/storage/:bucket/:file/thumbnail?w=200&h=200`. Asserts response is image/jpeg, dimensions ≤ requested | `go test ./components/storage/ -run TestThumbnail -v` — **RED** |
| 2 | Implement thumbnail handler. Use Go stdlib `image` package for resize. Cache thumbnails on disk | `go test ./components/storage/ -run TestThumbnail -v` — **GREEN** |

### C-2: TDD — UI Page

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `StoragePage.test.tsx`: renders file grid, toggles to list view, opens preview modal for images, opens preview modal for text files, handles drag-drop zone, renders folder tree | `cd ui && npm test -- StoragePage` — **RED** |
| 2 | Create/overhaul `ui/src/pages/StoragePage.tsx`: grid/list toggle, image preview modal, text preview modal, drag-drop upload zone, folder tree navigation, bucket selector | `cd ui && npm test -- StoragePage` — **GREEN** |
| 3 | Write tests for new primitives: `PreviewModal`, `FileGrid`, `FileList`, `FolderTree`, `DropZone` | Component tests green |

### C-3 through C-9: Quality Gates

Same pipeline. Single PR for backend + UI.

**017-C verify command:** `go test ./components/storage/ -run TestThumbnail -v && cd ui && npm test -- StoragePage -- --coverage && npm run build`

---

## Slice 017-D: Deploy & CICI Detail Viewers

**Scope:** Enhanced `#/deploy/:id` page with build log streaming, status timeline, env vars editor. Enhanced `#/cici/:id` with workflow DAG visualization.

**Backend work:** `components/deploy/` — log stream endpoint.

### D-1: TDD — Log Stream Endpoint

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `TestLogStream` in `components/deploy/deploy_test.go`: start a deployment, call `GET /api/deploy/:id/logs/stream` (SSE), assert event stream delivers log lines as deployment progresses | `go test ./components/deploy/ -run TestLogStream -v` — **RED** |
| 2 | Implement SSE log stream handler. Read from deploy process stdout/stderr pipe, emit as `data: {line, stream}` events | `go test ./components/deploy/ -run TestLogStream -v` — **GREEN** |

### D-2: TDD — UI Pages

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `DeployDetailPage.test.tsx`: renders status timeline, connects to SSE log stream and displays entries, renders env vars editor, handles deploy not found, shows build duration | `cd ui && npm test -- DeployDetailPage` — **RED** |
| 2 | Write `CiciDetailPage.test.tsx`: renders workflow DAG, shows step statuses, run history table, trigger info (push/manual), actor, duration | `cd ui && npm test -- CiciDetailPage` — **RED** |
| 3 | Overhaul `ui/src/pages/DeployPage.tsx` (detail view). Add `StreamLog` component that connects to SSE. Add status timeline (queued → building → deploying → live). Add env vars editor section | `cd ui && npm test -- DeployDetailPage` — **GREEN** |
| 4 | Overhaul `ui/src/pages/CiciPage.tsx` (detail view). Add workflow DAG visualization (nodes + edges), step status badges, run history table | `cd ui && npm test -- CiciDetailPage` — **GREEN** |
| 5 | Write tests for new primitives: `StreamLog`, `StatusTimeline`, `DAG`, `EnvVarEditor` | Component tests green |

### D-3 through D-9: Quality Gates

Same pipeline. Single PR.

**017-D verify command:** `go test ./components/deploy/ -run TestLogStream -v && cd ui && npm test -- --coverage && npm run build`

---

## Slice 017-E: Dashboard Overhaul

**Scope:** Real-time request rate chart, error rate gauge, active user count, component health indicators (16/16 grid), dark mode toggle with localStorage persistence, responsive sidebar (240px → 64px icon-only on mobile), toast notification system, quick-action buttons (Create Collection, Deploy Site, Run Function).

**This is the capstone slice** — it ties together all previous work and adds the systemic features (dark mode, toasts, sidebar) used by all pages.

### E-1: TDD — New Primitives

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `ToastProvider.test.tsx`: renders toast on `useToast().show()`, auto-dismisses after timeout, stacks multiple toasts, supports info/success/error variants | `cd ui && npm test -- Toast` — **RED** |
| 2 | Write `ThemeToggle.test.tsx`: toggles between light/dark, persists to localStorage, applies `data-theme` attribute to `<html>` | `cd ui && npm test -- Theme` — **RED** |
| 3 | Write `ResponsiveSidebar.test.tsx`: desktop shows full sidebar (240px), mobile shows icon-only (64px), hamburger toggle expands mobile, nav links highlight active route | `cd ui && npm test -- Sidebar` — **RED** |
| 4 | Write `MetricCard.test.tsx`, `RequestChart.test.tsx`, `ComponentHealthGrid.test.tsx`, `QuickActions.test.tsx` | All **RED** |

### E-2: TDD — Implement Primitives

| Step | Action | Verify |
|------|--------|--------|
| 1 | Create `ui/src/context/ToastContext.tsx` + `ui/src/hooks/useToast.ts`. Toast component with auto-dismiss, stacking, variants. Add `<ToastProvider>` to App.tsx | `cd ui && npm test -- Toast` — **GREEN** |
| 2 | Create `ui/src/hooks/useTheme.ts` + `ui/src/context/ThemeContext.tsx`. Dark mode toggle, localStorage persistence, `data-theme` attribute. Add `<ThemeProvider>` to App.tsx | `cd ui && npm test -- Theme` — **GREEN** |
| 3 | Refactor `ui/src/Layout.tsx`: responsive sidebar with hamburger toggle. 240px on desktop, 64px icon-only on mobile. Active route highlighting. Collapse/expand animation | `cd ui && npm test -- Sidebar` — **GREEN** |
| 4 | Create `MetricCard.tsx` (icon, label, value, trend), `RequestChart.tsx` (line chart, uses polling from monitoring API), `ComponentHealthGrid.tsx` (16 component cards with status), `QuickActions.tsx` (3 buttons: Create Collection, Deploy Site, Run Function) | Component tests all **GREEN** |

### E-3: TDD — Integrate Dashboard Page

| Step | Action | Verify |
|------|--------|--------|
| 1 | Write `DashboardPage.test.tsx`: renders health banner, component grid (16/16), request rate chart, error rate gauge, active user count, quick-action buttons, activity feed. Tests dark mode responsiveness. Tests toast appears on quick-action click | `cd ui && npm test -- DashboardPage` — **RED** |
| 2 | Overhaul `ui/src/pages/DashboardPage.tsx`: integrate all new primitives. Health banner from monitoring `/health`. Request chart polling `/api/monitoring/metrics`. Component grid from component status. Quick actions wire to create modals or navigation | `cd ui && npm test -- DashboardPage` — **GREEN** |

### E-4: verify-work Gate

| Step | Action | Verify |
|------|--------|--------|
| 1 | Cold-start smoke: run `go run . serve`, navigate to all 8 screens: Dashboard, Sites, SQL Editor, Storage, Users, Git Repos, CI/CD, Monitoring | All render without console errors |
| 2 | Dark mode: toggle dark/light on every screen. Reload page — mode persists | Theme persists across reload on all 8 screens |
| 3 | Responsive: resize to ≤768px, verify sidebar collapses, hamburger works, all pages still usable | All screens functional on mobile |
| 4 | Toasts: trigger a quick action, verify toast appears and auto-dismisses | Toast animates in/out |
| 5 | Build: `cd ui && npm run build` | Build passes clean, `dist/` output verified |
| 6 | Typecheck: `cd ui && npx tsc --noEmit` | Zero type errors |
| 7 | Lint: `cd ui && npm run lint` | Zero lint errors |
| 8 | Full test suite: `cd ui && npm test -- --coverage` | All tests pass, coverage ≥ 80% for new code |
| 9 | Go tests: `go test ./...` | All backend tests still green |

### E-5 through E-9: Quality Gates

Same pipeline. **This PR is the largest** — reviewer should focus on dark mode consistency, responsive breakpoints, toast accessibility (aria-live), and sidebar keyboard navigation.

**017-E verify command:** `go test ./... && cd ui && npm test -- --coverage && npm run build`

---

## Post-Epic Cleanup

| Step | Action | Verify |
|------|--------|--------|
| 1 | Update `specs/TASKS.md` to match current RELEASE-PLAN.md numbering (017=Admin UI, 018=Multi-DB, etc.) | TASKS.md epics match RELEASE-PLAN.md |
| 2 | Update `specs/SCOPE.md` to match current numbering | SCOPE.md epics match |
| 3 | Update `specs/STATE.md` — mark Epic 017 as complete, update decisions | STATE.md reflects completion |
| 4 | Full CI: `npm run preflight` | All preflight checks pass |
| 5 | Sync `specs/` with repo, verify no stale references to old numbering | `git diff --stat` shows only intentional changes |

---

## Summary: All Verify Commands

| Slice | Verify Command |
|-------|---------------|
| P0 | `cd ui && npm test && npm run build` |
| 017-A | `cd ui && npm test -- RealtimePage -- --coverage && npm run build` |
| 017-B | `go test ./components/functions/ -run TestLogs -v && cd ui && npm test -- FunctionLogsPage -- --coverage && npm run build` |
| 017-C | `go test ./components/storage/ -run TestThumbnail -v && cd ui && npm test -- StoragePage -- --coverage && npm run build` |
| 017-D | `go test ./components/deploy/ -run TestLogStream -v && cd ui && npm test -- --coverage && npm run build` |
| 017-E | `go test ./... && cd ui && npm test -- --coverage && npm run build` |

---

## Dependency Graph Within Epic 017

```
P0 (tokens + vitest) ──────────────────────────────────┐
                                                        │
017-A (realtime) ── independent ──────────────────────┐ │
017-B (functions) ── independent ─────────────────────┤ │
017-C (storage) ──── independent ─────────────────────┤ │
017-D (deploy/cici) ─ independent ────────────────────┤ │
                                                        │ │
017-E (dashboard) ── needs A for realtime chart ──────┘ │
                    needs dark mode from E              │
                    needs toasts from E                 │
                    needs sidebar from E                │
```

Slices A-D can be built in parallel by different developers. E must come after at least A (for the realtime chart data source), but can start after A is merged since it primarily integrates UI components.

---

## Estimated Effort

| Slice | Backend | UI | Tests | Gates | Total |
|-------|---------|----|-------|-------|-------|
| P0-A (tokens) | — | 2h | — | — | 2h |
| P0-B (vitest) | — | 1h | 0.5h | — | 1.5h |
| 017-A (realtime) | — | 3h | 1.5h | 1h | 5.5h |
| 017-B (functions) | 2h | 2h | 1.5h | 1h | 6.5h |
| 017-C (storage) | 1.5h | 3h | 1.5h | 1h | 7h |
| 017-D (deploy/cici) | 1.5h | 4h | 2h | 1h | 8.5h |
| 017-E (dashboard) | — | 6h | 3h | 1.5h | 10.5h |
| Cleanup | — | — | — | 0.5h | 0.5h |

**Total: ~42 hours** (5-6 working days for one developer, 2-3 days with parallel slice work)

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| HTML prototype design assumptions diverge from real data shapes | Medium | Medium | Verify against actual API responses in 017-A first — that's the purest read-only page |
| Dark mode breaks existing pages not touched in 017-E | Medium | High | 017-E includes full visual regression: visit all 8 screens in light + dark mode |
| Vitest + jsdom compatibility issues with React 19 | Low | Medium | P0-B catches this immediately. Fallback: pin React Testing Library to compatible version |
| Backend log endpoints don't have real log data yet | Medium | Low | 017-B/017-D tests create synthetic log data via function execution / deploy trigger |
| Stale TASKS.md/SCOPE.md numbering causes confusion during parallel work on Epic 018 | Low | Medium | Post-epic cleanup step fixes this. Add comment in STATE.md about numbering during execution |
