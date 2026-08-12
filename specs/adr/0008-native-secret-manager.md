# ADR 008 — Native Infisical-Inspired Secret Manager

type: adr
context: BigBase secret management, project scoping, and deployment delivery

## Status

Proposed — implementation is gated on plan audit and threat-model acceptance.

## Decision

Implement a native `SecretManager` module inside BigBase. Do not embed or
wholesale-port Infisical. Reuse Infisical's domain ideas — project/environment/
folder scope, immutable versions, permission-separated reads, references as a
future seam, and envelope encryption — but implement them with BigBase's Go,
ECC, SQLite/PostgreSQL, and composition-root conventions.

## Domain model

```text
Organization -> Project -> Environment -> SecretFolder -> Secret -> SecretVersion
Site -> Project
```

`site_env_vars` remains a compatibility source during migration. Native project
secrets are resolved before site compatibility values; site values win on key
collision to preserve existing behavior.

## Module seams

- `SecretManager` owns metadata, versions, policy checks, masking, audit events,
  and access to encrypted values.
- `KeyHierarchy` owns root-key bootstrap, project data keys, ciphertext versions,
  and future rotation.
- `EnvResolver` remains the only deployment environment-resolution seam.
- HTTP, Admin UI, MCP, and Deploy use adapters at the composition root rather
  than importing one another directly.

## Rationale

A flat `project_secrets` table would reproduce the current shallow design and
cannot represent environment, folder, version, or read-policy semantics. A
wholesale Infisical port would introduce Node/PostgreSQL/Redis assumptions and
enterprise-code provenance risk. Native modules preserve locality and allow
incremental compatibility migration.

## Consequences

Positive:

- One source of truth for secret semantics and redaction.
- Existing Sites and Deploy callers can migrate incrementally.
- SQLite and PostgreSQL remain supported.
- Advanced Infisical capabilities can attach through future seams.

Negative:

- Project and Environment are new BigBase domain entities.
- Existing site values require dual-read migration.
- External KMS and provider integrations are deferred.

## Alternatives rejected

1. Continue extending `site_env_vars`: preserves flat scope and no version history.
2. Run Infisical as a sidecar: adds operational and authentication complexity.
3. Copy Infisical source: mismatched architecture and enterprise-code risk.
