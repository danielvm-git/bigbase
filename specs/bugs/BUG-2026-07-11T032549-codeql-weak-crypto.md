# Use of a broken or weak cryptographic hashing algorithm on sensitive data

**Source:** GHS Code Scanning (CodeQL)
**Severity:** MAJOR
**CWE:** CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
**GitHub Alert:** #16

## Description
CodeQL detected the use of a weak cryptographic hashing algorithm for sensitive data. This could allow an attacker to reverse or collide hashes.

## Recommendation
Replace weak hash algorithms (MD5, SHA-1) with SHA-256 or SHA-3 for hashing sensitive data. For password storage, use bcrypt (which is already used in the auth component).

## Resolution

### Approach
Replaced raw SHA-256 (`sha256.Sum256`) with HMAC-SHA256 using the auth component's server secret key. This binds each API key hash to the server secret, preventing rainbow table attacks on stored hashes. API keys are already high-entropy random tokens, so the primary threat is offline hash cracking if the database is compromised — HMAC-SHA256 eliminates that risk without the overhead of a KDF (which is unnecessary for random keys).

### Changes
- **`components/auth/apikeys.go`**: Added `crypto/hmac` import. Converted `hashAPIKey` from a standalone function to a method on `*Auth` that uses `hmac.New(sha256.New, a.secret)`. Updated all 5 call sites to `a.hashAPIKey(...)`.

## Status
fixed

## Source
seal.github_code_scanning

## Discovered
2026-07-11
