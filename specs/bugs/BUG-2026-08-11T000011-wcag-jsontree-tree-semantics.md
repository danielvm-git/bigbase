# JsonTree missing tree semantics and keyboard navigation

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 4.1.2 Name, Role, Value / 2.1.1 Keyboard — Level A

## Description

`JsonTree.tsx` uses plain `<div>` nodes with NO `role="tree"`/`role="treeitem"`, and the toggle `<button>` (lines 58-65) has NO `aria-expanded`. No arrow-key (Up/Down/Left/Right), Home/End, or asterisk navigation — navigating a deep tree requires Tabbing through every button.

## Affected Files
- `ui/src/components/JsonTree.tsx`

## Recommended Fix
Add `role="tree"` + `role="treeitem"` + `aria-expanded` + `aria-level` per node, or switch to the disclosure pattern (button + region) with `aria-expanded`/`aria-controls`; implement roving tabindex with arrow-key navigation. Include key name and child count in toggle labels (e.g. `${keyName}: expand ${entries.length} items`).

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
