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
fixed

## Resolution

**Fixed:** 2026-08-11

Non-color cues added so semantic state is never conveyed by color alone (WCAG 1.4.1):

- `Alert.tsx` — visible per-variant status word prefix (`Info:`/`Success:`/`Warning:`/`Error:`) rendered before the message.
- `Badge.tsx` — distinct per-variant indicator glyph (`•`/`✓`/`!`/`✕`/`★`/`i`, `aria-hidden`) alongside the visible text.
- `MetricCard.tsx` — trend arrows wrapped with visually-hidden text (`trending up`/`trending down`/`trend flat`), arrow glyph marked `aria-hidden`; color-coded value gets a visually-hidden status word (`success`/`warning`/`error`).
- `OnboardingChecklist.tsx` — each step rendered as a disabled `checkbox` with `aria-checked` + accessible label (`<label>, done|to do`); progress count already wrapped in `aria-live="polite"`.
- `RequestChart.tsx` — status-code legend category text (fixed in parallel by W2Charts).

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
