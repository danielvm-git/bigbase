# Heading hierarchy: span-as-heading and h1→h3 skips

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** MODERATE
**WCAG:** 1.3.1 Info and Relationships / 2.4.6 Headings and Labels — Level A/AA

## Description

Headings rendered as `<span>` (absent from heading navigation) or heading levels skipped:

- `Card.tsx` CardHeader title is a `<span className="card-title">` — cascades to MetricCard, RequestChart, ComponentHealthGrid, DashboardMetrics
- `EmptyState.tsx` title is a `<span>` — all empty-state messages
- `SiteCard.tsx` site name is a `<span>` — deploy list
- `SystemStatusPanel.tsx` title is a `<div>` — dashboard
- h1 → h3 skips (no h2): `FunctionsPage.tsx`, `ForgePage.tsx`, `CiciPage.tsx` (CiciPage also renders h3 before h2 in DOM order)
- `RealtimePage.tsx` error/loading branches render no h1 at all

## Recommended Fix
Render card/empty-state titles as heading elements (`<h2>`/`<h3>` via adjustable level prop); never skip heading levels; always render the page `<h1>` before branching on state.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
