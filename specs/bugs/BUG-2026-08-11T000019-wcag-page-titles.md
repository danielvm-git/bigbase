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
fixed

## Resolution
Route-aware titles implemented in `ui/src/App.tsx` via a new `RouteTitle` component (pathname → page name map, with prefix fallbacks for dynamic routes like `/deploy/:siteId`, `/functions/:id`) and a new `useDocumentTitle` hook (`ui/src/hooks/useDocumentTitle.ts`) that sets `document.title` and restores the previous title on unmount. Title format: `"<Page Name> — BigBase Admin"`. Unknown paths get "Page Not Found — BigBase Admin". Verified: `npx vitest run src/hooks/useDocumentTitle.test.tsx` (3/3 pass) and `npm run build` (tsc -b + vite build OK).

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
