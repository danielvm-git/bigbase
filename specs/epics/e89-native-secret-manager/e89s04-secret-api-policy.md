# e89s04 — Secret REST API, Policies, and Audit Events

**type:** feat
**risk:** P0
**context:** domain
**BCPs:** 4

## 1. Type

Security-sensitive API feature.

## 2. Context

SecretManager needs an authenticated HTTP surface that distinguishes metadata
visibility from decrypted value access.

## 3. Summary

Expose Project/Environment/Folder secret CRUD, version reads, masked listings,
policy checks, rate limiting, and value-free audit events.

## 4. Problem

Current Site APIs conflate ownership and value access, and mutation responses can
return plaintext.

## 5. Users

Organization administrators, Project operators, and automation clients.

## 6. Solution

Use organization context from auth, canonical Project routes, and separate actions:
`describe_secret`, `read_secret_value`, `create_secret`, `update_secret`, and
`delete_secret`.

## 7. Alternatives

- Trust `org_id` in the URL: rejected because it permits tenant confusion.
- Return plaintext on create/update: rejected because clients frequently log responses.

## 8. Dependencies

e89s02, e89s03, existing Auth middleware, route policy, rate limiter, and audit logger.

## 9. Assumptions

The first policy implementation uses organization membership plus Project role;
folder/key-specific policies are represented for future extension.

## 10. Risks

List, read, mutation, and version routes can diverge in authorization. All routes
must use one handler policy helper.

## 11. Migration Plan

No legacy route removal. Existing Site routes remain separate until compatibility
migration is complete.

## 12. Data Model

Add audit records with actor, organization, Project, Secret reference, action,
request ID, and timestamp; never value or ciphertext.

## 13. API

```text
GET/POST /api/projects/{project}/environments/{env}/secrets
GET/PUT/DELETE /api/projects/{project}/environments/{env}/secrets/{key}
GET /api/projects/{project}/environments/{env}/secrets/{key}/versions
GET /api/projects/{project}/environments/{env}/secrets/{key}/value
```

Lists and mutations return metadata plus masked previews. Value reads require the
explicit read-value action and use the separate `/value` response type. Unauthenticated
requests return the standard JSON 401 error, unauthorized actions return 403, and
cross-organization targets use a non-disclosing 404. No response contains plaintext or
ciphertext except the explicit authorized value-read response.
### First-release policy and bounds

| Authenticated role | Describe/list | Read value | Create/update/delete | List versions |
|---|---:|---:|---:|---:|
| `org_admin` | yes | yes | yes | yes |
| `project_operator` | yes | yes | yes | yes |
| `project_member` | yes | no | no | yes |

The existing rate-limiter contract applies per authenticated actor and Project: at most
30 mutating requests per minute. Secret keys are limited to 128 bytes, values to 64 KiB,
and a single import/mutation batch to 1,000 keys. These bounds are security contracts,
not UI hints; over-limit requests are rejected before persistence.

## 14. Affected Code

New SecretManager handlers, `components/auth`, `components/api` or dedicated
secret route registration, audit events, rate limiting, contract tests, and the
coordinator-owned `main.go` route mount.

HTTP contract tests, cross-org denial, role/action matrix, no-plaintext response
assertions, rate limits, malformed input, SQL safety, and audit-value absence.

## 16. Rollback Plan

Disable native routes through configuration while retaining stored versions and
legacy Site endpoints.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s04-P0-01] List secrets returns metadata only
  Given a Project member can describe secrets but cannot read values
  When they list a Folder
  Then keys and masked previews are returned without plaintext values

Scenario: [SC-e89s04-P0-02] Read value requires explicit permission
  Given a caller can describe but not read Secret K
  When they request K's value
  Then the request is denied

Scenario: [SC-e89s04-P0-03] Cross-org access is denied
  Given Secret K belongs to organization A
  When organization B requests K by key or version
  Then the response does not reveal K's existence

Scenario: [SC-e89s04-P0-04] Audit event contains no sensitive value
  Given a Secret is created, read, and updated
  When audit records are queried
  Then they contain actor and scope metadata but no plaintext or ciphertext
Scenario: [SC-e89s04-P1-05] Secret mutations are bounded and rate-limited
  Given a caller submits an oversized payload or repeated mutations
  When the request is processed
  Then limits apply and malformed input cannot alter SQL structure

Scenario: [SC-e89s04-P1-06] Version listing is bounded and policy-independent
  Given a caller can describe but cannot read Secret values
  When they list versions
  Then versions are bounded and ordered without returning any value
```

## Requirements
+
#### ADDED: Policy-separated secret REST API
BigBase MUST expose Project secret CRUD and reads with separate Describe Secret and Read Secret Value actions, masked metadata responses, rate limits, organization isolation, and value-free audit events.

## 18. Implementation Steps

1. Add canonical routes and typed request/response schemas → verify: `go test ./components/secrets ./components/api -run 'Test.*Secret.*API' -count=1`
2. Add policy helper and organization/project isolation → verify: `go test ./components/secrets ./components/auth -run 'Test.*Secret.*Auth|Test.*Org' -count=1`
3. Add masked metadata responses, value reads, versions, and rate limits → verify: `go test ./tests/contract -run 'Test.*Secret' -count=1`
4. Add audit events and sensitive-field assertions → verify: `go test ./components/secrets ./components/monitoring -run 'Test.*Secret.*Audit' -count=1 && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Authenticate as an organization member with describe-only permission.
2. List a Folder and inspect the response for plaintext/ciphertext fields.
3. Attempt a value read and confirm denial.
4. Grant read-value permission and confirm the value read succeeds.
5. Repeat all calls from another organization.
6. Inspect audit rows for absence of values.

## 20. Out of scope

Secret sharing, approvals, external identities, imports, references, and generated SDKs.
## 21. Zoom-Out Check

- **Purpose:** REST handlers adapt authenticated HTTP requests to SecretManager actions; Auth supplies organization/role context; rate limiting and audit logging enforce shared API contracts.
- **Callers:** Admin UI, automation clients, contract tests, and the s05/s07 adapter wave; `main.go` mounts routes and injects dependencies.
- **Contracts:** one policy helper across list/read/mutate/version actions, explicit `/value` reads, JSON 401/403/non-disclosing 404 errors, masked responses, bounded mutations, parameterized SQL, and value-free audit events.
- **Reason for Depth:** one policy helper is required so list, value, mutation, and version routes cannot drift in authorization behavior.
