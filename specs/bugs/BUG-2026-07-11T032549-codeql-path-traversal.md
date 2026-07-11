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
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
