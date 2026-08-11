# ThemePicker menu lacks keyboard navigation and focus management

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 2.1.1 Keyboard / 4.1.2 Name, Role, Value / 2.4.3 Focus Order — Level A

## Description

`ThemePicker.tsx` (lines 64-70) declares `role="menu"` with `role="menuitem"` children but provides no Arrow/Home/End keyboard navigation (items only reachable by Tab), uses `aria-current` where selectable menu items require `role="menuitemradio"` + `aria-checked`, and on open focus stays on the trigger (no focus move into menu, no close-on-blur).

## Affected Files
- `ui/src/components/ThemePicker.tsx`

## Recommended Fix
Implement roving-tabindex + Arrow/Home/End per the ARIA menu pattern and switch selected items to `role="menuitemradio"` `aria-checked`, OR drop `role="menu"`/`menuitem` for a simpler accessible widget. Move focus to the selected item on open; close on blur/Tab-out.

## Status
fixed

## Resolution
2026-08-11 — `ui/src/components/ThemePicker.tsx` implements the ARIA menu pattern: items are `role="menuitemradio"` with `aria-checked` (replacing `aria-current`), roving tabindex with ArrowDown/ArrowUp/Home/End navigation (wrapping), focus moves to the selected item when the menu opens, Enter/Space activates the focused item, and focus leaving the popover (Tab-out/blur) closes it. Tests in `ui/src/components/ThemePicker.test.tsx` updated to `menuitemradio`/`aria-checked` and extended with focus-on-open, arrow/Home/End navigation, Enter selection, and Tab-out close coverage. Targeted vitest: 15/15 passed; `npm run build` green.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
