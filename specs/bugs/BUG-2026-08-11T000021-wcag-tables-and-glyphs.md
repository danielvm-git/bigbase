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
fixed

## Resolution
2026-08-11 — Tables: added `scope="col"` to all header cells and visually-hidden `<caption>`s (Request logs, Deploy keys, Build cache, Custom domains, Environment variables, Stored files, Deployment history, Rollback history); `Table.tsx` gained a `caption` prop rendered as the caption's first child, honors caller-provided `scope`, and labels its scrollable wrapper as `role="region"` + `aria-label` when a caption is given. Decorative glyphs: env-var Build/Runtime ✓/— cells now announce "Yes"/"No" via visually-hidden text with the glyph `aria-hidden`; StoragePage grid 📄 and SiteDetailPage StatusTimeline dots/connectors/live-dot are `aria-hidden="true"`; RealtimePage ⚠/🔌 and EmptyState default icon handled in T000018; StreamLog.tsx timestamp toggle button now has `aria-label="Toggle timestamps"` and the copy-button emoji (`📋`/`✓`) is wrapped in `aria-hidden` spans so the accessible name comes from visible text.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
