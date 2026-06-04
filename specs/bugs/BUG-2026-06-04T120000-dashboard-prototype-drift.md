# BUG-2026-06-04T120000: Dashboard prototype drift and stale metrics

## Problem

- **Actual:** Dashboard title/layout differs from prototype; CPU shows 0.0%; no Memory tile; footer is version-only and scrolls away.
- **Expected:** Welcome header, SystemStatusPanel (Components/CPU/Memory + activity), four stat cards, recent deployments + jump back in; live CPU/memory; sticky app footer with landing-style attribution.
- **Reproduce:** Log in → `/` → observe CPU 0.0%, no Memory card; scroll — footer leaves viewport.

## Root Cause Analysis

- Monitoring `SystemMetrics()` never set `cpu_percent` (always zero in JSON).
- `DashboardMetrics` omitted Memory and used legacy layout not matching prototype.
- Layout used minimal `.page-footer` without scroll shell for pinned footer.

**Risk level:** Medium

## TDD Fix Plan

1. **RED:** `TestSystemMetricsCPUPercentAfterSecondSample` — second sample returns CPU ≤ 100.
   **GREEN:** Rusage-based CPU sampling in monitoring.
   **verify:** `go test ./components/monitoring/...`

2. **RED:** `SystemStatusPanel.test.tsx` — operational banner, Components, CPU %, Memory MB.
   **GREEN:** `SystemStatusPanel.tsx` + dashboard CSS.
   **verify:** `cd ui && npx vitest run src/components/SystemStatusPanel.test.tsx`

3. **RED:** `DashboardPage.test.tsx` — Welcome back, Memory, 4 stats, Jump back in.
   **GREEN:** Refactor `DashboardPage.tsx`.
   **verify:** `cd ui && npx vitest run src/pages/DashboardPage.test.tsx`

4. **RED:** `Layout.test.tsx` — app-footer links, scroll shell.
   **GREEN:** `Layout.tsx` + CSS sticky footer.
   **verify:** `cd ui && npx vitest run src/Layout.test.tsx`

**REFACTOR:** `fmtMemoryMB` / `fmtUptime` in `ui/src/lib/format.ts` if shared.

## Acceptance Criteria

- [x] CPU non-zero after warm-up on live server
- [x] Memory tile with bar from API
- [x] Dashboard layout matches prototype sections
- [x] Footer pinned with full prototype copy and BigPowers attribution
- [x] All tests pass

## Resolution

**Fixed:** 2026-06-04

**Root cause confirmed:** `SystemMetrics()` omitted `CPUPercent`; dashboard UI used legacy `DashboardMetrics` without Memory or prototype layout; footer was not in a sticky scroll shell.

**Fix applied:**
- Rusage delta sampling sets `cpu_percent` in `components/monitoring/monitoring.go`.
- `SystemStatusPanel` + prototype-aligned `DashboardPage` (welcome header, CPU/Memory tiles, activity, 4 stat cards, deployments + jump back in).
- Sticky `app-footer` in `Layout.tsx` with scrollable `.content`; footer attribution matches landing (`BigPowers` → `github.com/danielvm-git/bigpowers`, `danielvm-git` → `github.com/danielvm-git`).

**Hardening added:** `TestSystemMetricsCPUPercentAfterSecondSample` asserts CPU percent > 0 on second sample; `fetchMonitoringMetricsWarmed` (dashboard + monitoring) for client-side CPU warm-up; `/health` returns `running` count with `degraded` when components offline; UI regression tests for warm fetch, footer scroll shell, and degraded component copy.

**Review follow-up (respond-review):** Client metrics warm-up, components tile degraded state, Go CPU test tightened, proxy health `running` field, Monitoring CPU card, `100dvh` layout, `process heap` label.

**Evidence:**
```text
go test ./components/monitoring/... -run TestSystemMetricsCPUPercent  → PASS
go test ./...                                                         → PASS (all packages)
cd ui && npx vitest run src/components/SystemStatusPanel.test.tsx \
  src/pages/DashboardPage.test.tsx src/Layout.test.tsx                → 16/16 PASS
cd ui && npm test                                                     → 179/179 PASS
cd ui && npm run build                                                → PASS (tsc + vite)
ui/dist bundle contains SystemStatusPanel, memory_mb, BigPowers links
```

**Commit:** `fix(ui): align dashboard with prototype and live CPU/memory metrics`
