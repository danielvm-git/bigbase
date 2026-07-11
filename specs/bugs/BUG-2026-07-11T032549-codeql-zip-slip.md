# Arbitrary file access during archive extraction ("Zip Slip")

**Source:** GHS Code Scanning (CodeQL)
**Severity:** MAJOR
**CWE:** CWE-22 (Path Traversal)
**GitHub Alert:** #22

## Description
Archive extraction does not validate file paths inside the archive. A malicious zip file with entries containing `../` path components could overwrite files outside the extraction target directory.

## Recommendation
Validate each archive entry's resolved path stays within the extraction directory. Reject entries with `../` or absolute paths.

## Status
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
