# StoragePage image preview is an ad-hoc div overlay (keyboard trap)

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** CRITICAL
**WCAG:** 2.1.1 Keyboard / 2.1.2 No Keyboard Trap / 4.1.2 — Level A

## Description

Image preview (`StoragePage.tsx` ~L170-195, `{previewFile && (<div style={{position:'fixed'...}} onClick={close}>)`) is an ad-hoc fullscreen `<div>` with no `role="dialog"`, no `aria-modal`, no `aria-label`, no focus management, no Escape handler, no focus trap.

## Affected Files
- `ui/src/pages/StoragePage.tsx`

## Recommended Fix
Render through `<Modal>`/`<Dialog>` with `title={previewFile.name}` and proper focus handling.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
