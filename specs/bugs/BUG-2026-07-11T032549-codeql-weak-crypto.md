# Use of a broken or weak cryptographic hashing algorithm on sensitive data

**Source:** GHS Code Scanning (CodeQL)
**Severity:** MAJOR
**CWE:** CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
**GitHub Alert:** #16

## Description
CodeQL detected the use of a weak cryptographic hashing algorithm for sensitive data. This could allow an attacker to reverse or collide hashes.

## Recommendation
Replace weak hash algorithms (MD5, SHA-1) with SHA-256 or SHA-3 for hashing sensitive data. For password storage, use bcrypt (which is already used in the auth component).

## Status
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
