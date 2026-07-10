---
bug_id: BUG-2026-07-10T195500
status: fixed
severity: high
scope: ui
title: v2.76.0 deploy blocked — CopyButton prop mismatch in SiteDeployKeysTab
---

# BUG-2026-07-10T195500: v2.76.0 release/deploy failed — TS error blocked build

## Problem

**Actual:** Production (bigbase.click) is running v2.75.0, and the new Deploy Keys /
API key feature is not visible in the Site Detail page, despite v2.76.0 being
tagged and merged to `main`.

**Expected:** v2.76.0 binary (with Deploy Keys UI) deployed and running in production.

**Reproduce:**
1. `cd ui && npm run build`
2. Observe: `src/components/SiteDeployKeysTab.tsx(111,25): error TS2322: Type '{ text: string; }' is not assignable to type 'IntrinsicAttributes & CopyButtonProps'.`

**CI Evidence:** [Run 29119589178](https://github.com/danielvm-git/bigbase/actions/runs/29119589178) — semantic-release tagged v2.76.0, but the "Build Admin UI" step failed on this TS error, so the binary was never built or shipped to the VPS. The job failed before the deploy/SSH steps ran.

## Root Cause Analysis

`ui/src/components/SiteDeployKeysTab.tsx:111` calls `<CopyButton text={result.key} />`,
but `ui/src/components/CopyButton.tsx` declares `CopyButtonProps` with a `value` prop,
not `text`. This is a prop-name mismatch introduced when `SiteDeployKeysTab.tsx` was
added in commit `68d9f6ae` (feat(ui): add Deploy Keys tab to Site Detail page).

Because semantic-release runs and tags the version *before* the build/test steps in
`release-deploy.yml`, the git tag `v2.76.0` was created and is visible on GitHub even
though the actual binary build failed and deploy never happened — hence the version
mismatch between GitHub (2.76.0) and the live site (2.75.0).

**Root cause (code layer):** Wrong prop name passed to `CopyButton` (`text` instead of `value`).

**Risk level:** Low to fix (1-line change), but high impact — blocks all deploys since v2.76.0 was tagged, and hides the deploy-keys feature this epic (e74) was built for.

## TDD Fix Plan

### Cycle 1 — Fix CopyButton prop name

**RED:** `cd ui && npm run build` fails with TS2322 on `SiteDeployKeysTab.tsx:111`

**GREEN:** Change `<CopyButton text={result.key} />` to `<CopyButton value={result.key} />` in [ui/src/components/SiteDeployKeysTab.tsx:111](ui/src/components/SiteDeployKeysTab.tsx:111)

**verify:** `cd ui && npm run build`

**REFACTOR:** None needed — single prop rename.

## Acceptance Criteria

- [x] `ui/src/components/SiteDeployKeysTab.tsx` uses `value` prop for `CopyButton`
- [x] `cd ui && npm run build` passes with no TypeScript errors
- [ ] CI "Build Admin UI" step completes successfully on push
- [ ] Deploy to Contabo VPS succeeds, health check passes
- [ ] Production shows v2.76.0+ and Deploy Keys tab is visible on Site Detail page

## Resolution

**Fixed:** 2026-07-10
**Root cause confirmed:** `CopyButton` prop mismatch (`text` vs `value`) in `SiteDeployKeysTab.tsx`.
**Fix applied:** Renamed prop to `value` at line 111.
**Evidence:** Local `npm run build` passes (`dist/` produced, CSS verification OK).
**Next:** Commit and push to `main` to re-trigger `release-deploy.yml` and ship v2.76.0+ to production.
