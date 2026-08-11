# Systemic: meaning conveyed by color alone

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** MODERATE
**WCAG:** 1.4.1 Use of Color — Level A

## Description

Semantic state conveyed by background/text color only, with no icon, shape, or text cue:

- `Alert.tsx` — variant (success/warning/error/info) by color class only
- `Badge.tsx` — semantic variant by background color only
- `MetricCard.tsx` — trend arrows (↑/↓ Unicode, no aria-label); good/bad value color-only
- `RequestChart.tsx` — status codes color-coded (2xx green/5xx red) with no category text
- `OnboardingChecklist.tsx` — done/not-done via aria-hidden check glyph + dimmed color

## Recommended Fix
Add per-variant icons (aria-hidden + visible text label) or visually-hidden status words (e.g. "Success:", "Error:"); make trend arrows carry `aria-label`/visually-hidden text; add category labels to chart legends.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
