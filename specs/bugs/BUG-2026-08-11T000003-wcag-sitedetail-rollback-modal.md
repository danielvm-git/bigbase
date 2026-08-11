# SiteDetailPage rollback confirmation is an ad-hoc div modal (keyboard trap)

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** CRITICAL
**WCAG:** 2.1.1 Keyboard / 2.1.2 No Keyboard Trap / 4.1.2 — Level A

## Description

The rollback confirm overlay (`SiteDetailPage.tsx` ~L600-630, `{rollbackTarget && (<div style={{position:'fixed',inset:0...}} onClick={close}>`) is a hand-built `<div>` with no `role="dialog"`, no `aria-modal`, no `aria-labelledby`, no Escape handler, no focus move/trap on open, no focus restoration on close. A keyboard user who opens it cannot reliably dismiss it by keyboard and focus is unmanaged.

## Affected Files
- `ui/src/pages/SiteDetailPage.tsx`

## Recommended Fix
Replace with the shared `<Modal>`/`<Dialog>` component, which already provides `role="dialog"`, `aria-modal`, initial focus, Escape-to-close, and a focus trap.

## Status
fixed

## Resolution
Replaced the hand-built rollback overlay in `ui/src/pages/SiteDetailPage.tsx` with the shared `Modal` component (imported from `../components`). The overlay now renders with `role="dialog"`, `aria-modal="true"`, `aria-labelledby` (title), initial focus on open, Tab focus trap, Escape-to-close, and focus restoration to the trigger on close. Added a `closeRollback` callback; confirm/cancel buttons, error display, and loading state preserved unchanged. Regression assertion added to `SiteDetailPage.test.tsx` (role=dialog + aria-modal + Escape closes); all 15 tests pass and `npm run build` is green. Fixed 2026-08-11 (commit `fix(ui): replace rollback overlay with accessible Modal (WCAG 2.1.2)`).

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
