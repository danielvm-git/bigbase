# Text color contrast fails WCAG 1.4.3 — fg-tertiary token 2.91:1

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** CRITICAL
**WCAG:** 1.4.3 Contrast (Minimum) — Level AA

## Description

`--fg-tertiary` (neutral-400, rgb 151,151,155) fails the 4.5:1 minimum contrast ratio on all light-mode backgrounds:

| Pair | Ratio | Verdict |
|---|---|---|
| neutral-400 on white | 2.91:1 | FAIL |
| neutral-400 on bg-default (neutral-25) | 2.79:1 | FAIL |
| neutral-400 on bg-surface-secondary | 2.65:1 | FAIL |
| dark: neutral-500 on neutral-850 | 4.33:1 | large-text only |

Used for: input placeholder text (`.input::placeholder`), hints, tertiary labels, disabled ghost buttons (`.btn-ghost:disabled`), status-pending indicators (`.status-indicator-pending`). Fails everywhere it appears in light mode.

## Affected Files
- `ui/src/styles/tokens.css` (--neutral-400 definition)
- `ui/src/styles/theme.css` (dark-mode --fg-tertiary)
- `ui/src/index.css` (placeholder/disabled consumers)

## Recommended Fix
Darken `--neutral-400` from rgb(151,151,155) to ≥ rgb(122,122,127) (≈4.5:1), or introduce a distinct compliant `--fg-placeholder`/`--fg-tertiary` token and re-point consumers.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
