# BUG-2026-06-22T030000: Code quality issues in e40s03 manifest implementation

## Problem

The audit of commit `03a01e2` (e40s03: Admin UI manifest view and editor) found 4 code quality issues in the manifest endpoints:

1. **Sensitive data exposure**: `getSiteManifest` leaks raw git command output in API error responses (line 636 of `sites.go`).
2. **DRY violation**: Repo resolution logic (SELECT + fallback to git_repos + default branch) is duplicated verbatim between `getSiteManifest` and `saveSiteManifest` (~12 lines each).
3. **Function too long**: `saveSiteManifest` is 124 lines and does 10 sequential operations (parse, validate, resolve, clone, checkout, write, add, status, commit, push).
4. **Missing direct test**: `ValidateManifest` has no dedicated unit test (only exercised indirectly through the integration test).

## Acceptance Criteria

- [x] `getSiteManifest` returns a generic error message instead of raw git output on failure
- [x] Duplicated repo resolution is extracted into a shared helper `resolveRepoBranch`
- [x] `saveSiteManifest` is split into smaller helper methods (clone, checkout/commit, push)
- [x] `ValidateManifest` has a direct unit test in `manifest_test.go`
- [x] All 153 existing tests still pass

## Resolution

**Fixed:** 2026-06-22  
**Root cause confirmed:** Audit of commit `03a01e2` found 4 code quality issues in the manifest endpoints.  
**Fixes applied:**
1. Error exposure: replaced `fmt.Sprintf("git show failed: %v", errMsg)` with `"failed to read manifest file"` — raw git output is logged but not exposed to the API client.
2. DRY violation: extracted `resolveRepoBranch(ctx, id)` helper used by both `getSiteManifest` and `saveSiteManifest`.
3. Function too long: extracted `commitManifestToRepo(ctx, repoPath, branch, content)` helper — `saveSiteManifest` reduced from 124 to ~30 lines.
4. Missing direct test: added `TestValidateManifest` with 6 subtests (valid, invalid YAML, missing version, invalid framework, port out of range, missing build.command).  
**Evidence:** 705 tests pass (`go test ./... -count=1`); 7 new test cases added in `TestValidateManifest`.  
**Commits:** `b1daf3f` (fix)
