# BUG-2026-06-01T143800: Missing component exports in UI

## Problem

**Actual:** CI build fails with TypeScript errors during "Build Admin UI" step: `Module '"../components"' has no exported member 'ChoiceCard'` (and 3 others).

**Expected:** `npm run build` succeeds; all components imported from `../components` index are available.

**Reproduce:**
1. Pull main branch (commit bca77a2+)
2. Run `cd ui && npm run build`
3. Observe TypeScript compilation errors for ChoiceCard, WizardSteps, PreviewBanner, SitesListSkeleton

**CI Evidence:** [Run 26770913297](https://github.com/danielvm-git/bigbase/actions/runs/26770913297) Build Admin UI step failed with 5 TS2305 errors.

## Root Cause Analysis

Four new React components were added to the codebase (`ChoiceCard.tsx`, `WizardSteps.tsx`, `PreviewBanner.tsx`, `SitesListSkeleton.tsx`), and new pages (`CreateSitePage.tsx`, `SiteDetailPage.tsx`) import them via barrel export: `import { ChoiceCard, ... } from '../components'`.

However, [ui/src/components/index.ts](ui/src/components/index.ts) does not export these four components, only the original ones (Button, Card, Input, Badge, Tabs, PageHeader, EmptyState).

**Root cause (code layer):** Component files exist but barrel export is incomplete.

**Risk level:** Low — simple missing exports, easily fixed with 4 lines. No production downtime.

## TDD Fix Plan

### Cycle 1 — Export missing components

**RED:** `npm run build` should not fail with TypeScript errors about missing exports

**GREEN:** Add the four missing exports to [ui/src/components/index.ts](ui/src/components/index.ts):
- `export { ChoiceCard } from './ChoiceCard'`
- `export { WizardSteps } from './WizardSteps'`
- `export { PreviewBanner } from './PreviewBanner'`
- `export { SitesListSkeleton } from './SitesSkeleton'`

**verify:** `cd ui && npm run build`

**REFACTOR:** None needed — simple barrel export addition

## Acceptance Criteria

- [x] All four component files exist in `ui/src/components/`
- [x] All four components are exported from `ui/src/components/index.ts`
- [x] `npm run build` passes with no TypeScript errors
- [x] CI "Build Admin UI" step completes successfully
- [x] Full CI run: all jobs green

## Resolution

**Fixed:** 2026-06-01 at 17:41 UTC
**Root cause confirmed:** Four new React components existed but were not exported from the barrel export file (index.ts), causing TypeScript compilation errors when pages imported them.
**Fix applied:** Added 4 export lines to ui/src/components/index.ts for ChoiceCard, WizardSteps, PreviewBanner, and SitesListSkeleton.
**Hardening added:** Components are now properly registered in the barrel export barrel—no missing exports by design.
**Evidence:** 
- Local: `npm run build` passes
- CI run [26771419432](https://github.com/danielvm-git/bigbase/actions/runs/26771419432): Test & Build ✓ SUCCESS (46s), Deploy to Contabo ✓ SUCCESS (34s)
- No TypeScript errors in logs
**Commit:** `fix(ui): export missing components from index.ts`

## Acceptance Criteria

- [x] All four component files exist in `ui/src/components/`
- [x] All four components are exported from `ui/src/components/index.ts`
- [x] `npm run build` passes with no TypeScript errors
- [x] CI "Build Admin UI" step completes successfully  
- [x] Full CI run: all jobs green (Test & Build ✓, Deploy to Contabo ✓)
