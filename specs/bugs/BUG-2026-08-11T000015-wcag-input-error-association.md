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
fixed

## Resolution
2026-08-11 — `ui/src/components/Input.tsx` now wires error association per the Select.tsx pattern: the error message carries a stable `id` (`${id}-error`, `role="alert"`), the hint carries `${id}-hint`, and the input/textarea/select gets `aria-invalid={error ? 'true' : undefined}` plus `aria-describedby` pointing at the error (or hint) id on all three variants. Tests added in `ui/src/components/Input.test.tsx` covering aria-describedby/aria-invalid on input, textarea, and select variants, hint association, and the no-error case. Targeted vitest: all 3 suites pass (40 tests total); `npm run build` green.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
