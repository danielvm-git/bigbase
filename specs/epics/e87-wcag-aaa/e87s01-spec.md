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

## Acceptance Criteria

- [ ] Contrast matrix: 100% of text pairs ≥7:1, both modes
- [ ] `npm run build` + targeted chart tests green
- [ ] axe-scan still 0 violations (re-run `npx playwright test axe-scan`)
