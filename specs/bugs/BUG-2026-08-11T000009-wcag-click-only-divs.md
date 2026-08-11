# Systemic: click-only divs/spans without keyboard access

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 2.1.1 Keyboard — Level A

## Description

Interactive elements implemented as click-only `<div>`/`<span>` elements — unreachable by keyboard, no role/tabindex/keydown handling:

- `Card.tsx` (lines 6-11, 24) — spreads `...rest` onto `<div>`, so `onClick` makes a click-only div with no `role="button"`/`tabIndex`/`onKeyDown` enforcement
- `ForgePage.tsx` (~line 228) — Kanban `board-card` `<div onClick>` — mouse-only
- `StoragePage.tsx` (~L115, ~L145) — image filename `<span onClick>` and grid `<div onClick>` — mouse-only
- `DropdownMenu.tsx` (line 53) — trigger wrapper `<span onClick>` — keyboard activation depends entirely on the child

## Recommended Fix
Use real `<button>`/`<a>` elements, or add `role="button" tabIndex={0}` + `onKeyDown` (Enter/Space) automatically when interactive.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
