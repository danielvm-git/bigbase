# Charts/gauges lack data in accessible names

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 1.1.1 Non-text Content — Level A

## Description

Chart components have `role="img"`/`aria-label` or none at all, but accessible names are generic and omit the actual data — assistive tech gets no values:

- `BarGauge.tsx` (SingleBar, lines 35-49) — no `role="progressbar"`, no `aria-valuenow/min/max`, no `aria-label`
- `DonutGauge.tsx` (lines 30-36) — `aria-label="gauge"` never includes used/total/percentage
- `Sparkline.tsx` (lines 16, 29) — `aria-label="sparkline"` — no data summary
- `SystemStatusPanel.tsx` (lines 78-83, 95-98) — CPU/memory bars no `role="progressbar"`/aria-values
- `RequestChart.tsx` (line 44) — per-segment values in `title` attribute only (not keyboard-focusable, ignored by most screen readers)

## Recommended Fix
Compose data-bearing labels (e.g. `aria-label={\`CPU: ${used} of ${total} (${pct} percent)\`}`) and add `role="progressbar"` with `aria-valuenow`/`aria-valuemin`/`aria-valuemax` to bar/gauge tracks. Replace `title`-only values with visible text or aria-labels.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
