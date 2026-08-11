# Dialog component missing focus trap, restoration, and initial focus

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 2.1.2 No Keyboard Trap / 2.4.3 Focus Order — Level A

## Description

`Dialog.tsx` renders `role="dialog" aria-modal="true"` with correct `aria-labelledby`, but has NO focus trap (keyboard users Tab out to background content), NO focus restoration on close, and NO auto-focus when opened. Only Escape-to-close is handled. Inconsistent with `Modal.tsx`, which implements the full pattern.

## Affected Files
- `ui/src/components/Dialog.tsx`

## Recommended Fix
Port the focus-management effect from `Modal.tsx`: capture trigger focus on open, focus the dialog/first focusable, trap Tab within the dialog, restore focus on unmount.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
