# No skip-to-content link; LoginPage has no landmarks

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** SERIOUS
**WCAG:** 2.4.1 Bypass Blocks / 1.3.1 Info and Relationships — Level A

## Description

No skip-to-content link exists anywhere in `ui/src/` (grep for skip-link/skip-to returns zero matches), and `<main className="content">` (Layout.tsx:135) has no `id` and no `tabindex="-1"` — so there is no skip target. Keyboard/screen-reader users tab through the entire sidebar+footer on every route. Additionally, `LoginPage` renders OUTSIDE the Layout route (App.tsx:51) with no `<main>`/`<nav>`/`<footer>` landmark — regionless content.

## Affected Files
- `ui/src/components/AppShell.tsx` (skip link insertion point)
- `ui/src/Layout.tsx` (main id/tabindex)
- `ui/src/pages/LoginPage.tsx` (landmark wrapper)

## Recommended Fix
Add `<a className="skip-link" href="#main-content">Skip to content</a>` as the first focusable element, give the main landmark `id="main-content" tabIndex={-1}` (plus a `.skip-link:focus` visible style), and wrap LoginPage in `<main>`.

## Status
fixed

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
