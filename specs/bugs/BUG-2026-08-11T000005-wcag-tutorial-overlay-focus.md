# TutorialOverlay declares modal dialog but has no focus management

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** CRITICAL
**WCAG:** 2.1.2 No Keyboard Trap / 2.4.3 Focus Order / 4.1.2 — Level A

## Description

`TutorialOverlay.tsx` (line 81) declares `role="dialog" aria-modal="true"` but implements NO focus trap, NO focus-move-into-dialog on open, NO focus-restore on close, and NO Escape-to-close handler. Focus leaks to background controls behind the backdrop; the only keyboard dismiss path is tabbing to the ×/Finish buttons.

## Affected Files
- `ui/src/components/TutorialOverlay.tsx`

## Recommended Fix
Mirror `Modal.tsx` (which does this correctly): capture trigger focus on open, focus the dialog/first element, trap Tab within the dialog, restore focus on unmount, and add a keydown Escape → onClose handler.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
