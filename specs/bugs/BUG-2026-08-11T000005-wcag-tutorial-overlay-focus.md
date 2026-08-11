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
fixed

## Resolution
Ported the Modal.tsx focus-management pattern into `TutorialOverlay.tsx`:
- On mount, captures `document.activeElement` as the trigger and moves focus to the first focusable element inside the dialog (fallback: the dialog itself).
- Adds a `keydown` listener that traps Tab within the dialog (wraps first/last, and pulls focus back in if it escapes) and closes on Escape (via the same `handleDone` path as the ×/Finish buttons, preserving the "tutorial done" semantics).
- Cleans up the listener and restores focus to the trigger element on unmount.
- Dialog card now owns `role="dialog" aria-modal="true"` (backdrop is `role="presentation"`, matching Modal.tsx), gains `tabIndex={-1}`, and `aria-describedby="tutorial-description"` links the step body text.
- Added `ui/src/components/TutorialOverlay.test.tsx` covering: focus moves into dialog on open, Escape calls onClose, Tab/Shift+Tab wrap within the dialog, focus restored on close, and aria-describedby linkage. 6/6 tests pass; `npm run build` green.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
