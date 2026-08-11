# e88s03 — AAA re-certification — matrix + axe + conformance statement

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

Prove the closing pass: run the full verification stack after e88s01/e88s02 and publish the conformance result. Output: 14/14 theme contrast matrix (default + 13 accents, light + dark links) ≥7:1, axe 17/17 zero violations, UI suite + build + lint green, and the conformance statement wired into the release.

## Context

The verification stack exists: `tests/e2e/axe-scan.spec.ts` (17 routes, wcag2a+aa+aaa), vitest (707 tests), `npm run build`, `npm run lint`, and the contrast-matrix script (extend for accent themes in e88s01). This story runs it end-to-end on merged main and records the result.

## Requirements

#### ADDED: Certified conformance record (AAA-closing)
All gates green on merged main: 14/14 themes' links ≥7:1 (light + dark), axe 17/17 zero violations, 707+ UI tests, build + lint clean. Result recorded in CONFORMANCE.md (from e88s02) with the run date + version.

## Implementation Steps

1. After e88s01/e88s02 merge: run the full matrix script (default + 13 themes × light/dark links) → assert 14/14 PASS ≥7:1. → verify: `node specs/epics/e88-aaa-closing/e88s01-contrast-matrix.mjs | tail -1 | grep -q "PASS"`
2. Full UI suite + build + lint. → verify: `cd ui && npx vitest run 2>&1 | tail -2 | grep -q "passed" && cd ui && npm run build && cd ui && npm run lint`
3. axe scan 17 routes. → verify: `npx playwright test axe-scan --config tests/e2e/playwright.config.ts 2>&1 | tail -1 | grep -q "17 passed"`
4. Update CONFORMANCE.md with the certified run (date, version, gate results); tick e88 acceptance in the epic. → verify: `grep -q "certified" specs/CONFORMANCE.md`
5. Release: land on main (solo), archive e88 capsule, push → CI + semantic-release (target 2.87.x minor). → verify: `git log --oneline -1 && gh release view --json tagName`

## Risks

- The axe scan runs against the embedded dist (built from ui/src) — ensure `npm run build` precedes the scan so the served SPA includes the changes.
- If any theme's brandLink misses 7:1 at re-check, the matrix FAILs loudly (intentional — no silent partial).

## Acceptance Criteria

- [ ] 14/14 themes ≥7:1 (light + dark links)
- [ ] axe 17/17 zero violations on the rebuilt dist
- [ ] UI suite + build + lint green
- [ ] CONFORMANCE.md certified + release cut
