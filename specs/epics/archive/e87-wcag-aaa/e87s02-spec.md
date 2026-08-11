# e87s02 — Focus appearance enhanced (2.4.13)

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

WCAG 2.2 AAA 2.4.13 requires the keyboard focus indicator to have ≥3:1 contrast against adjacent colors and sufficient area (WCAG 2.2 AA 2.4.11/2.4.12 set the minimums; AAA is stricter). Current `:focus-visible` ring uses `--border-focus` = neutral-300 (light) → **1.7:1 on white — fails**. Switch to the brand accent ring (brand-600, ~7.9:1) and normalize ring geometry.

## Context

Focus styles live in `ui/src/index.css` as `:focus-visible` rules on `.btn`, `.tab`, `.theme-trigger`, `.sidebar-nav a`, plus `:focus` on inputs (`--input-focus-border` = brand-500). The AA pass verified focus is visible; AAA demands the 3:1 threshold and consistent geometry (2px ring + 2px offset, per the 2.4.11/2.4.13 indicators).

## Requirements

#### ADDED: Focus indicator meets 3:1 contrast with sufficient area (2.4.13)
Every interactive element's `:focus-visible`/`:focus` ring is ≥3:1 against both the element background and adjacent colors, with a consistent 2px ring + ≥2px offset. Ring color: `--border-focus` → brand-600 (light) / brand-400-ish (dark, ≥3:1 vs neutral-850).

## Implementation Steps

1. Update `--border-focus` tokens: light → brand-600; dark → a lighter indigo (≈rgb(129,140,248), ~5.5:1 on neutral-850). Update `tokens.css` + `theme.css` + `tokens.ts`. → verify: `grep -q "border-focus: var(--brand-600)" ui/src/styles/tokens.css && cd ui && npm run build`
2. Normalize ring geometry app-wide: 2px outline + 2px offset on all `:focus-visible` rules; ensure inputs use the same ring (not just border-color) so text fields show a visible indicator. → verify: `grep -c ":focus-visible" ui/src/index.css`
3. Verify no focus ring is clipped by overflow containers (e.g. scrollable log viewers, table wrappers) — add `outline-offset` compensation where needed. → verify: `cd ui && npx vitest run src/components/Button.test.tsx src/components/Tabs.test.tsx`
4. Regression: keyboard walkthrough — Tab through every interactive element on 3 representative pages (Dashboard, Monitoring, Settings), assert the indicator is visible at each stop. → verify: `npx playwright test axe-scan --config tests/e2e/playwright.config.ts` (axe `focus-order` + `color-contrast` rules)

## Risks

- Brand-600 ring on brand-500 buttons (primary button focus) — ring must contrast against the button fill; use a light ring on accent fills (ring-white or brand-200) or a doubled indicator.
- Focus ring on dark mode accent backgrounds needs a different treatment than light mode.

## Acceptance Criteria

- [ ] All `:focus-visible` rings ≥3:1 vs adjacent, 2px+2px geometry
- [ ] axe-scan 0 violations incl. focus-related rules
- [ ] Manual Tab walkthrough shows visible indicator on every stop
