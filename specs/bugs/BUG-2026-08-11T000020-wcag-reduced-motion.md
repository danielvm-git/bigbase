# No prefers-reduced-motion support in CSS

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** MODERATE
**WCAG:** 2.3.3 Animation from Interactions — Level AAA (best practice for vestibular safety)

## Description

CSS animations (`spin` spinner, `shimmer` skeleton, `status-pulse`, `toast-in`) have NO `@media (prefers-reduced-motion: reduce)` gating. A `usePrefersReducedMotion` hook exists in `ui/src/tokens/motion.ts` but is **never imported by any component** — the capability is dead code.

## Affected Files
- `ui/src/index.css` (animation keyframes and consumers)
- `ui/src/tokens/motion.ts` (unused hook)

## Recommended Fix
Add `@media (prefers-reduced-motion: reduce)` rules disabling/slowing animations app-wide, and/or wire the existing `usePrefersReducedMotion` hook into components that animate. Reduce/remove the infinite `spin` for vestibular safety.

## Status
fixed

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
