# No document.title management on route change

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** MODERATE
**WCAG:** 2.4.2 Page Titled — Level AA

## Description

No page sets `document.title` on route change (no `useDocumentTitle` hook exists; grep returns zero matches across ui/src). The browser tab and the screen-reader page-navigate announcement never reflect the current page — every route announces the static "BigBase Admin" title.

## Affected Files
- `ui/src/pages/*` (all 21 routed pages)
- `ui/index.html` (static title)

## Recommended Fix
Add a `useDocumentTitle(pageName)` effect hook (or `<title>`-syncing router component) and call it from each page; set route-aware titles like "Users — BigBase Admin".

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
