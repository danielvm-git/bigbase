# e51s05: Accessibility & Responsive Audit

**Story ID:** e51s05 | **Epic:** e51 | **BCPs:** 2 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** e51s02 (core components) | **Blocks:** none

## §1 — Summary

Perform a systematic accessibility and responsive design audit of all admin
console pages and components. Run automated checks, identify violations,
fix high-severity issues, and add documentation for remaining items. Add
`@axe-core/react` integration test that runs on all route pages to catch
a11y regressions in CI. Verify responsive layout at canonical breakpoints
(320px mobile, 768px tablet, 1024px desktop).

## §2 — Motivation

The admin console was built rapidly across 44 epics. Accessibility was
considered but not systematically verified. An audit identifies concrete gaps
before they become entrenched. Adding automated axe-core testing prevents
regressions as new pages are built in future epics.

## §3 — Background / Context

- 18 page components + ~35 shared components
- Existing a11y features: focus-visible rings, aria-labels on some elements, semantic HTML structure
- Known gaps (from code review):
  - Sidebar toggle button has `aria-expanded` and `aria-controls` ✓
  - Modal has focus trap and Escape handling ✓
  - `Spinner` component planned with `role="status"` (e51s02)
  - `Switch` component planned with `role="switch"` (e51s02)
  - Some page headings skip hierarchy (h1 → h3 without h2)
  - Images/icons may lack alt text
  - Color-only status indicators (no text alternatives)
  - Responsive: sidebar collapses but mobile behavior minimal below 480px

## §4 — Zoom-Out Check

- **Module purpose**: Admin console SPA
- **Callers**: Browser users
- **Contracts**: All pages must be navigable by keyboard, readable by screen readers, usable at 320px width

## §5 — Prior Art

- WCAG 2.2 AA is the target standard
- axe-core is the industry standard automated a11y testing tool
- Existing `Layout.test.tsx` and page tests already render components in jsdom

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| Use `@axe-core/react` for automated checks | De facto standard; integrates with testing-library |
| Run axe on every page route | Catch regressions as new pages are built |
| Manual audit checklist for items axe can't catch | Keyboard navigation flow, focus order, content clarity |
| Fix severity "critical" and "serious" in this story | "Moderate" and "minor" logged as tech debt |
| Responsive: add proper breakpoints at 480px (mobile) | Current 768px breakpoint leaves gap for small phones |

## §7 — Architecture / Component Design

```
ui/
  src/
    test-setup.ts                  ← EXTEND: add axe-core setup
    __tests__/
      a11y/
        axe-pages.test.tsx         ← NEW: axe scan on every route
        axe-components.test.tsx    ← NEW: axe scan on isolated components
  specs/
    epics/e51-design-system/
      a11y-audit-report.md         ← NEW: findings log
```

## §8 — Data Model / Types

N/A — audit task, no new types.

## §9 — API / Interface Contract

No API changes. `@axe-core/react` is a devDependency only.

## §10 — State Management

N/A.

## §11 — Error Handling

axe-core violations are logged as test failures. Non-critical violations are
excluded via axe configuration rules.

## §12 — Testing Strategy

| Test | Description |
|------|-------------|
| axe-pages.test.tsx | Render each page route, run axe, assert 0 violations |
| axe-components.test.tsx | Render each shared component in isolation, run axe |
| Keyboard navigation audit | Manual: Tab through every page, verify focus order |
| Screen reader audit | Manual: VoiceOver/NVDA pass on Dashboard, SiteDetail, Settings |
| Responsive audit | Manual: verify layout at 320px, 480px, 768px, 1024px, 1440px |

## §13 — Performance Considerations

axe-core adds ~50ms per page scan — acceptable for CI, not for dev hot-reload.

## §14 — Security Considerations

N/A.

## §15 — Accessibility

This is the accessibility story — see §17 acceptance criteria for specifics.

## §16 — Internationalization (i18n)

N/A.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: Zero critical or serious axe violations on any page
  Given the axe-core test suite
  When run against every page route
  Then there are 0 violations at the "critical" and "serious" severity levels

Scenario: All interactive elements are keyboard accessible
  Given any page in the admin console
  When the user navigates using only Tab and Shift+Tab
  Then every interactive element receives focus in logical order and can be activated with Enter/Space

Scenario: Status indicators have text alternatives
  Given any StatusBadge or Badge component
  When rendered with only a color indicator
  Then a text label is also visible (not color-only)

Scenario: Responsive layout works at all breakpoints
  Given viewport widths of 320px, 480px, 768px, 1024px
  When each page is rendered
  Then content is readable without horizontal scroll, and all interactive elements are accessible

Scenario: Heading hierarchy is valid
  Given any page
  When axe-core scans for heading-order violations
  Then there are no violations (headings increase by no more than 1 level at a time)
```

## §18 — Verification Script (Step-by-Step)

1. Install axe-core: `cd ui && npm install --save-dev @axe-core/react`
2. Run automated axe tests: `cd ui && npx vitest run src/__tests__/a11y/`
3. Run full test suite: `cd ui && npm test`
4. Manual keyboard audit: Tab through Dashboard, Deploy, SiteDetail, Settings — note any issues
5. Manual responsive audit: resize browser to 320px, 480px, 768px — verify all pages
6. Document findings in `specs/epics/e51-design-system/a11y-audit-report.md`
7. Fix all critical/serious violations
8. Re-run axe tests to confirm 0 violations
9. Build: `cd ui && npm run build && cd .. && go build ./...`

## §19 — Out of Scope

- WCAG AAA compliance
- Screen reader testing with real users
- Color contrast ratio testing beyond axe automated checks
- International accessibility standards beyond WCAG 2.2 AA
- Fixing all "moderate" and "minor" violations (logged as tech debt)

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| axe flags false positives on 3rd-party code | Configure axe rules with known-safe exclusions |
| Fixing a11y breaks visual layout | Review visual diffs; prefer semantic fixes over layout changes |
| Manual audit is subjective | Use WCAG 2.2 AA checklist as objective criteria |
