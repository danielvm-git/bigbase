# Audit Report — e42s02: Admin UI Cache Management

**Epic:** e42 — Build Cache
**Story:** e42s02 — Admin UI cache management
**Date:** 2026-06-25
**Branch:** feat/e42s02-cache-ui
**Scope:** Cache-management HTTP endpoints (components/deploy) + per-site Cache tab + global Build Cache panel
**Result:** ✅ **PASS**

---

## Checklist

### Supply Chain & Security
- [x] No new dependencies — uses stdlib (net/http, encoding/json) and existing UI libs
- [x] No secrets in diff — ran a secret-pattern grep over source-only diff (credential/key prefixes); no matches
- [x] Endpoints behind existing auth middleware — smoke confirmed 401 without token, 200 with admin token
- [x] Request bodies capped via `http.MaxBytesReader` (1 MiB)
- [x] OWASP: path param `siteID` only used for in-memory filtering / file glob within the cache dir; no SQL injection (config upsert is parameterized)

### Provenance & Metadata
- [x] e42 capsule has `type:`/`context:` metadata; tasks expanded with design_decisions
- [x] Audit report saved as specs/verifications/AUDIT-e42-e42s02.md

### Law of Demeter
- [x] All cache endpoints live in components/deploy (owns the cache); sites/admin stay decoupled
- [x] UI talks to the data layer (sitesData.ts), not fetch directly inside components

### CONVENTIONS.md Compliance
- [x] snake_case Go filenames (cache_api.go, cache_api_test.go)
- [x] PascalCase exports, camelCase unexported; acronyms (HTTP/JSON/ID) all-caps
- [x] Generic client errors ("internal error"), full detail via d.logger.Error — no stack/raw errors leaked
- [x] No `gh issue create` / direct GitHub REST calls
- [x] No `any` type misuse — only `map[string]any` for JSON responses, the established idiom (envvars.go, api.go)

### Scope
- [x] Changes limited to cache management; no unrelated refactors
- [x] No speculative features beyond the three story tasks

### Boy Scout Rule
- [x] New components mirror existing patterns (SiteEnvVarsTab, envvars.go handlers)
- [x] No dead/commented-out code; rebuilt dist committed (embedded via ui/embed.go)

### Types and Safety
- [x] Concrete types throughout; CacheEntry/CacheStats/SiteCacheStatus typed in Go and TS
- [x] prune `max_age_days` uses `*int` to distinguish absent (default 7) from explicit 0 (400)
- [x] SetMaxBytes guards non-positive values; config validated server-side (> 0)

### Test Coverage
- [x] Go: 11 new tests — 4 Cache methods (incl. -race) + 9 handler tests (stats/clear/prune/config-persist/site/validation)
- [x] UI: 19 new tests — fmtBytes (1), SiteCacheTab (5), BuildCachePanel (7) + DeployPage mock update
- [x] UAT smoke against running binary caught a real bug (prune 0 → 7 default); fixed + regression-tested

### Defensive Code (CONVENTIONS.md categories)
- [x] Timeout — all DB ops wrapped in context.WithTimeout(5s)
- [x] Graceful degradation — cache config absent → falls back to CLI/default; preview-mode UI returns mocks
- [x] Concurrency — Cache mutex-guarded (inherited from e42s01 hardening); new methods hold the lock

### SOLID and Heuristics
- [x] Single responsibility: cache_api.go is handlers only; cache.go is storage logic; config persistence isolated
- [x] Cache stays storage-agnostic (the DB persistence lives in deploy.go/cache_api.go, not Cache)

### Red Flags
- [x] No rationalizations — the one deferred item (lint react-hooks/set-state-in-effect) is pre-existing
      codebase debt (35 errors on main, same rule in SiteEnvVarsTab/DeployPage/UsersPage); new code
      follows the established pattern rather than diverging. CI does not gate on lint.

---

## F.I.R.S.T (enforce-first --quick)

- **Fast** — t.TempDir + in-memory SQLite (Go); mocked data layer (UI). No real network in unit tests.
- **Independent** — each test owns its temp cache dir / DB; concurrent-restore test is self-contained.
- **Repeatable** — Prune test backdates via relative offset, not wall-clock literals.
- **Self-validating** — explicit assertions on status codes, JSON shape, and side effects.
- **Timely** — written red-first before each implementation slice.

No F.I.R.S.T violations.

---

## Verdict

All checklist items pass. Go suite 803 pass (26 packages), UI 295 pass, both builds clean, deploy lint 0 issues.
The two failing integration tests under full-suite load (TestRedeployReplacesPrevious, port-close timing) are
pre-existing real-port flakes that pass in isolation and on re-run — unrelated to cache changes.
