# Input component errors not associated via aria-describedby

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 3.3.1 Error Identification — Level A

## Description

`Input.tsx` renders error text as a bare `<span className="input-error-text">` with no `id`, and does not wire `aria-describedby` or `aria-invalid` to the field. Screen readers cannot associate an error message with its field. `Select.tsx` and `Checkbox.tsx` implement the correct pattern — Input does not.

## Affected Files
- `ui/src/components/Input.tsx` (lines 84, and missing aria wiring on the input element)

## Recommended Fix
Give the error text a stable `id` (e.g. `${id}-error`), set `aria-describedby={error ? \`${id}-error\` : undefined}` and `aria-invalid={error ? 'true' : undefined}` on the input/textarea/select, matching the Select.tsx pattern.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
