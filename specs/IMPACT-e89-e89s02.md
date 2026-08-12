type: impact-assessment
context: domain
epic: e89
story: e89s02
mode: lightweight

# Impact Assessment: e89s02

## Target

Add organization-owned Projects and Environments, attach Sites, and preserve existing
Site behavior through idempotent compatibility migration.

## Module purpose, callers, contracts

- Purpose: the proposed `components/projects` component owns Project/Environment schema,
  organization-scoped CRUD, and dependency ordering; `components/sites` remains the
  owner of Site records and attachment lifecycle.
- Callers: Site creation/migration, authenticated API handlers, Admin UI data clients,
  SecretManager startup, Deploy lookup, and future REST/MCP adapters.
- Contracts: org identity comes from authenticated context; SQLite/PostgreSQL behavior
  is equivalent; existing Sites receive exactly one compatibility Project and production
  Environment; deletion is blocked while Sites remain attached; migrations are idempotent.

## Impact and risks

High. This changes shared Site schema/startup and introduces a new component dependency.
The `sites -> projects -> db` startup edges must be explicit because component
registration order is not a schema-order guarantee. Main composition wiring is
coordinator-owned to avoid conflicts with s04/s06/s07.

## Coverage

Scenarios: SC-e89s02-P1-01 through SC-e89s02-P1-05. Existing callers and migration
paths are covered by the e89s02 task ledger and `components/projects` package tests.

## Recommended action

Implement only after s01 is accepted. Keep secret values untouched. Run SQLite,
PostgreSQL-compatible migration tests, cross-org contract tests, and plan consistency
before kickoff.
