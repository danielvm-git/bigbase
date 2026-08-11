# Sidebar NavLinks lose accessible name on mobile (empty link names)

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** CRITICAL
**WCAG:** 4.1.2 Name, Role, Value — Level A

## Description

On mobile (≤768px) the collapsed sidebar hides link text via `display:none` (`@media (max-width:768px)` in `ui/src/index.css`), leaving only the `<Icon>` which renders `<svg aria-hidden>`. Every `NavLink` therefore resolves to an EMPTY accessible name; screen readers announce a string of nameless links. Violates 4.1.2 and harms 2.4.4 Link Purpose.

## Affected Files
- `ui/src/components/Sidebar.tsx` (NavLink rendering, ~line 31)
- `ui/src/index.css` (~line 1993, mobile collapse rule)

## Recommended Fix
Add `aria-label={item.label}` to each `<NavLink>` (works collapsed and expanded), or use a visually-hidden (sr-only) label technique instead of `display:none`.

## Status
fixed

## Resolution
2026-08-11 — Added `aria-label={item.label}` to every NavLink rendered by `SidebarSection` in `ui/src/components/Sidebar.tsx`. This applies to all nav sections (build/data/etc. passed via props from Layout.tsx) since every link is produced by the same items map. The label now resolves to a non-empty accessible name in both collapsed (≤768px, label span `display:none`) and expanded states; aria-label duplicates the visible text, so the accessible name is unchanged when expanded. Regression test added to `ui/src/components/Sidebar.test.tsx` asserting every nav link has a non-empty accessible name (aria-label or text content). Targeted vitest: 9/9 passed; `npm run build` green.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
