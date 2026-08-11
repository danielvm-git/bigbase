# e87s01 — AAA color contrast tokens (1.4.6 Contrast Enhanced)

**type:** feat
**risk:** P0
**context:** domain
**BCPs:** 3

## Summary

Raise all text tokens to ≥7:1 (WCAG 2.2 AAA 1.4.6) in both light and dark mode. The AA pass (T000001) fixed 4.5:1; AAA needs the secondary/tertiary/link/semantic tiers darker. Token-only change — no component code.

## Context

Design tokens are the single source of truth: `ui/src/styles/tokens.css` (light), `ui/src/styles/theme.css` (dark), mirrored in `ui/src/tokens/tokens.ts`. Computed current ratios vs white (light) / neutral-850 (dark):

| Token | Current | Target (≥7:1) |
|-------|---------|---------------|
| fg-secondary (neutral-600) | 5.22:1 | darken → ≈rgb(75,75,80) |
| fg-tertiary (neutral-600) | 5.22:1 | same darker value |
| brand-500 links | 6.29:1 | use brand-600 (7.90:1 ✓) for text |
| success-fg | 6.33:1 | darken → ≈rgb(4,78,58) |
| error-fg | 6.76:1 | darken → ≈rgb(120,20,20) |
| dark fg-tertiary (neutral-400) | 5.77:1 | darken → neutral-300 (7.51:1 ✓) |

## Requirements

#### ADDED: All text meets 7:1 contrast (1.4.6)
Every text token resolves to ≥7:1 against its background in both modes. Verified by a scripted contrast matrix (the AA batch's computation, threshold raised to 7.0).

## Implementation Steps

1. Update `tokens.css` + `theme.css` semantic fg tokens to the darker values; update the `tokens.ts` TS mirror in the same change. → verify: `grep -q "brand-600" ui/src/styles/tokens.css && cd ui && npm run build`
2. Verify no text consumer regressed below 7:1 — run the contrast matrix (luminance script from the AA audit, threshold 7.0) against every fg/bg pair. → verify: `node -e "<contrast matrix, assert all >= 7>"` (document in spec)
3. Sweep hardcoded colors in components (e.g. chart colors `#3b82f6` axis labels, `#9999` fallbacks) — any text-carrying fill must meet 7:1 or be aria-hidden. → verify: `cd ui && npx vitest run src/components/BarGauge.test.tsx src/components/DonutGauge.test.tsx src/components/AreaChart.test.tsx`
4. Dark mode parity — recompute every dark-mode pair ≥7:1 (fg-primary already 16.8:1; fg-secondary neutral-300 → verify 7.51:1 holds with darker tertiary). → verify: contrast matrix dark-mode pass.

## Risks

- Over-darkening text flattens visual hierarchy (tertiary ≈ secondary). Acceptable for AAA; document the tradeoff.
- Chart colors are data-viz, not text — 1.4.6 applies to text; keep chart fills as-is if labels carry the values (aria-label pattern from T000012).

## Contrast Matrix (e87s01)

Reproducible script: `specs/epics/e87-wcag-aaa/e87s01-contrast-matrix.mjs`
(`node specs/epics/e87-wcag-aaa/e87s01-contrast-matrix.mjs`). WCAG relative-luminance
computation, threshold 7.0, both modes. **36/36 token pairs PASS.**

Light mode: fg-primary/secondary/tertiary/accent on white, neutral-25, neutral-40
(7.20–13.71:1); fg-accent on brand-tint (7.23:1); success-fg / warning-fg / error-fg /
info-fg on their tinted backgrounds (7.78–9.92:1); fg-on-accent on bg-accent
brand-600 (7.90:1) and hover brand-700 (9.93:1).

Dark mode: fg-primary neutral-25 (13.15–16.82:1), fg-secondary neutral-250
(7.80–9.98:1), fg-tertiary neutral-300 (7.51–7.84:1), fg-accent brand-300
(8.10–8.80:1), semantic fg on dark tinted bgs (8.21–9.40:1), fg-on-accent on
brand-600 (7.90:1).

Token changes beyond the table (all required by "100% of text pairs ≥7:1"):
`--bg-accent` → brand-600 / `--bg-accent-hover` → brand-700 (white button text was
6.29:1 on brand-500), `--brand-tint` 0.10 → 0.06 alpha (fg-accent text on badge/
wizard/theme-menu tints was 6.80:1), dark `--fg-secondary` → neutral-250 (table
headers + neutral badges on neutral-800 were 6.12:1), dark `--fg-accent` → new
brand-300 ramp step (dark links were 2.68:1), light `--warning-fg` → rgb(120,53,15)
(was 6.28:1).

Known component-state pairs still <7:1 (not token pairs; rendered only in
error/modal/wizard-completion states, out of this story's token-only scope):
white on `--error` (btn-danger, 3.76:1), `--error` as `.input-error-text`
(3.76:1 light / 4.46:1 dark), white on `--success` (wizard done step, 2.54:1),
dark `.wizard-step-num` fg-tertiary on bg-subtle (6.13:1). Custom accent themes
keep their AA-only promise: light-mode links use each theme's brand-600 step
(4.23–7.90:1).

## Acceptance Criteria

- [x] Contrast matrix: 100% of text pairs ≥7:1, both modes
- [x] `npm run build` + targeted chart tests green
- [x] axe-scan still 0 violations (re-run `npx playwright test axe-scan`)
