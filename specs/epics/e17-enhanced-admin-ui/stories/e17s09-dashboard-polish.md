---
id: e17s09
title: Dashboard tests and polish
status: done
legacy_slice: "017-I"
tasks:
  - desc: Write DashboardPage.test.tsx — user card, health banner, quick actions, stat cards
    verify: "cd ui && npx vitest run src/pages/DashboardPage.test.tsx"
  - desc: Test deployment/Messaging/Storage sections with mock API data
    verify: "cd ui && npx vitest run src/pages/DashboardPage.test.tsx"
  - desc: Test bar charts, toast notifications, and navigation on click
    verify: "cd ui && npx vitest run src/pages/DashboardPage.test.tsx"
  - desc: Add 8+ CSS custom property references to tokens.css (22 → 30+)
    verify: "grep -c 'var(--' ui/src/styles/tokens.css | awk '{if($1>=30) print \"PASS\"; else print \"FAIL\"}'"
  - desc: Full verify — DashboardPage tests pass + CSS token count >= 30
    verify: "cd ui && npm test -- DashboardPage -- --coverage && grep -c 'var(--' ui/src/styles/tokens.css | awk '{if($1>=30) print \"PASS\"; else print \"FAIL\"}'"

context: |
  DashboardPage.tsx (228 lines) exists with full rendering: user card, health
  banner, quick action buttons (Deploy Site, Run Function, Create Collection),
  DashboardMetrics widget, stat grid (Git Repos, Deployments, Messages, Files,
  Functions), deployment status bars, messaging channel bars, recent deployments
  table, recent messages table, and toasts on button clicks. No test file exists.
  CSS tokens: tokens.css currently has 22 var(-- references (compositional
  references between tokens). The verify gate requires ≥ 30.
---

## Implementation Steps

**Context**: `DashboardPage.tsx` is the most complex page — 8 parallel API calls
on mount, 5 stat cards, 2 bar charts, 2 tables, 3 quick-action buttons with
toasts, and a health banner. The test must mock all API responses and verify
each section renders correctly.

### Phase A: DashboardPage tests

#### Step 1: Test file scaffolding

Create `ui/src/pages/DashboardPage.test.tsx` following the `FunctionLogsPage.test.tsx`
pattern: Vitest + React Testing Library + MemoryRouter. Mock all 8 fetch calls
with realistic data shapes.

→ verify: `cd ui && grep -q 'describe.*DashboardPage' src/pages/DashboardPage.test.tsx`

#### Step 2: Authenticated render — user card and health banner

Mock `/api/auth/me` returning `{ id: 1, email: "admin@example.com" }`.
Mock `/health` returning `{ status: "ok", components: 16 }`.
Assert:
- "Signed in as admin@example.com" appears
- "All systems operational" banner (green) appears with "16 components running"

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "user card"`

#### Step 3: Quick action buttons and toasts

Assert 3 quick action buttons: "+ Deploy Site", "⚡ Run Function",
"📦 Create Collection". Click "Deploy Site" → assert toast showed "Create a new
deployment". Click "Run Function" → assert toast. Click "Create Collection" →
assert toast.

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "quick actions"`

#### Step 4: Stat grid (5 cards)

Mock all resource endpoints (git repos, deployments, messages, storage files,
functions) with counts. Assert each stat card shows correct count and label.
Click a card → assert navigation to the correct route.

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "stat grid"`

#### Step 5: DashboardMetrics widget

Mock `/api/monitoring/metrics` returning system metrics. Assert:
- "Request Rate" card with total
- "Error Rate" card with 500 count (red when > 0)
- "CPU" card with bar fill proportional to cpu_percent
- "Components" card with healthy count

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "metrics"`

#### Step 6: Bar charts — deployments and messaging

Mock deployments response with status breakdown. Assert:
- Status bar segments for "running", "failed", "building", "pending"
- Total deployment count
- Color mapping per status

Mock messages response with channel breakdown. Assert:
- Channel bar segments for "email", "sms", "push"
- Total message count

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "bar chart"`

#### Step 7: Recent activity tables

Assert deployments table shows rows with repo_id, branch, status badge, and
app_type. Assert messaging table shows rows with channel, to_addr, subject,
status. Verify "No recent activity" appears when arrays are empty.

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "activity"`

#### Step 8: Error and edge cases

- `/health` returns unknown status → warning banner with ⚠️
- `/api/auth/me` fails → "Loading..." shown (not crash)
- All API calls fail → page renders without error (graceful degradation)
- Single metric with zero value renders as "0" not empty

→ verify: `cd ui && npx vitest run src/pages/DashboardPage.test.tsx -t "error"`

### Phase B: CSS token gap

#### Step 9: Add 8+ compositional var(-- references to tokens.css

The current 22 references are in the semantic role section:
```css
--bg-default: var(--neutral-25);
--bg-surface: var(--neutral-0);
/* ... */
```

Add 8 more compositional references in these categories:
- 3 border semantic roles: `--border-default`, `--border-subtle`, `--border-accent`
- 2 divider roles: `--divider-default`, `--divider-subtle`
- 2 shadow roles: `--shadow-sm`, `--shadow-md`
- 1 overlay role: `--overlay-scrim`

Each references existing neutral tokens (e.g., `var(--neutral-100)`).

→ verify: `grep -c 'var(--' ui/src/styles/tokens.css | awk '{if($1>=30) print "PASS"; else print "FAIL"}'`

### Phase C: Full verify

#### Step 10: End-to-end verify

→ verify: `cd ui && npm test -- DashboardPage -- --coverage && grep -c 'var(--' ui/src/styles/tokens.css | awk '{if($1>=30) print "PASS"; else print "FAIL"}'`

## Verification Script (Manual)

1. `cd ui && npx vitest run src/pages/DashboardPage.test.tsx` — all tests pass
2. `grep -c 'var(--' ui/src/styles/tokens.css` → shows 30 or more
3. `npm test -- DashboardPage -- --coverage` — green with coverage > 50%
4. `npm run build` — builds cleanly

## Out of scope
- Integrating MetricCard/RequestChart/ComponentHealthGrid/QuickActions from e17s08 into DashboardPage (separate story or done as part of e17s08 integration)
- Real-time WebSocket metrics updates
- E2E tests for dashboard
- Lighthouse score measurement (epic-level closure task)

## Risks
- DashboardPage mocks 8 fetch calls; test must restore all mocks between tests
- `useToast` hook depends on context; test must wrap in `<ToastProvider>` or mock `useToast`
- Bar chart colors use CSS classes not inline styles; test by className assertions
- `MemoryRouter` needed for `useNavigate` clicks on stat cards
