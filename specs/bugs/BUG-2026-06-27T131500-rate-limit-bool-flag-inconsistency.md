---
bug_id: BUG-2026-06-27T131500
status: fixed
severity: low
scope: config,main
title: rate-limit-enabled bool flag uses manual os.Getenv instead of FlagOrEnv pattern
---

# BUG-2026-06-27T131500: rate-limit-enabled bool flag inconsistent with FlagOrEnv pattern

## Problem

**Actual**: `--rate-limit-enabled` (a boolean flag) used manual `os.Getenv("BIGBASE_RATE_LIMIT_ENABLED")`
with ad-hoc bool parsing (`e == "true" || e == "1"`), while string flags like `--rate-limit-ip-max`
and `--rate-limit-user-max` used the consistent `config.FlagOrEnv()` pattern.

**Expected**: All CLI flags with env var fallbacks use the same `config.FlagOrEnv*` helper family.

**How to reproduce**: grep `os.Getenv` in `main.go` startProxy() — the rate-limit-enabled block was
the only `BIGBASE_` env var accessed via raw os.Getenv instead of through the config helpers.

## Root Cause

`config.FlagOrEnv` only handles string types. Boolean flags with a non-zero default (`true`) cannot
distinguish "user passed the flag explicitly" from "flag defaulted", so `FlagOrEnv`'s precedence
semantics (flag wins when non-empty) would never check the env var. A dedicated `FlagOrEnvBool`
was needed with env-first semantics.

## Fix

1. Added `config.FlagOrEnvBool(flagVal bool, envKey string) bool` to `config/serve_env.go`:
   - When env var is set, parsed via `strconv.ParseBool` (handles "true"/"false"/"1"/"0")
   - When env var is absent, falls back to the CLI flag value
   - Garbage env values treated as false (logged by caller)
2. Replaced the manual block in `main.go`:
   ```go
   // Before
   rlEnabled := *rateLimitEnabled
   if e := os.Getenv("BIGBASE_RATE_LIMIT_ENABLED"); e != "" {
       rlEnabled = e == "true" || e == "1"
   }
   // After
   rlEnabled := config.FlagOrEnvBool(*rateLimitEnabled, "BIGBASE_RATE_LIMIT_ENABLED")
   ```
3. Added 8-case `TestFlagOrEnvBool` covering true/1/false/0/garbage/empty/missing-false/missing-true.

## Acceptance Criteria

- [x] `--rate-limit-enabled` env var resolves via `config.FlagOrEnvBool`
- [x] No raw `os.Getenv` for BIGBASE_ env vars remains in rate limit config
- [x] `TestFlagOrEnvBool` passes (8 cases)
- [x] All 26 Go packages pass, 310 UI tests pass
- [x] `go vet ./...` clean
- [x] `golangci-lint` clean on changed files
- [x] Convention comment in `main.go` guides future flag additions

## Resolution

**Fixed:** 2026-06-27
**Root cause confirmed:** `FlagOrEnv` only handles strings; boolean flags needed a dedicated `FlagOrEnvBool` helper with env-first precedence.
**Fix applied:** Added `config.FlagOrEnvBool` + 8-case test + replaced manual `os.Getenv` block in `main.go`.
**Hardening added:** `TestFlagOrEnvBool` (8 edge cases) + convention comment in `main.go`.
**Evidence:** `go test ./...` — 26/26 pass; `go vet ./...` clean; `golangci-lint` 0 issues on changed files.
**Commit:** `fix(config): add FlagOrEnvBool for consistent bool flag env resolution`
