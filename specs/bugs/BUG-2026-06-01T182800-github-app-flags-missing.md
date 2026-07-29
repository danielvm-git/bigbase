# BUG-2026-06-01T182800: GitHub App configuration flags not registered or parsed in serve command

## Problem

When the user attempts to connect their GitHub App by navigating to `/api/github/install` (either directly or via the UI), the server returns a `503 Service Unavailable` error:
`{"error":"GitHub App not configured; set --github-app-id, --github-app-slug, --github-app-private-key-path"}` (or similar depending on build version)

Although the `github` component is instantiated and registered in the kernel, it is never passed the required GitHub App configuration (App ID, Slug, Private Key Path, Webhook Secret). 

## Root Cause Analysis

1. **Missing CLI Flags in `main.go`**:
   In `main.go`'s `startProxy()` function, only a few flags are registered and parsed for the `serve` command:
   ```go
   serveFS := flag.NewFlagSet("serve", flag.ContinueOnError)
   port := serveFS.String("port", "8080", "HTTP server port")
   dbPath := serveFS.String("db", "bigbase.db", "SQLite database path")
   googleClientID := serveFS.String("google-client-id", "", "Google OAuth client ID")
   googleClientSecret := serveFS.String("google-client-secret", "", "Google OAuth client secret")
   _ = serveFS.Parse(os.Args[2:])
   ```
   The flags specified in ADR 003 (`--github-app-id`, `--github-app-slug`, `--github-app-private-key-path`, `--github-webhook-secret`) are never registered or parsed.

2. **Incomplete Options Passed to `github.New`**:
   When the `github` component is created, only the database `d` and the `logger` are provided:
   ```go
   gh := github.New(github.Options{
       DB:     d,
       Logger: logger,
   })
   ```
   The configuration options `AppID`, `AppSlug`, `PrivateKeyPath`, and `WebhookSecret` remain unpopulated (`""`). Therefore, `g.configured()` always returns `false` at runtime.

## Risk Level

**Low** — This change adds missing command-line flags and passes them to the existing, already-implemented constructor. It has no risk of data loss or breakage to other components.

## TDD Fix Plan

### RED-GREEN Cycle 1: Wire and parse GitHub CLI flags in `main.go`

**RED**: 
Write an integration test in `main_test.go` or check if there is an existing test suite. We will verify that passing these flags configures the GitHub component correctly in the kernel.

**GREEN**:
1. In `main.go` (`startProxy`):
   Define the following flags on `serveFS`:
   ```go
   githubAppID := serveFS.String("github-app-id", "", "GitHub App ID")
   githubAppSlug := serveFS.String("github-app-slug", "", "GitHub App slug")
   githubPrivateKeyPath := serveFS.String("github-app-private-key-path", "", "GitHub App private key path")
   githubWebhookSecret := serveFS.String("github-webhook-secret", "", "GitHub App webhook secret")
   ```
2. Pass these variables to `github.New()`:
   ```go
   gh := github.New(github.Options{
       DB:             d,
       Logger:         logger,
       AppID:          *githubAppID,
       AppSlug:        *githubAppSlug,
       PrivateKeyPath: *githubPrivateKeyPath,
       WebhookSecret:  *githubWebhookSecret,
   })
   ```

## Acceptance Criteria

- [x] Command-line flags `--github-app-id`, `--github-app-slug`, `--github-app-private-key-path`, and `--github-webhook-secret` are successfully registered.
- [x] Running `bigbase serve` with these flags populates the configurations of the `github` component.
- [x] Navigating to `/api/github/install` redirect successfully (or doesn't throw the unconfigured error) when configured.
- [x] Running `go test ./...` passes.

## Status
fixed

## Resolution

**Fixed:** 2026-06-01

GitHub App flags are registered in main.go (lines 168-171) and passed to github.New() (lines 264-269). All acceptance criteria are met.
