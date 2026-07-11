# Uncontrolled data used in path expression (5 instances)

**Source:** GHS Code Scanning (CodeQL)
**Severity:** MAJOR
**CWE:** CWE-22 (Path Traversal)
**GitHub Alerts:** #17, #18, #19, #20, #21

## Description
CodeQL detected 5 instances where user-controlled data is used in file path operations without proper validation. An attacker could read or write files outside the intended directory.

## Recommendation
Use `filepath.Clean` and `filepath.Abs` to resolve paths and verify they stay within the intended base directory using `strings.HasPrefix`. Validate all path components against an allowlist.

## Status
fixed

## Source
seal.github_code_scanning

## Discovered
2026-07-11

## Fixes

### Alert #17 (True Positive) — components/deploy/engine.go:138
- `deploy.RepoID` is user-controlled, used unsanitized in `filepath.Join` for git repo path
- Fix: added `filepath.Clean(deploy.RepoID)` + `strings.Contains(repoID, "..")` containment check before path construction

### Alert #18 (True Positive) — components/deploy/gateway.go:189
- `id` from URL path param used unsanitized in `os.RemoveAll(filepath.Join(d.buildsDir, id))`
- Fix: added `filepath.Clean(id)` + `strings.Contains(cleanID, "..") || strings.Contains(cleanID, "/")` guard at top of `handleDeleteDeployment`

### Alert #20 (True Positive) — components/git/git.go:275
- `id` from URL path param used unsanitized in `os.RemoveAll(filepath.Join(g.dir, id+".git"))`
- Fix: added `filepath.Clean(id)` + `strings.Contains(cleanID, "..") || strings.Contains(cleanID, "/")` guard before filesystem operation

### Alert #19 — components/deploy/manifest.go (False Positive)
- Already has guard on lines 127-128 via `LoadManifestPath` path containment check
- Status: wontfix

### Alert #21 — components/storage/storage.go (False Positive)
- Already has traversal check on line 390
- Status: wontfix

### Alert #22 — components/deploy/cache_archive.go (False Positive)
- Zip slip already has `extractEntry` containment on lines 121-124
- Status: wontfix
