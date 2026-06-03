# BUG-2026-06-03T122114: Create site shows repo search only — Connect GitHub hidden, repos 500

## Problem

**Actual:** On **Create site** → **Connect GitHub (Recommended)**, after completing GitHub App install, the user sees only a **search input** and the message **"No repositories match."** There is no visible **Connect GitHub** button, no repo list, and no explicit search/refresh action (filter is live as-you-type).

Browser console shows **`/api/github/repos` → 500** (and various unrelated 401s on legacy/wrong paths).

**Expected:** After install, the user should see their GitHub repositories (or a clear error with **Reconnect** / **Connect GitHub**). If not connected, the **Connect GitHub** card with install button should remain visible.

**Reproduce:**

1. Log in at `https://bigbase.click/admin/`
2. Deploy → Create site → select **Connect GitHub**
3. Complete GitHub App installation (if not already)
4. Return to wizard — observe search-only UI with no repos

**Prior art:** Related to fixed bugs [BUG-2026-06-02T211731](BUG-2026-06-02T211731-github-app-not-configured-prod.md) (prod credentials) and [BUG-2026-06-03T120000](BUG-2026-06-03T120000-github-callback-requires-auth.md) (callback auth). Install callback now works; this is a **follow-on UX + API failure** after `connected: true`.

## Root Cause Analysis

**Phase 1 — Reproduce:** Confirmed from user screenshot and code review. Create-site GitHub panel has two mutually exclusive blocks controlled by `ghConnected` from `GET /api/github/status`. When `connected === true`, the **Connect GitHub** card is hidden and the repo search/list is shown.

**Phase 2 — Isolate:**

1. **UI:** `getGitHubRepos()` calls `GET /api/github/repos`. On non-OK responses (including **500**), it returns `{ data: [], previewMode: false }` with **no error surfaced**. `filteredGh` is empty → UI shows **"No repositories match."** (misleading — implies filter miss, not API failure).

2. **API:** Protected repos handler loads `installation_id` from DB, calls GitHub `installationToken` + `installation/repositories`. Any failure (unreadable PEM on VPS, invalid installation id, GitHub API 401/403, missing app permissions) logs and returns **500** `failed to list repositories`.

3. **State machine gap:** `connected` is set when a row exists in `github_installations` (callback persists `installation_id`). That does **not** guarantee GitHub API repo listing succeeds. UI treats `connected` as “show repo picker,” hiding the install CTA.

**Phase 3 — Hypotheses (ranked):**

1. **(Confirmed — UX)** `ghConnected === true` hides Connect GitHub while `getGitHubRepos` fails silently → search-only dead end.
2. **(Likely — API)** `listInstallationRepos` fails on production (PEM permissions, GitHub App repo permissions, or stale/test `installation_id` in DB) → 500.
3. **(Ruled out for primary symptom)** User never installed app — would show Connect card (`connected: false`).

**Phase 4 — Verify:** Code path traced: `handleRepos` lines 257–261 return 500 on `listInstallationRepos` error; `getGitHubRepos` swallows failure into empty array; CreateSitePage condition `(ghConnected || previewMode)` at line ~215 hides connect card. **Root cause verified** for UX; API failure mechanism verified by handler contract (exact GitHub error requires server logs).

**Risk level:** Medium — blocks GitHub deploy path after successful install appearance.

## TDD Fix Plan

### Cycle 1 — Surface repos fetch errors to UI

**RED:** Test `getGitHubRepos` (or CreateSitePage with mocked fetch) returns an `error` field when `/api/github/repos` responds 500.

**GREEN:** Extend `SitesDataResult` usage: on `!ok`, return `{ data: [], error: '...' }`. In CreateSitePage, show an error `Card` with message and actions.

**verify:** `cd ui && npm test -- CreateSitePage --run`

### Cycle 2 — Show Connect / Reconnect when repos unavailable

**RED:** Component test: when `connected: true` and repos fetch fails (or `data: []` with error), **Connect GitHub** or **Reconnect** button is visible.

**GREEN:** Change visibility rule, e.g. show connect/reconnect when `!ghConnected || ghReposError || (ghConnected && ghRepos.length === 0 && !loading)`.

**verify:** `cd ui && npm test -- CreateSitePage --run`

### Cycle 3 — Backend: resilient repos listing

**RED:** Test `handleRepos` when `listInstallationRepos` returns error — prefer **502** with safe message and log detail; optional: return `{ data: [], error: "github_api" }` with 200 for softer UI (choose one contract).

**GREEN:** Harden `appJWT` / `installationToken` errors; verify PEM readable on start (log `configured` + key readable). Return structured JSON error code for UI.

**verify:** `go test ./components/github/... -count=1`

### Cycle 4 — Ops verification on prod

**GREEN:** SSH/logs: confirm `list github repos` error string; verify `/opt/bigbase/secrets/github-app.pem` owned by `bigbase`, App permissions include **Contents: Read** + **Metadata: Read**, installation id in DB matches GitHub.

**verify:** Authenticated `GET /api/github/repos` returns 200 with `data.length > 0`

**REFACTOR:** Use `configured && connected && reposOk` derived state in CreateSitePage; add loading skeleton while fetching repos.

## Acceptance Criteria

- [x] User always sees **Connect GitHub** or **Reconnect** when repos cannot be listed
- [x] Failed repo fetch shows explicit error (not only "No repositories match.")
- [x] `GET /api/github/repos` returns 200 with repos OR actionable **502** (`github_api_error`) for logged-in admin when GitHub API fails (contract tested; authed prod session not available in CI)
- [ ] Selecting a repo and Continue works end-to-end after fix (requires logged-in user when GitHub API returns repos)
- [x] `go test ./...` and `cd ui && npm test` pass

## Resolution

**Fixed:** 2026-06-03

**Root cause confirmed:** UI hid the Connect GitHub card when `connected: true` while `getGitHubRepos` swallowed non-OK API responses as an empty list, showing only “No repositories match.”

**Fix applied:**

- `ui/src/lib/sitesData.ts` — propagate `error` on failed `/api/github/repos`
- `ui/src/pages/CreateSitePage.tsx` — `showGitHubConnect` when `ghReposError`; Reconnect card; loading and distinct empty/filter copy
- `components/github/github.go` — **502** + `code: github_api_error` when `listInstallationRepos` fails

**Hardening added:** Typed `SitesDataResult.error`; API codes `github_api_error` / `github_not_installed`; UI branches on `code`; `CreateSitePage.test.tsx`; no-install returns 404 not silent `[]`; empty connected repo list shows Connect card

**Evidence:** `go test ./... -count=1`, `cd ui && npm test`, `npm run preflight`, `golangci-lint run ./components/github/...`, `npx tsc --noEmit` (ui) — all pass

**Commit:** `fix(github): surface repos API errors in create-site UI` (`d47becf`); follow-up `fix(github): reconnect UX tests and empty repo list` (audit + review gaps)

**Behavioral proof (prod, deploy run 26894968721):**

- Admin bundle `index-BffhwgB0.js` contains `Reconnect GitHub` and `Could not load GitHub repositories`
- `GET /api/github/callback?installation_id=42` → **302** (public, unchanged)
- Unauthenticated `GET /api/github/repos` → **401** (auth still required; not the reported symptom)

**Prod follow-up:** If a logged-in user still sees Reconnect after install, check VPS logs for `list github repos`, PEM at `/opt/bigbase/secrets/github-app.pem`, and GitHub App **Contents/Meta read** permissions.
