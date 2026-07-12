# CodeQL Path Traversal #19 — manifest.go

> **Source:** Seal GitHub Code Scanning (2026-07-11)
> **Severity:** MAJOR
> **Application:** BigBase BaaS Platform (cbb33318)
> **CWE:** CWE-22

## Finding

CodeQL alert #19 flags `components/deploy/manifest.go` for uncontrolled data used in path expression.

## Assessment

**False positive.** The `LoadManifestPath` function in `manifest.go` already has a path containment check:

1. Rejects absolute paths
2. `filepath.Abs` + `strings.HasPrefix` containment verification
3. Resolves and verifies the result stays within the project directory

This was previously investigated and documented in `BUG-2026-07-11T032549-codeql-path-traversal.md` as alert #19 — false positive.

## Decision

**wontfix** — containment guards already exist. No action required.

## Related

- `BUG-2026-07-11T032549-codeql-path-traversal.md`
- `BUG-2026-07-11T032549-codeql-zip-slip.md`

## Seal Reference

- Vulnerability ID: `57e07540-925b-4eab-a5ab-e61aed20b825`
- Code Scanning Alert: https://github.com/danielvm-git/bigbase/security/code-scanning/19
