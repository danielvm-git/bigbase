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
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
