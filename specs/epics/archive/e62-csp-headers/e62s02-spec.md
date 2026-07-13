# e62s02: Admin UI — CSP Policy Builder
## Story ID: e62s02 | Epic: e62 | BCPs: 1 | Status: planned

### Story e62s02: Admin UI — CSP Policy Builder — Implementation Steps

**type:** feat
**context:** domain
**Context**: This story implements the visual CSP Policy Builder in the Admin UI. Since Content-Security-Policy strings are easy to malform and hard to write, the UI provides a builder for common directives (`default-src`, `script-src`, `style-src`, `font-src`, `img-src`, `connect-src`) with explicit support for injecting `{nonce}`. This policy is saved to the site's `csp_template` configuration and powers the backend feature built in e62s01.

## Zoom-Out Check
- **Purpose**: Admin UI to read, build, and save the `csp_template` for a specific site.
- **Callers**: `SiteDetailPage.tsx` (in the Settings tab).
- **Contracts**: The API payload to `PUT /api/sites/:id` MUST include `csp_template`.

## Steps

1. In `ui/src/types/sites.ts` (or `ui/src/lib/sitesData.ts`), update the `Site` interface to include `csp_template: string`. → verify: `npm run typecheck`
2. Create a new `CSPPolicyBuilder.tsx` component in `ui/src/components/` that takes `value` and `onChange` props. It should allow adding common directives, predefined sources (like `https://fonts.googleapis.com`), and a `{nonce}` checkbox for scripts/styles. → verify: `npm run lint`
3. In `ui/src/pages/SiteDetailPage.tsx`, within the Settings tab, add a "Security Headers" section. Mount the `CSPPolicyBuilder` bound to `site.csp_template`. Add a Save button that calls `updateSite` (or similar existing function) with the updated `csp_template`. → verify: `npm test -- SiteDetailPage`
4. Add E2E or unit tests in `ui/src/pages/SiteDetailPage.test.tsx` that assert the CSP builder renders and updates the API payload correctly when saved. → verify: `npm test -- SiteDetailPage`

## Verification Script (Step-by-Step)

1. Boot the BigBase application (`go run . serve`) and log into the Admin UI.
2. Navigate to "Sites" and select a test site.
3. Go to the "Settings" tab.
4. Scroll to the "Security Headers" section and use the CSP Policy Builder to configure `default-src 'self'` and `script-src 'self' 'nonce-{nonce}'`.
5. Click Save.
6. Refresh the page to verify the configuration was successfully persisted and reloaded from the API.

## Out of scope
- Pre-flight validation of the CSP string in the browser (basic text format is fine).
- Editing other security headers (HSTS, X-Frame-Options) – this is strictly for CSP.

## Risks
- Users might accidentally lock themselves out of their own deployed assets if the UI encourages overly strict policies without warning.
