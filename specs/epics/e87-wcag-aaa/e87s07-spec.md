# e87s07 — Visual presentation + location (1.4.8/2.4.8)

**type:** feat
**risk:** P2
**context:** domain
**BCPs:** 3

## Summary

Two AAA criteria with a "partial + documented" outcome: 1.4.8 Visual Presentation (user-controllable text spacing/width; ≥1.5 line-height; text resizable to 200%) and 2.4.8 Location (user knows where they are in a set of pages). 1.4.8 is the hardest AAA criterion for a design system — this story makes the feasible parts compliant (line-height/letter-spacing floor, em-based scaling for text) and documents the rest; 2.4.8 adds a location indicator on top-level pages.

## Context

Type is px-based (`--text-xs: 12px` … `--text-3xl: 40px`); body line-height defaults to browser default (~1.2) which is below 1.5. Browser zoom (200%) works for zoom but 1.4.8 wants text-relative sizing. 2.4.8: breadcrumbs exist on detail pages (`Breadcrumb.tsx`); top-level pages (Dashboard, Users, …) rely on sidebar highlight alone.

## Requirements

#### ADDED: Text presentation controls (1.4.8) — partial
Body text line-height ≥1.5 and paragraph spacing ≥1.5× (feasible); text sizes switch to `rem`/`em` so they scale with user font settings (feasible); user-selectable foreground/background + text spacing controls (1.4.8 bullets 1-4) documented as not implemented with a rationale (admin tool, browser-provided zoom/contrast accepted).

#### ADDED: Location indicator (2.4.8)
Every page shows the user's location in the app: breadcrumb on detail pages (exists) + a location indicator on top-level pages (active nav + page header pair, or an explicit "You are here" breadcrumb of depth 1).

## Implementation Steps

1. Line-height floor: set `line-height: 1.5` on body/paragraph defaults and verify headings don't clip with the raised line-height. → verify: `grep -q "line-height" ui/src/styles/tokens.css && cd ui && npm run build`
2. Convert text-size tokens from px to rem (`--text-xs: 0.75rem` … `--text-3xl: 2.5rem`) so text scales with browser font-size settings; keep spacing tokens px (layout doesn't need to scale). → verify: `grep -c "rem" ui/src/styles/tokens.css` + `cd ui && npx vitest run` (visual regressions caught by component snapshots)
3. Top-level location indicator: render a depth-1 breadcrumb (or "Section · Page" in the page header) on pages without detail breadcrumbs (Dashboard, Users, Deploy, Monitoring, …). → verify: `cd ui && npx vitest run src/components/Breadcrumb.test.tsx src/components/PageHeader.test.tsx`
4. Document the 1.4.8 non-conformance: user-selectable color/spacing controls not implemented, rationale (browser OS-level settings cover; admin console density constraint). Record in the epic's conformance notes. → verify: conformance note artifact exists

## Risks

- px→rem conversion can subtly shift layout (rem scales from the root font-size; `--text-m: 16px` → `1rem` must equal 16px at default). Verify with the e2e suite.
- 1.4.8 full compliance (user color/spacing controls) is a product decision — this story explicitly partials it rather than gold-plating.

## Acceptance Criteria

- [ ] Body line-height ≥1.5; text tokens in rem
- [ ] Location indicator on all top-level pages
- [ ] 1.4.8 partial conformance documented with rationale
