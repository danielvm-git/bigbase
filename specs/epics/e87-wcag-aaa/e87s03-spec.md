# e87s03 — Target size enhanced — 44px (2.5.5)

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

WCAG 2.2 AAA 2.5.5 requires pointer targets ≥44×44 CSS px. The admin console's `btn-sm` (~28px), icon-only buttons (~24px), and table action cells fall short. This is the **biggest design tradeoff** of the AAA push — 44px targets reduce density. Scope documents the decision: compact components get a "spacing exception" only where a larger adjacent target covers them (2.5.5 allows inline exceptions), otherwise min-dimensions enforced.

## Context

Affected components: `Button.tsx` (sm size), `CopyButton.tsx`, table action cells (`.actions-cell`), icon buttons (`.modal-close`, `.toast-close`, `.dropdown-item`), `ThemePicker` trigger, sidebar toggle, `Select`/`Input` (height ok, width fine). The 2.5.5 exception: targets < 44px pass if the same function is available on a larger target or the target is inline within a line of text (spacing exception).

## Requirements

#### ADDED: Pointer targets ≥44px (2.5.5)
All interactive targets are ≥44×44 CSS px, or qualify for the inline/spacing exception (documented per component). Compact density is preserved where the exception applies.

## Implementation Steps

1. Inventory: measure every interactive target class (`.btn-sm`, `.actions-cell button`, icon buttons, `.dropdown-item`) via a Playwright measurement script; produce a table of target → size → exception status. → verify: `npx playwright test --config tests/e2e/playwright.config.ts tests/e2e/target-size.spec.ts` (new spec measuring bounding boxes)
2. `Button.tsx`: bump `btn-sm` min-height to 44px (or introduce `min-height: 44px` on all button variants with an opt-out `density="compact"` for exception cases). → verify: `cd ui && npx vitest run src/components/Button.test.tsx`
3. Icon-only buttons (`.modal-close`, `.toast-close`, CopyButton, sidebar toggle): min 44×44 with padding, or an enlarged clickable area via `::before` expansion when the visible glyph stays small. → verify: `cd ui && npx vitest run src/components/CopyButton.test.tsx src/components/Modal.test.tsx`
4. Table action cells: add per-cell min-height/padding so the smallest actionable control ≥44px (or confirm the inline exception: text-link actions within a row qualify). → verify: target-size measurement spec passes
5. Document the exception map in the spec (which targets rely on inline/spacing exception) for the conformance record. → verify: spec file contains the exception table

## Risks

- **Density regression** — the reason this is a separate decision story. 44px rows on Users/Deploy tables lengthen pages. Mitigation: prefer the inline-exception for row-level text actions, reserve hard 44px for standalone controls.
- Sidebar collapsed icon-only links: nav icon targets must be ≥44px (currently ~36px) — likely needs padding increase.

## Acceptance Criteria

- [ ] Target-size spec passes (all targets ≥44px or documented exception)
- [ ] `npm run build` + component suites green
- [ ] Exception map recorded in this spec
