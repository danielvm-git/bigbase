# BUG-2026-06-03T120000: GitHub install callback returns authorization required

## Problem

**Actual:** After clicking **Connect GitHub** in Admin → Deploy → Create site, the user completes GitHub App installation on github.com. The browser is redirected back to BigBase and shows raw JSON:

```json
{"error":"authorization required"}
```

**Expected:** GitHub should redirect to the app **setup/callback URL**, the server should persist the `installation_id`, then redirect the browser to `/admin/#/deploy/new` with GitHub connected.

**Reproduce:**

1. Log in at `https://bigbase.click/admin/`
2. Go to Deploy → Create site → **Connect GitHub**
3. Approve installation on GitHub
4. Observe JSON error instead of returning to the wizard

**Prior art:** Related to [BUG-2026-06-02T211731](BUG-2026-06-02T211731-github-app-not-configured-prod.md) (prod GitHub App credentials — **fixed**). This is a **new issue**: install flow reaches GitHub but the **return URL** is blocked.

## Root Cause Analysis

**Phase 1 — Reproduce:** Confirmed on production without a session cookie:

```bash
curl -s "https://bigbase.click/api/github/callback?installation_id=12345"
# → {"error":"authorization required"}

curl -s -o /dev/null -w "%{http_code}" -X POST "https://bigbase.click/api/github/webhook"
# → 401
```

Same failure occurs when returning from github.com after install (user report).

**Phase 2 — Isolate:** Flow: UI `window.location.href` → `GET /api/github/install` (authenticated) → 302 to GitHub → user installs → `GET /api/github/callback?installation_id=…` (cross-site navigation from github.com). The composition root registers **all** `/api/github/` traffic through `auth` middleware before the github component. The middleware returns `authorization required` when no `token` cookie or Bearer header is present.

**Phase 3 — Hypotheses (ranked):**

1. **(Confirmed)** `/api/github/callback` requires JWT like other API routes, but GitHub’s redirect does not include BigBase credentials. The session cookie uses `SameSite=Strict`, so it is **not** sent on cross-site top-level navigations from github.com even if the user was logged in.
2. **(Ruled out)** GitHub App still unconfigured — user reached GitHub install UI; prior 503 would have appeared earlier.
3. **(Confirmed secondary)** `/api/github/webhook` is also behind auth middleware, so GitHub push webhooks would receive 401 (not yet reported by user).

**Phase 4 — Verify:** Unauthenticated request to callback matches auth middleware’s exact error string from the auth component. Google OAuth callback is mounted on unauthenticated auth routes; GitHub callback was never exempted. **Root cause verified.**

**Risk level:** Medium — blocks entire GitHub connect journey after install; webhooks broken for auto-deploy.

## TDD Fix Plan

### Cycle 1 — Callback works without session

**RED:** Add an integration test (or extend `components/github` handler test) that calls `GET /api/github/callback?installation_id=999` on the **public** route registration (no auth wrapper) and expects HTTP 302 to `/admin/#/deploy/new`, not 401.

**GREEN:** In the composition root, register github routes that must be public **without** auth middleware:

- `GET /api/github/callback` — GitHub post-install redirect
- `POST /api/github/webhook` — GitHub event delivery (HMAC verified inside component)

Keep protected (auth required):

- `GET /api/github/install`, `GET /api/github/status`, `GET /api/github/repos`, `POST /api/github/repos/connect`

Implementation options: split `github.Handler()` into public mux + protected mux, or mount two `p.Handle` prefixes in the composition root.

**verify:** `go test ./components/github/... -count=1 && curl -s "https://bigbase.click/api/github/callback?installation_id=1" | head -1` (expect redirect or 302, not JSON error)

### Cycle 2 — Webhook accepts POST without JWT

**RED:** Test `POST /api/github/webhook` with empty body returns not 401 (may be 400/200 depending on signature).

**GREEN:** Same public route registration as Cycle 1.

**verify:** `curl -s -o /dev/null -w "%{http_code}" -X POST https://bigbase.click/api/github/webhook` → not 401

### Cycle 3 (optional hardening) — Cookie SameSite for future OAuth-style returns

**RED:** Document or test that `token` cookie `SameSite=Strict` blocks any cross-site return path.

**GREEN:** Consider `SameSite=Lax` for `token` cookie if other external OAuth returns need session (out of scope unless needed beyond GitHub callback fix).

**REFACTOR:** Align route registration pattern with Google OAuth (`/api/auth/oauth/google/callback` unauthenticated).

## Acceptance Criteria

- [ ] After GitHub App install, user lands on `/admin/#/deploy/new` (not JSON error)
- [ ] `GET /api/github/callback?installation_id=…` works without `Authorization` header or `token` cookie
- [ ] `POST /api/github/webhook` is not blocked by auth middleware (signature check still applies)
- [ ] `GET /api/github/install` and repo APIs still require authentication
- [ ] All tests pass: `go test ./... -count=1`

## Resolution

**Fixed:** 2026-06-03

- Split github routes: `PublicHandler()` (callback, webhook) without auth middleware; `ProtectedHandler()` for install/status/repos.
- Registered `/api/github/callback` and `/api/github/webhook` before protected `/api/github/` prefix in composition root.
- Tests: `TestGitHubCallbackPublicNoAuth`, `TestGitHubWebhookPublicNotUnauthorized`.
