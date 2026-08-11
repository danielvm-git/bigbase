# Systemic: loading/error/status states not announced (no live regions)

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 4.1.3 Status Messages — Level AA

## Description

Loading and error states are visual-only across the app — no `role="status"`/`aria-busy` for loaders, no `role="alert"` for dynamically-inserted errors. Screen-reader users get no indication that content is loading or that an error occurred. Correct exemplars exist: `Spinner.tsx` and `SkeletonCard.tsx` (role="status" + aria-busy + aria-label) — copy those patterns.

Affected: `BuildLogs`, `StreamLog`, `TerminalLogViewer`, `FunctionLogsPanel`, `RequestLogs`, `SiteCacheTab`, `BuildCachePanel`, `ListPage` (custom loadingMessage branch), `DashboardPage`, `OnboardingChecklist` (progress), `EventsPage` (SSE log), `MonitoringPage` (SSE metrics).

## Recommended Fix
Wrap loading states in `role="status" aria-busy="true"`; give dynamic errors `role="alert"`; add `aria-live="polite"` to SSE-updated regions (EventsPage log, MonitoringPage metrics).

## Status
open

**Fixed:** 7fecbe061 — loading branches wrapped in role="status" aria-busy, dynamic errors role="alert", SSE/stream regions aria-live="polite" across 12 components

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
