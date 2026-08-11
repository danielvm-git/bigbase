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
fixed

## Resolution
Fixed 2026-08-11. Darkened the semantic `--fg-tertiary` mapping (neutral scale untouched; `--neutral-400` remains for non-text borders).

- Light mode (`ui/src/styles/tokens.css`): `--fg-tertiary: var(--neutral-400)` → `var(--neutral-600)` (rgb 108,108,113).
  - neutral-600 on white (bg-surface): **5.22:1** PASS
  - neutral-600 on neutral-25 (bg-default): **5.01:1** PASS
- Dark mode (`ui/src/styles/theme.css`): `--fg-tertiary: var(--neutral-500)` → `var(--neutral-400)` (rgb 151,151,155).
  - neutral-400 on neutral-850 (bg-surface): **5.77:1** PASS
- `ui/src/tokens/tokens.ts` TS token mirror `fg.tertiary` aligned to `var(--neutral-600)`.
- Consumer audit (grep): every text consumer (`.input::placeholder`, hints, breadcrumbs, empty states, `.btn-ghost:disabled`, `.status-indicator-pending`, table headers, etc.) reads `var(--fg-tertiary)` and is covered by the token change. No CSS rule consumes `--input-placeholder` for text (definition retained). The only direct `--neutral-500` text use, `.sql-textarea::placeholder`, sits on the code editor's fixed `--neutral-900` background where it passes at **4.53:1** (re-pointing it to the new light-mode `--fg-tertiary` would regress it to 3.36:1, so it was left as-is).
- Verified with node contrast script; `npm run build` green.

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
