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
fixed

## Resolution
- `CardHeader` renders the title as a heading element with an adjustable `headingLevel` prop (default `h2`); all existing consumers (MetricCard, RequestChart, ComponentHealthGrid, DashboardMetrics, BuildCachePanel, OnboardingChecklist, DashboardPage, SettingsPage, SiteDetailPage) render unchanged.
- `EmptyState` title is now an `<h2>`; icon span is `aria-hidden="true"` and the default em-dash glyph was replaced with a neutral circle (T000021 glyph fix).
- `SiteCard` site name is now an `<h3>`; the link carries `aria-label={`Open ${site.name} deployment`}`.
- Removed h1→h3 level skips: FunctionsPage (New Function, Run Result), ForgePage (New Issue, board columns, issue detail, Comments), CiciPage (New Workflow) all promote to h2 (Comments h4→h3, issue detail h3→h2), fixing CiciPage's h3-before-h2 DOM order.
- RealtimePage always renders the page `<h1>` before branching on error/loading/empty/loaded state; ⚠ and 🔌 decorative glyphs are `aria-hidden="true"` (T000021 glyph fix).

Verified: `npx vitest run src/components/Card.test.tsx src/pages/CiciPage.test.tsx src/pages/RealtimePage.test.tsx` (19 passed) and `npm run build` (tsc + vite) in ui/.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
