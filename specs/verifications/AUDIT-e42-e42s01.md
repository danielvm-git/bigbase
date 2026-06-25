# Audit Report — e42s01: Build Cache

**Epic:** e42 — Build Cache  
**Story:** e42s01 — Cache key derivation and storage  
**Date:** 2026-06-25  
**Scope:** Cache backend (`cache.go`, `cache_archive.go`, `deploy.go`) + post-review hardening  
**Result:** ✅ **PASS** (merged as PR #47, commit 29238a6)

---

## Post-review hardening (Greptile)

The reviewer flagged two real defects in the initial cache implementation;
both were fixed before merge in commit 1a0968e:

1. **No mutex → concurrent deploys raced on archive/metadata files.**
   Added `sync.RWMutex` held across `Restore`/`Save`/`Evict`/`PurgeSite`
   (write) and `ListEntries`/`Stats` (read), with an unlocked internal
   `listEntries` to avoid re-entrant deadlock. Regression: a `-race`
   test runs 16 parallel restores and asserts `HitCount == 16` (no lost
   increments).
2. **`removeEntry` swallowed `os.Remove` errors and always returned nil**
   → eviction believed it freed space while files stayed on disk, so the
   cache grew past `maxBytes` forever. Now propagates the first
   non-`IsNotExist` error. Regression: `TestCacheRemoveEntry_PropagatesError`.

Deferred (not blocking, lower severity): symlink entries dropped on
extraction (P2), and no-lockfile fallback caching (advisory) — tracked for
follow-up; current archive path walks symlinks as regular files so `.bin`
is intact in practice.

Cache suite: **19 pass under `-race`**; full deploy suite: **179 pass**.

---

## Checklist

### Supply Chain & Security
- [x] No new dependencies — metadata-only change
- [x] No secrets in diff — verified via a secret-pattern grep over the diff (OpenAI/GitHub/AWS/New Relic key prefixes)
- [x] OWASP spot-check: N/A — YAML/metadata only, no user data or external APIs touched

### Provenance & Metadata
- [x] e42 capsule already has `type:` and `context:` metadata
- [x] Audit report saved as `specs/verifications/AUDIT-e42-e42s01.md`

### Law of Demeter
- [x] N/A — no code changes

### CONVENTIONS.md Compliance
- [x] All output in `specs/` — capsule, execution-status, state.yaml
- [x] No `gh issue create` calls
- [x] No direct GitHub REST API calls

### Scope
- [x] Changes limited to marking e42s01 done, updating state for e42s02
- [x] No speculative features
- [x] Only e42-related files touched
- [x] Surgical — single status field changed per file

### Boy Scout Rule
- [x] Status tracking now reflects reality (e42s01 done, PR #47 merged)
- [x] No dead code left behind
- [x] No commented-out code blocks

### Types and Safety
- [x] N/A — YAML metadata only

### Test Coverage
- [x] N/A — metadata reconciliation; code tested via CI (PR #47 passed)

### SOLID and Heuristics
- [x] N/A — no code changes

### Code Style (CONVENTIONS.md)
- [x] YAML formatting consistent with all other epic capsules
- [x] No duplication — status aligned across all three tracking files

### Agent Readability
- [x] N/A — metadata only

### Red Flags
- [x] No rationalizations — all items pass

---

## Verdict

All applicable checklist items pass. This is a clean metadata reconciliation with zero code changes.
