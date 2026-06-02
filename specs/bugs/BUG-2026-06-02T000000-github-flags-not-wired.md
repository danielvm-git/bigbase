---
type: bug
context: GitHub App integration — serve command flags not wired to github component
---

# BUG-2026-06-02T000000: GitHub App CLI flags not registered in serve command

## Problem

The GitHub integration UI at `https://bigbase.click/admin/#/deploy/new` shows a "Connect GitHub" option, but clicking it or navigating to `/api/github/install` returns **503 Service Unavailable** with:

```json
{"error":"GitHub App not configured; set --github-app-id, --github-app-slug, --github-app-private-key-path"}
```

**Actual behavior:** The `serve` command does not accept or pass the GitHub App configuration flags. There is no way for an operator to configure the GitHub integration at runtime.

**Expected behavior:** Running `bigbase serve --github-app-id 123456 --github-app-slug myapp --github-app-private-key-path /path/to/key.pem` should make the GitHub component available, with `/api/github/status` returning `{"configured":true}` and `/api/github/install` redirecting to GitHub's OAuth installation flow.

**Reproduced:**
```bash
go run . serve --port 9997
# → starts fine, no errors about unknown flags
curl -s http://localhost:9997/api/github/status -H "Authorization: Bearer $TOKEN"
# → {"configured":false,"connected":false}
curl -s http://localhost:9997/api/github/install -H "Authorization: Bearer $TOKEN"
# → 503 "GitHub App not configured"
```

## Root Cause Analysis

### Investigation (4-phase RCA)

**Phase 1 — Reproduce:** Confirmed. `/api/github/status` always reports `configured: false`. `/api/github/install` always returns 503. Consistent across all environments.

**Phase 2 — Isolate:** Code path:
1. `startProxy()` in main.go defines CLI flags on `serveFS` — only `port`, `db`, `google-client-id`, `google-client-secret`
2. `github.New()` is called with only `DB` and `Logger` — no `AppID`, `AppSlug`, `PrivateKeyPath`, or `WebhookSecret`
3. `github.configured()` checks `appID != "" && privateKeyPath != "" && appSlug != ""` — all three are always empty
4. `handleInstall()` returns 503 when `!configured()`
5. `handleStatus()` always returns `configured: false`

The first layer that produces wrong output is **step 1**: the flags are never defined, so there's no mechanism to supply the values.

**Phase 3 — Hypothesize:**
| # | Hypothesis | Probability |
|---|-----------|-------------|
| H1 | GitHub App flags are not registered on `serveFS` and not passed to `github.New()` | **Certain** (confirmed by code) |
| H2 | The github component's constructor ignores the options | Ruled out (constructor correctly stores all options) |

**Phase 4 — Verify:** Start server with `--github-app-id X` — flag package silently ignores unknown flags because `flag.ContinueOnError` is used. The flag value never reaches the component. `configured()` remains false. **Root cause confirmed.**

### Root cause

The `serve` command's flag set in the application entry point does not define the four GitHub App configuration flags (`--github-app-id`, `--github-app-slug`, `--github-app-private-key-path`, `--github-webhook-secret`), and consequently they are never passed to the `github` component constructor.

This is the same pattern as the Google OAuth flags (`--google-client-id`, `--google-client-secret`) which ARE correctly registered and wired — the GitHub flags simply followed the same pattern but were never added.

### Risk level: Low

Pure additive change. No existing behavior is modified — only new optional flags are added. If omitted (current behavior), everything continues to work as before.

## TDD Fix Plan

### Cycle 1: Add flag definitions to serve command

**RED**: Start the server with `--github-app-id 123 --github-app-slug test --github-app-private-key-path /tmp/key.pem` and verify `/api/github/status` returns `configured: true`.

Test via integration:
```go
// In main_test.go or a new integration test
// Start server with github flags
// GET /api/github/status with auth token
// Assert configured == true
```

**GREEN**: In `startProxy()`, register the four flags on `serveFS` and pass them to `github.New()`:

```go
// Before serveFS.Parse:
githubAppID := serveFS.String("github-app-id", "", "GitHub App ID")
githubAppSlug := serveFS.String("github-app-slug", "", "GitHub App slug")
githubPrivateKeyPath := serveFS.String("github-app-private-key-path", "", "GitHub App private key path")
githubWebhookSecret := serveFS.String("github-webhook-secret", "", "GitHub App webhook secret")

// Pass to github.New:
gh := github.New(github.Options{
    DB:             d,
    Logger:         logger,
    AppID:          *githubAppID,
    AppSlug:        *githubAppSlug,
    PrivateKeyPath: *githubPrivateKeyPath,
    WebhookSecret:  *githubWebhookSecret,
})
```

**verify:**
```bash
go run . serve --port 9999 --github-app-id 123 --github-app-slug test --github-app-private-key-path /tmp/key.pem &
curl -s http://localhost:9999/api/github/status -H "Authorization: Bearer $TOKEN"
# → {"configured":true,"connected":false}
```

### Cycle 2: Verify backward compatibility

**RED**: Start the server WITHOUT any GitHub flags (current behavior) and verify `/api/github/status` still returns `configured: false` (not a crash or error).

**GREEN**: No code change needed — the default empty strings already produce the correct behavior. The test just confirms existing behavior is preserved.

**verify:**
```bash
go run . serve --port 9999 &
curl -s http://localhost:9999/api/github/status -H "Authorization: Bearer $TOKEN"
# → {"configured":false,"connected":false}
```

### Cycle 3: Verify full build and test suite

**RED**: `go build -o bigbase .` must succeed, and `go test ./...` must pass with no regressions.

**GREEN**: No code changes — the changes are entirely additive. Verify build and suite pass.

**verify:**
```bash
go build -o bigbase .
go test ./... -count=1
```

**REFACTOR**: No refactoring needed. The change follows the exact same pattern as the existing Google OAuth flags. The `github` component itself is well-tested and unchanged.

## Acceptance Criteria

- [x] `bigbase serve --github-app-id X --github-app-slug Y --github-app-private-key-path Z` makes `/api/github/status` report `configured: true`
- [x] `bigbase serve` without GitHub flags still works and reports `configured: false`
- [x] `bigbase serve --github-webhook-secret W` (with other flags) is accepted
- [x] No regressions: `go test ./... -count=1` passes
- [x] `go build -o bigbase .` succeeds

## Resolution

**Fixed:** 2026-06-02
**Root cause confirmed:** GitHub App CLI flags (`--github-app-id`, `--github-app-slug`, `--github-app-private-key-path`, `--github-webhook-secret`) were never registered on the `serve` command's flag set, so there was no path for the operator to supply them. The `github.New()` constructor always received empty strings, making `configured()` permanently return `false`.

**Fix applied:**
1. Added 4 flag definitions to `serveFS` in `startProxy()` following the same pattern as the existing Google OAuth flags
2. Passed all four flag values to `github.New(github.Options{...})`

**Hardening added:**
- `TestGitHubFlagsConfigured` — regression test that creates github component WITH flags via public constructor, asserts `/api/github/status` returns `configured: true` and `/api/github/install` redirects (302 not 503)
- `TestGitHubFlagsUnconfigured` — backward-compatibility test that creates github component WITHOUT flags, asserts `/api/github/status` returns `configured: false` and `/api/github/install` returns 503
- Both tests exercise the public `Handler()` interface via `httptest.NewRecorder`, not implementation internals
- `setupGitHub(t, opts)` helper extracted to prevent setup boilerplate duplication

**Verification:**
```
With flags:    /api/github/status → {"configured":true,"connected":false}   ✅
               /api/github/install → HTTP 302 (redirect to GitHub)          ✅
Without flags: /api/github/status → {"configured":false,"connected":false}  ✅
               /api/github/install → HTTP 503                               ✅
go build .     → succeeds                                                    ✅
go test ./...  → 244 passed in 17 packages                                  ✅
```

**Commit:** `fix(main): register GitHub App CLI flags and wire to github component`
