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
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
