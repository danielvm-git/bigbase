# BUG-2026-06-04T112700: Sites UI missing prototype CSS (layout “creepy”)

## Problem

**Actual:** Create site and Sites pages rendered choice cards, wizard steps, and repo lists as overlapping inline text (“creepy message”). Stepper showed duplicate numbers (`1 1Source`). Sites list was a legacy deployments table, not the prototype card grid.

**Expected:** Layout matches `specs/archive/prototype/project/BigBase Console.html` — grid choice cards, horizontal wizard rail, vertical repo picker, Sites card grid, friendly OAuth copy.

**Reproduce:**
1. Open `/deploy/new` on production or `cd ui && npm run dev`
2. Observe mashed option text and broken stepper
3. `grep choice-card ui/src/index.css` before fix → no matches

## Root Cause Analysis

React components (`ChoiceCard`, `WizardSteps`, `CreateSitePage`, `SiteCard`) were added and exported (see BUG-2026-06-01T143800) but **prototype component CSS was never ported** to the admin bundle. BEM modifiers (`choice-card--selected`) also differed from prototype class names (`.selected`), so even partial CSS would not have applied.

Secondary gaps: `DeployPage` still implemented deployments table; wizard had 4 steps vs prototype 3; OAuth copy was vendor-heavy.

**Risk level:** Low — presentation layer only; no API contract change.

## TDD Fix Plan

1. **RED:** Tests assert `.choice-grid` uses `display: grid` and `.repo-picker` uses `flex-direction: column` (with Vitest `css: true`).
   **GREEN:** Add `ui/src/styles/sites.css`, import from `index.css`.
   **verify:** `cd ui && npm test -- CreateSitePage.test.tsx`

2. **RED:** DeployPage tests expect Sites heading and `.site-grid`.
   **GREEN:** Rewrite `DeployPage` as Sites list with `getSites()` + `SiteCard`.
   **verify:** `cd ui && npm test -- DeployPage.test.tsx`

3. **RED:** CreateSitePage expects 3 steps and updated OAuth copy.
   **GREEN:** Align wizard, breadcrumb, `Icon` usage, centered GitHub panel.
   **verify:** `cd ui && npm test -- CreateSitePage.test.tsx && npm run build`

**REFACTOR:** `ui/src/lib/format.ts` for `timeAgo`, thumb colors, display URL.

## Acceptance Criteria

- [x] Create site: grid cards, horizontal stepper, vertical repo list
- [x] No overlapping option/repo text in light/dark mode (CSS loaded in dist)
- [x] Sites list: card grid with search and env filters
- [x] All UI tests pass (`155` tests)
- [x] `npm run build` succeeds
- [x] OAuth copy benefit-first, permissions in secondary line

## Resolution

**Fixed in source:** 2026-06-04  
**Evidence:** `cd ui && npm test -- --run`; `npm run build` runs `verify-sites-css-in-dist.mjs`; contract tests in `sites.contract.test.ts`.

**Production still broken until redeploy:** Go embeds `ui/dist` (`ui/embed.go`). If VPS serves an old binary or stale `dist`, users still see the pre-fix UI (4-step wizard, “How do you want to add your app?”, mashed repo text). After merge:

```bash
cd ui && npm run build
go build -o bigbase .
# restart bigbase on VPS (systemd/coolify)
```

**Regression guards added:**
- `ui/src/styles/sites.contract.test.ts` — source CSS must define grid/flex selectors
- `ui/scripts/verify-sites-css-in-dist.mjs` — built `dist/assets/index-*.css` must contain tokens
- `npm run build` fails if dist CSS is missing Sites rules

**Commit suggestion:** `fix(ui): port Sites prototype CSS and align create wizard`
