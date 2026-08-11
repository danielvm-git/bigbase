# Tables missing scope/caption; decorative glyphs not aria-hidden

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** MODERATE
**WCAG:** 1.3.1 Info and Relationships — Level A

## Description

Two related structural gaps:

1. **Table semantics** — several `<table>` blocks lack `scope="col"` on `<th>` cells and have no `<caption>`: `RequestLogs.tsx`, `SiteDeployKeysTab.tsx`, `SiteCacheTab.tsx`, `SiteDomainsTab.tsx`, `SiteEnvVarsTab.tsx`, plus the shared `Table.tsx` (no caption prop; `scope` defaults to 'col' unconditionally). Also `Table.tsx` scrollable wrapper lacks `role="region"`/`aria-label` (1.4.10 Reflow).
2. **Decorative glyphs not hidden** — emoji/symbols announced to screen readers with no meaning: `RealtimePage.tsx` (⚠, 🔌), `StoragePage.tsx` (📄), `SiteDetailPage.tsx` StatusTimeline dots, `SiteEnvVarsTab.tsx` ✓/— boolean cells (ambiguous "check mark"), `EmptyState.tsx` default em-dash icon, `StreamLog.tsx` emoji-only buttons (🕐).

## Recommended Fix
Add `scope="col"` to headers and a `<caption>` per table; add `role="region"` + `aria-label` to scrollable table wrappers; mark decorative glyphs `aria-hidden="true"` and replace symbol-only status cells with visually-hidden "Yes"/"No" text; give emoji-only buttons proper `aria-label`s.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
