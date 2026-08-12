# Threat Model: e89 — Native Secret Manager

**Date:** 2026-08-11
**Epic:** e89 — Native Infisical-Inspired Secret Manager
**Risk Level:** CRITICAL before e89s01; HIGH during project-secret rollout

## 1. Attack surface

| Surface | Auth gate | Risk |
|---|---|---|
| Site env REST CRUD | Auth middleware + site ownership | HIGH |
| Project secret REST CRUD | Auth middleware + org/project policy | CRITICAL |
| Secret value read | Separate read-value policy | CRITICAL |
| Deployment resolver | Internal trusted seam + site/project association | HIGH |
| Build/runtime logs | Redaction pipeline | HIGH |
| MCP list/get/set/delete | Org key + narrow secret scopes + target binding | CRITICAL |
| Database/backups | Root-key separation and ciphertext-only rows | HIGH |
| Migration and key rotation | Resumable operator command | HIGH |

## 2. Trust boundaries

```text
[Browser / API client / MCP client]
          |
          v
[Proxy + Auth + org identity]
          |
          v
[SecretManager policy seam]
          |
          +--> [SQL store: ciphertext and metadata]
          |
          +--> [Deploy EnvResolver: in-memory plaintext only]
                         |
                         +--> [child build/runtime process]
```

Tenant identity MUST come from authenticated context. Request bodies, path
parameters, MCP `site_id`, and deployment payloads are not trusted identity.

## 3. Findings and mitigations

| CWE | Finding | Mitigation | Gate |
|---|---|---|---|
| CWE-311/CWE-312 | Missing key currently enables plaintext storage | Wire key; fail closed; explicit dev opt-in | e89s01 |
| CWE-639 | MCP target site is not checked against org identity | Resolve target ownership before every operation | e89s01, e89s07 |
| CWE-200/CWE-532 | Plaintext mutation responses and runtime logs | Metadata-only writes; centralized redaction | e89s01, e89s04, e89s06 |
| CWE-862 | Missing read-value policy | Separate describe/read actions | e89s04 |
| CWE-89 | Dynamic SQL risk in new scope queries | Parameterized SQL only; SQL-safety review | every story |
| CWE-384 | Confused machine-token scope | Bind token to project/environment/folder actions | e89s07 |
| CWE-327 | Weak or ambiguous encryption configuration | AES-256-GCM, versioned blobs, validated key size | e89s01, e89s03 |
| CWE-459 | Interrupted migration can leave mixed data | Dual-read, checkpoints, counts, resumable migration | e89s06 |
| CWE-400 | Unbounded values/list operations | Request size limits, pagination, bounded imports | e89s01, e89s04, e89s05 |

## 4. Mandatory invariants

- No secret value in logs, audit events, list responses, or write responses.
- No cross-organization secret read or mutation.
- Site deploy keys cannot access unrelated project secrets.
- Decryption failure fails protected deployment paths.
- Secret updates are immutable versions.
- Ciphertext is bound to its scope through authenticated metadata.
- Root key material never enters the database in plaintext.

## 5. Security verification

Every task marked `security: high` MUST include:

```text
no new security findings in affected paths
```

Required commands before release:

```bash
go test -race ./components/sites/... ./components/deploy/... ./components/mcp/...
go test ./...
go vet ./...
golangci-lint run ./...
```

The dedicated security review must re-check tenant isolation, log redaction,
crypto configuration, migration behavior, and unsafe deserialization before release.
