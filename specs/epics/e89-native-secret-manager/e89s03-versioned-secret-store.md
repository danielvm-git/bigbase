# e89s03 — Versioned Envelope-Encrypted Project Secrets

**type:** feat
**risk:** P0
**context:** infra
**BCPs:** 5

## 1. Type

Security-sensitive domain storage.

## 2. Context

Infisical encrypts secret values with project data keys and preserves immutable
secret versions. BigBase currently has one optional ciphertext field per Site key.

## 3. Summary

Create the native Secret, SecretVersion, and SecretFolder storage path using the
KeyHierarchy defined by ADR 0009.

## 4. Problem

Overwriting one ciphertext field prevents audit history, rollback, key rotation,
and future scope-bound references.

## 5. Users

SecretManager, REST handlers, Deploy resolver, and future adapters.

## 6. Solution

Store metadata separately from immutable encrypted versions. Generate or load a
Project Data Key under transactional serialization and bind ciphertext to scope.

## 7. Alternatives

- Reuse one global AES key: rejected because compromise affects every tenant.
- Store version history as JSON: rejected because it weakens constraints and queryability.

## 8. Dependencies

ADR 0008, ADR 0009, e89s02 Projects/Environments, `envcrypto` primitives.

## 9. Assumptions

Core release uses internal AES-GCM only. External KMS is a future Adapter.

## 10. Risks

Key loss makes ciphertext unrecoverable. Root-key backup and rotation verification
are release gates.

## 11. Migration Plan

Native tables are additive. Existing Site values remain in the Compatibility Layer
until e89s06.

## 12. Data Model

`secret_folders`, `secrets`, `secret_versions`, `project_key_records` with foreign
keys, unique active keys per folder, version numbers, key IDs, algorithm IDs, and
scope metadata.

## 13. API

Expose a typed SecretManager interface for metadata listing, value reads, writes,
version listing, and deletion. HTTP routes are delivered by e89s04.

## 14. Affected Code

New `components/secrets` modules, `components/internal/envcrypto`, database
migration layer, backup/restore, and tests.

## 15. Testing Strategy

Round-trip, wrong-key, nonce uniqueness, AAD scope binding, version immutability,
concurrent first-key creation, rotation checkpointing, backup/restore, and dual DB.

## 16. Rollback Plan

Disable native secret writes while retaining tables and ciphertext. Keep legacy
Site reads available. Never drop native tables during application rollback.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s03-P0-01] Secret value is encrypted with a Project Data Key
  Given a Project with a valid root key
  When a Secret is stored
  Then the database contains ciphertext and no plaintext value

Scenario: [SC-e89s03-P0-02] Secret update creates an immutable version
  Given Secret K has version 1
  When K is updated
  Then version 1 remains readable and version 2 becomes current

Scenario: [SC-e89s03-P0-03] Ciphertext is bound to scope
  Given ciphertext created for folder A
  When it is copied to folder B
  Then authenticated decryption fails

Scenario: [SC-e89s03-P0-04] Concurrent first writes are safe
  Given two requests create the first secret in a Project concurrently
  When both complete
  Then exactly one Project Data Key exists and both versions decrypt
Scenario: [SC-e89s03-P0-05] Invalid key material fails closed
  Given a SecretVersion has an invalid root key, key ID, algorithm, nonce, or ciphertext
  When decryption is attempted
  Then decryption fails without exposing plaintext

Scenario: [SC-e89s03-P1-06] Backup and interrupted rotation are recoverable
  Given encrypted versions and key records are backed up during rotation
  When restore or resume runs from a checkpoint
  Then versions remain decryptable and the old key is not retired early
```

## Requirements
+
#### ADDED: Versioned envelope-encrypted project secrets
BigBase MUST store Project secrets as metadata plus immutable encrypted SecretVersions using a scope-bound Project Data Key and validated root-key hierarchy.

## 18. Implementation Steps

1. Add SecretFolder/Secret/SecretVersion schema and constraints → verify: `go test ./components/db ./components/secrets -run 'Test.*Schema|Test.*Constraint' -count=1`
2. Implement KeyHierarchy and versioned AES-GCM envelope → verify: `go test ./components/secrets ./components/internal/envcrypto -run 'Test.*Key|Test.*Cipher|Test.*Version' -count=1`
3. Implement SecretManager metadata/value operations through the public seam → verify: `go test ./components/secrets -run 'TestSecretManager' -count=1`
4. Add concurrency, wrong-scope, backup/restore, transaction-seam, and race coverage → verify: `go test -race ./components/secrets ./components/backup ./components/db ./kernel -count=1 && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Start a test database with a configured root key.
2. Create a Project, Folder, and Secret.
3. Inspect the database and confirm plaintext is absent.
4. Update the Secret and read both versions.
5. Attempt wrong-folder decryption.
6. Run concurrent first writes and key-rotation simulation.

## 20. Out of scope

HTTP policy routes, Admin UI, MCP, dynamic secrets, external KMS, and legacy row
migration.
## 21. Zoom-Out Check

- **Purpose:** `components/secrets` owns SecretFolder/Secret/SecretVersion metadata, KeyHierarchy, encryption, masking, and the SecretManager public seam; the optional transaction seam owns atomic first-key creation.
- **Callers:** REST policy handlers, Deploy resolver, MCP tools, backup/restore, audit integration, and future adapters.
- **Contracts:** canonical base64 32-byte root key, AES-256-GCM with fresh nonces and scope-bound AAD, ciphertext-only persistence, immutable versions, transactional concurrency safety, SQLite/PostgreSQL parity, and separate metadata/value result types.
- **Reason for Depth:** one SecretManager seam prevents REST, Deploy, and MCP from duplicating crypto/policy logic; an optional transaction seam is required for atomic first-key creation on both drivers.
