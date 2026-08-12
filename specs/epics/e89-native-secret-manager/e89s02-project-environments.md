# e89s02 — Organization-Scoped Projects and Environments

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 4

## 1. Type

Domain feature.

## 2. Context

Infisical scopes secrets by Project and Environment. BigBase currently has Sites
but no active Project entity, so a native manager needs an explicit grouping model.

## 3. Summary

Add organization-owned Projects and named Environments, attach Sites to Projects,
and provide authenticated CRUD while preserving existing Site behavior.

## 4. Problem

A flat Site-only scope forces duplicate values and cannot represent development,
staging, and production contexts.

## 5. Users

Organization administrators and developers managing multiple Sites.

## 6. Solution

Add Project and Environment records. New Sites receive a Project and a default
production Environment. Existing Sites receive one compatibility Project each
until operators regroup them.

## 7. Alternatives

- Add only `project_id` to `site_env_vars`: rejected because it lacks environments.
- Infer environments from Site names: rejected because names are not stable identity.

## 8. Dependencies

e23 organization isolation, e18 dual-driver database support, e89s01 key/auth hardening.

## 9. Assumptions

A Site belongs to at most one Project in the first release. Project deletion is
blocked while Sites remain attached.

## 10. Risks

Existing installations have no Project rows. The compatibility migration must be
idempotent and preserve Site ownership.

## 11. Migration Plan

Create one Project and production Environment per existing Site. Attach the Site.
Do not move or rewrite Site secret values in this story.

## 12. Data Model

`projects(id, org_id, name, created_at, updated_at)`;
`project_environments(id, project_id, slug, name, created_at, updated_at)`;
`sites.project_id` with foreign key and index.

## 13. API

Authenticated Project CRUD and Environment listing/creation. Organization comes
from auth context and is checked against the target Project.

## 14. Affected Code

`components/projects`, `components/sites`, `components/auth`, `components/api`, `kernel` scope helpers,
database migrations, Admin UI project/site data types, tests, and `main.go` coordinator-owned
composition wiring.

## 15. Testing Strategy

Test SQLite/PostgreSQL schema creation, idempotent migration, org isolation, Site
attachment, default Environment creation, and deletion constraints.

## 16. Rollback Plan

Disable Project routes and preserve nullable `sites.project_id` rows. Do not delete
created Projects during rollback.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s02-P1-01] Create a Project inside an organization
  Given an authenticated administrator in organization A
  When they create Project P
  Then P belongs to organization A and is visible to A only

Scenario: [SC-e89s02-P1-02] Cross-organization Project access is denied
  Given Project P belongs to organization A
  When a user from organization B requests P
  Then the response is a non-disclosing not-found or forbidden response

Scenario: [SC-e89s02-P1-03] New Site receives a default Project and Environment
  Given an authenticated user creates a Site
  When creation completes
  Then the Site is attached to a Project and production Environment

Scenario: [SC-e89s02-P1-04] Existing Sites migrate idempotently
  Given an existing database with Sites and no Project rows
  When the migration runs twice
  Then each Site has exactly one compatibility Project and production Environment
Scenario: [SC-e89s02-P1-05] Project deletion respects Site attachments
  Given a Project still has an attached Site
  When an operator deletes the Project
  Then deletion is rejected and the Site attachment remains intact
```

## Requirements
+
#### ADDED: Organization-owned Projects and Environments
BigBase MUST provide organization-scoped Projects and named Environments, attach Sites to Projects, and create a production Environment for migrated Sites without moving secret values.

## 18. Implementation Steps

1. Add Project and Environment schema with dual-driver migration → verify: `go test ./components/db ./components/projects ./components/sites -run 'Test.*Project|Test.*Environment|Test.*Migration' -count=1`
2. Add organization-scoped Project and Environment handlers → verify: `go test ./components/api ./components/projects ./components/sites -run 'Test.*Project|Test.*Environment|Test.*Org' -count=1`
3. Attach new and existing Sites without changing secret values → verify: `go test ./components/sites -run 'Test.*Site.*Project|Test.*Compatibility' -count=1`
4. Add API contract and browser coverage for Project/Environment navigation → verify: `go test ./tests/contract ./components/projects -run 'Test.*Project|Test.*Environment' -count=1 && npx playwright test tests/e2e/project-environments-ui.spec.ts --config tests/e2e/playwright.config.ts && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Create two organizations.
2. Create a Project and Environment in organization A.
3. Confirm organization B cannot read or mutate them.
4. Create a Site and verify its default Project/Environment.
5. Run migration twice against an existing database.

## 20. Out of scope

Secret values, folders, SecretVersions, imports, references, and Project-level
secret UI.
## 21. Zoom-Out Check

- **Purpose:** `components/projects` owns Project/Environment schema and organization-scoped CRUD; Sites owns Site records and Project attachment; Auth/API provide request identity and transport.
- **Callers:** Site creation/migration, authenticated API handlers, Admin UI data clients, SecretManager startup, Deploy lookup, and future REST/MCP adapters.
- **Contracts:** organization identity comes from authenticated context, `sites -> projects -> db` startup dependencies are explicit, SQLite/PostgreSQL migrations are equivalent and idempotent, each Site has one compatible Project/production Environment, and attached Projects cannot be deleted.
- **Reason for Depth:** a dedicated Project component is required to guarantee schema startup ordering and keep Project ownership separate from Site storage.
