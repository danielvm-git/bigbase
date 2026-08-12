# ADR 009 — Secret Envelope Encryption and Key Lifecycle

type: adr
context: BigBase Secret Manager cryptographic storage

## Status

Proposed — required before native secret storage implementation.

## Decision

Use envelope encryption:

```text
BIGBASE_ROOT_ENCRYPTION_KEY
  -> encrypted root key record
    -> project data key
      -> SecretVersion ciphertext
```

Secret values use AES-256-GCM with a fresh random nonce. Ciphertext includes an
explicit version and key identifier. Secret versions are immutable.

Canonical startup configuration is `BIGBASE_ROOT_ENCRYPTION_KEY`, base64-encoded
32-byte key. The existing `BIGBASE_ENV_ENCRYPTION_KEY` hex key is a legacy
migration input only.

Production startup fails when the canonical key is missing or invalid. Plaintext
mode requires an explicit development-only opt-in and is never the default.

## Additional authenticated data

The implementation SHOULD authenticate stable scope metadata as AAD:

```text
project_id | environment_id | folder_id | secret_id | version
```

A ciphertext copied to another scope must fail authentication.

## Rotation

Key records carry key IDs. Rotation is resumable:

1. Create the new key record.
2. Read ciphertext with the old key.
3. Re-encrypt under the new project data key.
4. Update the version/key metadata transactionally.
5. Verify counts and decryptability.
6. Retire the old key only after verification and backup confirmation.

## Consequences

- The root key must be backed up separately from the database.
- Existing plaintext/no-op rows need an explicit migration mode; BigBase must not
  guess whether a legacy value is plaintext or ciphertext.
- External KMS and HSM adapters are deferred and must implement a future key seam.
- The implementation must never log key material or plaintext values.
