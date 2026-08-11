# e87s05 — Session timeout warning + re-authentication (2.2.5/2.2.6)

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

AAA timing criteria: 2.2.6 (users warned before a time limit expires, unless essential) and 2.2.5 (re-authentication after expiry with no data loss). The app's JWT expires (24h default) with no warning — a session silently dies mid-work. Add a pre-expiry warning toast + graceful re-auth flow that preserves the current route/state.

## Context

Auth: JWT access token (exp claim), `ui/src/context/ToastContext.tsx` provides toasts. Token lifetime configurable (`--jwt-access-expiry`). The UI stores the token in a cookie; 401s currently redirect to login. 2.2.3 (No Timing) is claimed as the security exception; 2.2.6/2.2.5 still require warning + safe re-auth for the session lifetime.

## Requirements

#### ADDED: Timeout warning before session expiry (2.2.6)
The UI warns (toast + dialog) N minutes before JWT expiry with a "Stay signed in" action that refreshes the token. Warning is dismissible; appears once per session.

#### ADDED: Re-authentication without data loss (2.2.5)
On expiry, the user re-authenticates and returns to the same route with their work state intact (no form data loss; current page restored after login).

## Implementation Steps

1. Read the JWT `exp` claim client-side (decode payload) and expose remaining time via the auth context; compute the warning threshold (configurable, default 5 min). → verify: `cd ui && npx vitest run src/context/ThemeContext.test.tsx` (pattern reference; add auth-context tests)
2. Add a `SessionTimeoutWarning` component: renders a toast/dialog when `remaining < threshold`, with "Stay signed in" → calls refresh endpoint (`POST /api/auth/refresh`) and resets the timer; countdown visible. → verify: `cd ui && npx vitest run src/components/SessionTimeoutWarning.test.tsx` (new)
3. Graceful re-auth: on 401 (expired), store the current route in `sessionStorage`, redirect to `/login` with a "session expired" message; after successful login, restore the saved route. → verify: `cd ui && npx vitest run src/pages/LoginPage.test.tsx`
4. E2E: with `--jwt-access-expiry=60s`, verify the warning appears, refresh extends the session, and an un-refreshed expiry returns to login then restores the route. → verify: `npx playwright test --config tests/e2e/playwright.config.ts tests/e2e/token-lifecycle.spec.ts` (extend or add session-timeout spec)

## Risks

- Refresh-token rotation races (the e50 token-lifecycle work) — the "Stay signed in" action must handle an already-expired access token (fall back to full re-auth).
- Restoring route after re-auth: deep links with params (site detail, function detail) must survive — use the full pathname+search, not just the route name.

## Acceptance Criteria

- [ ] Warning toast appears ≥1 min before expiry; "Stay signed in" refreshes
- [ ] Expired session → login → route restored with state intact
- [ ] Token-lifecycle E2E suite green
