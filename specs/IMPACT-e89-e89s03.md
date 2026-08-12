type: impact-assessment
context: infra
epic: e89
story: e89s03
mode: lightweight

# Impact Assessment: e89s03

## Target

Add SecretFolder, Secret, SecretVersion, Project Data Key, and envelope-encrypted
SecretManager storage behind a stable public seam.

## Module purpose, callers, contracts

- Purpose: `components/secrets` owns metadata, immutable versions, KeyHierarchy,
  ciphertext, masking, and SecretManager operations. `kernel`/DB transaction support
  provides atomic first-key creation without widening unrelated component interfaces.
- Callers: e89s04 REST policy, e89s06 Deploy resolver, e89s07 MCP, backup/restore,
  audit integration, and future adapters.
- Contracts: AES-256-GCM with fresh nonces; canonical base64 32-byte root key;
  Project Data Keys encrypted under the root hierarchy; scope-bound AAD; no plaintext
  or root key in rows; immutable versions; SQLite/PostgreSQL parity; typed metadata
  versus value-read responses.

## Impact and risks

Critical. This is new security-sensitive storage with concurrency, transaction, backup,
and key-loss blast radius. The optional `kernel.TxBeginner` seam and both DB drivers
must be verified before opening s04/s06 parallel work.

## Coverage

Scenarios: SC-e89s03-P0-01 through SC-e89s03-P0-05 and SC-e89s03-P1-06. Tests use
isolated in-memory SQLite, race-enabled concurrent writes, backup temp files, and an
explicit PostgreSQL DSN gate.

## Recommended action

Implement after s02. Freeze the SecretManager interface and encryption metadata before
REST or Deploy branches start. Do not modify s01 envcrypto semantics without a
coordinator-reviewed handoff.
