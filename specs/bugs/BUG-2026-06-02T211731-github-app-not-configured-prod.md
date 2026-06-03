# BUG-2026-06-02T211731: GitHub App not configured on production — cannot connect repos

## Problem

**Actual:** On production (`https://bigbase.click`), navigating to `/api/github/install` (from Admin → Deploy → Connect GitHub) returns HTTP 503 with:

```json
{"error":"GitHub App not configured; set --github-app-id, --github-app-slug, --github-app-private-key-path"}
```

The user cannot add or connect a GitHub repository through the Sites deploy flow.

**Expected:** When the operator has configured a GitHub App for the deployment, `/api/github/status` should report `"configured": true` and `/api/github/install` should redirect (302) to GitHub’s app installation URL.

**Reproduce:**

1. Log in to `https://bigbase.click/admin/`
2. Go to Deploy → Create site → Connect GitHub (or open `/api/github/install` while authenticated)
3. Observe 503 with the error above

**Prior art:** Related to [BUG-2026-06-02T000000](BUG-2026-06-02T000000-github-flags-not-wired.md) (CLI flags not registered — **fixed in code**). This report is a **recurrence on production**: flags exist but the running server never receives GitHub App credentials.

## Root Cause Analysis

**Phase 1 — Reproduce:** Confirmed on `bigbase.click` (user screenshot, 2026-06-02). Locally, `bigbase serve` without GitHub flags logs `github component ready configured=false` and returns the same 503 from `/api/github/install`.

**Phase 2 — Isolate:** Request path: UI → `GET /api/github/install` → auth middleware → github component `handleInstall` → `configured()` check. Failure occurs when `appID`, `appSlug`, and `privateKeyPath` are all empty on the github component instance.

**Phase 3 — Hypotheses (ranked):**

1. **(Confirmed)** Production systemd service starts `bigbase serve` with only `--port` and `--db`. GitHub App CLI flags are never passed. Deploy workflow writes Google OAuth vars to `/opt/bigbase/.env` but does not write GitHub App settings, upload a private key, or extend the service unit.
2. **(Ruled out)** CLI flags still unregistered in `serve` — flags are registered and wired to `github.New()` on current `main` (fixed in BUG-2026-06-02T000000).
3. **(Secondary)** Even if GitHub secrets were added to `.env`, the binary reads GitHub config only from CLI flags, not environment variables — same gap as Google OAuth on VPS.

**Phase 4 — Verify:** VPS systemd template (`setup-vps.sh`) `ExecStart` is `bigbase serve --port 8080 --db …/bigbase.db` with optional `EnvironmentFile` for Google-only keys. Release-deploy Phase 4 only writes `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`. No GitHub App ID, slug, private key path, or webhook secret are provisioned. **Root cause verified.**

**Risk level:** Medium — feature blocked on production; fix touches deploy pipeline and server config (secrets handling for PEM private key).

## TDD Fix Plan

### Cycle 1 — Env fallback for GitHub App config

**RED:** Add a test in the serve/config layer that when `GITHUB_APP_ID`, `GITHUB_APP_SLUG`, and `GITHUB_APP_PRIVATE_KEY_PATH` environment variables are set (and CLI flags omitted), the github component receives non-empty options and `/api/github/status` returns `"configured": true`.

**GREEN:** In `startProxy()`, after parsing CLI flags, fall back to `os.Getenv` for each GitHub (and optionally Google) setting when the flag value is empty.

**verify:** `go test ./... -run GitHub -count=1`

### Cycle 2 — Deploy pipeline provisions GitHub secrets on VPS

**RED:** Add a workflow/deploy script test or documented verify step: when GitHub secrets exist in the repo, deploy writes `/opt/bigbase/.env` entries (`GITHUB_APP_ID`, `GITHUB_APP_SLUG`, `GITHUB_WEBHOOK_SECRET`) and places the PEM at `/opt/bigbase/secrets/github-app.pem` (mode 600, owned by `bigbase` user).

**GREEN:** Extend `.github/workflows/release-deploy.yml` Phase 4 (Configure environment) to:

- Accept GitHub Actions secrets: `GITHUB_APP_ID`, `GITHUB_APP_SLUG`, `GITHUB_APP_PRIVATE_KEY`, `GITHUB_WEBHOOK_SECRET`
- SCP private key to VPS secrets path
- Append GitHub vars to `${VPS_ENV_FILE}` alongside Google OAuth

**verify:** `bash scripts/validate-specs-yaml.sh` (if spec updated) + manual `gh run view` after deploy with secrets set

### Cycle 3 — systemd unit reads env for GitHub (and document operator setup)

**RED:** Document in `setup-vps.sh` next-steps that GitHub App secrets are required for Sites GitHub connect; assert `ReadWritePaths` includes secrets directory if key is written at deploy time.

**GREEN:** Update `setup-vps.sh` systemd template: add `ReadWritePaths` for `${BIGBASE_HOME}/secrets`; list required GitHub secrets in the setup summary (mirror Google OAuth lines).

**verify:** `grep -q GITHUB_APP setup-vps.sh && grep -q GITHUB_APP .github/workflows/release-deploy.yml`

### Cycle 4 — Operator creates GitHub App (one-time, manual)

Not code — operator must create a GitHub App with:

- Callback URL: `https://bigbase.click/api/github/callback`
- Webhook URL: `https://bigbase.click/api/github/webhook`
- Permissions: repo contents, metadata, etc. per ADR 003

Store App ID, slug, webhook secret, and PEM in GitHub repo secrets; redeploy.

**verify:** `curl -s -H "Cookie: …" https://bigbase.click/api/github/status` → `"configured":true`; `/api/github/install` → 302 to `github.com/apps/…`

**REFACTOR:** Extract shared “resolve config from flag or env” helper for Google + GitHub in the composition root to avoid duplicated fallback logic.

## Acceptance Criteria

- [ ] Production `bigbase serve` receives GitHub App ID, slug, and private key path (via env or flags)
- [ ] `/api/github/status` returns `"configured": true` on bigbase.click after secrets + redeploy
- [ ] `/api/github/install` returns 302 redirect to GitHub (not 503) when configured
- [ ] Deploy workflow documents and supports GitHub App secrets
- [ ] Private key never committed to repo; stored on VPS with restrictive permissions
- [ ] All new tests pass; `go test ./... -count=1` green

## Resolution

**Fixed:** 2026-06-03

- Added `config.FlagOrEnv` and wired Google + GitHub settings in `startProxy()` from VPS `.env`.
- Extended `release-deploy.yml` to SCP PEM, merge `/opt/bigbase/.env`, and pass `BIGBASE_*` repo secrets.
- Updated `setup-vps.sh` with `secrets` ReadWritePaths and operator docs.

**Verify after deploy:** `/api/github/status` → `"configured": true` on bigbase.click.
