# Test Design: e89-native-secret-manager

**type:** test-plan
**context:** domain
**epic:** e89
**risk profile:** P0/P1; CRITICAL security boundary
**source:** `specs/security/epics/e89/THREAT_MODEL.md`, ADR 0008, ADR 0009

## 1. Risk Matrix & Scenarios

Scenario IDs are stable traceability anchors. Every story's §17 Gherkin acceptance
criterion must reference its corresponding ID.

| Scenario ID | Behavior Description | Risk | Test Level | Target File/Module |
|---|---|---:|---|---|
| SC-e89s01-P0-01 | Production startup rejects a missing or malformed canonical root key; explicit development plaintext mode is opt-in only. | P0 | Integration | `main.go`, `components/internal/envcrypto`, `components/sites` |
| SC-e89s01-P0-02 | Site secret create/update returns metadata and a masked preview, never `value` or ciphertext. | P0 | HTTP integration | `components/sites/env_vars.go` |
| SC-e89s01-P0-03 | REST and MCP Site-secret operations reject cross-organization and cross-Site targets without disclosing existence. | P0 | HTTP integration | `components/sites`, `components/mcp`, `adapters.go` |
| SC-e89s01-P0-04 | Build and runtime resolution use identical scope, precedence, and redaction behavior. | P0 | Integration | `components/deploy/envresolver.go`, `engine.go` |
| SC-e89s01-P0-05 | A selected Site-secret decryption failure stops protected delivery and never silently drops the variable. | P0 | Integration | `components/deploy`, `components/sites` |
| SC-e89s01-P0-06 | Secret values are absent from build, runtime, diagnostic, and LLM log output, including tool-echoed values. | P0 | Integration | `components/deploy`, `components/internal/llm` |
| SC-e89s01-P1-07 | Oversized or invalid Site-secret payloads are rejected without persistence or plaintext error output. | P1 | HTTP integration | `components/sites` |
| SC-e89s02-P1-01 | A Project and Environment are owned by the authenticated organization, not request body or path identity. | P1 | HTTP integration | `components/projects`, `components/auth` |
| SC-e89s02-P1-02 | Cross-organization Project reads and mutations are non-disclosing. | P1 | HTTP integration | `components/projects`, `components/api` |
| SC-e89s02-P1-03 | New Sites receive exactly one Project and production Environment. | P1 | Integration | `components/projects`, `components/sites` |
| SC-e89s02-P1-04 | Existing Sites migrate idempotently to one compatibility Project and production Environment without changing secret values. | P1 | Migration integration | `components/projects`, `components/sites`, `components/db` |
| SC-e89s02-P1-05 | Project deletion is blocked while Sites remain attached; valid deletion preserves referential integrity. | P1 | Integration | `components/projects`, `components/sites` |
| SC-e89s03-P0-01 | Stored Project secrets contain ciphertext and metadata only; plaintext and root key material are absent from database rows. | P0 | Integration | `components/secrets`, `components/db` |
| SC-e89s03-P0-02 | Secret updates create immutable versions; previous versions remain decryptable. | P0 | Integration | `components/secrets` |
| SC-e89s03-P0-03 | AES-GCM authenticated data binds ciphertext to Project, Environment, Folder, Secret, and version scope. | P0 | Unit + integration | `components/secrets`, `components/internal/envcrypto` |
| SC-e89s03-P0-04 | Concurrent first writes create exactly one Project Data Key and both writes decrypt correctly. | P0 | Race integration | `components/secrets`, `kernel`, `components/db` |
| SC-e89s03-P0-05 | Wrong root key, key ID, algorithm, nonce, or ciphertext fails closed without leaking plaintext. | P0 | Unit | `components/secrets` |
| SC-e89s03-P1-06 | Backup/restore preserves ciphertext, versions, key identifiers, and decryptability; interrupted rotation resumes from a checkpoint. | P1 | Integration | `components/secrets`, `components/backup` |
| SC-e89s04-P0-01 | Secret list and create/update responses contain metadata and masked previews only. | P0 | HTTP contract | `components/secrets`, `components/api` |
| SC-e89s04-P0-02 | Describe permission does not grant value read; explicit value-read permission is required for `/value`. | P0 | HTTP contract | `components/secrets`, `components/auth` |
| SC-e89s04-P0-03 | Cross-organization key, version, and value requests return non-disclosing errors. | P0 | HTTP contract | `components/secrets`, `components/auth` |
| SC-e89s04-P0-04 | Audit events include actor and scope metadata but never plaintext or ciphertext. | P0 | Integration | `components/secrets`, `components/monitoring` |
| SC-e89s04-P1-05 | Mutating routes enforce input size limits and rate limits; malformed keys cannot alter SQL structure. | P1 | HTTP contract | `components/secrets`, `components/api` |
| SC-e89s04-P1-06 | Version listing is bounded, ordered, and independent from value-read authorization. | P1 | Integration | `components/secrets` |
| SC-e89s05-P1-01 | `/secrets` Project → Environment → Folder lists masked metadata without retaining plaintext in list state. | P1 | Component + browser E2E | `ui/src/pages`, `ui/src/components` |
| SC-e89s05-P1-02 | Authorized Reveal calls the explicit `/value` route, displays one value, and clears it on close/unmount. | P1 | Component + browser E2E | `ui/src/components`, `ui/src/lib` |
| SC-e89s05-P1-03 | Unauthorized Reveal and mutation failures render value-free 401/403 errors. | P1 | Component | `ui/src/components`, `ui/src/lib` |
| SC-e89s05-P1-04 | Edit forms never prefill an existing plaintext value. | P1 | Component | `ui/src/components` |
| SC-e89s05-P1-05 | Mixed-validity `.env` import saves valid keys, identifies invalid keys, and never echoes submitted values. | P1 | Component + browser E2E | `ui/src/components`, `ui/src/lib` |
| SC-e89s05-P1-06 | `/secrets` has keyboard-accessible navigation, labels, focus behavior, destructive confirmation, and no axe violations. | P1 | Browser E2E | `tests/e2e/project-secrets-ui.spec.ts`, `tests/e2e/axe-scan.spec.ts` |
| SC-e89s06-P0-01 | Site compatibility values override Project values on key collision. | P0 | Unit + integration | `components/deploy/envresolver.go` |
| SC-e89s06-P0-02 | Build-only values are excluded from runtime and runtime-only values are excluded from build. | P0 | Integration | `components/deploy/engine.go` |
| SC-e89s06-P0-03 | Runtime, build, diagnostic, and LLM logs redact resolved Project and Site values. | P0 | Integration | `components/deploy`, `components/internal/llm` |
| SC-e89s06-P0-04 | Legacy Site values remain deployable through the compatibility layer during native-first dual-read. | P0 | Migration integration | `components/deploy`, `components/backup` |
| SC-e89s06-P0-05 | Unknown legacy format stops migration without changing rows and returns an actionable, value-free error. | P0 | Migration integration | `components/backup`, `components/deploy` |
| SC-e89s06-P1-06 | Restart, resume, rollback, and child-process environment behavior preserve scope and redaction invariants. | P1 | Integration | `components/deploy`, `components/backup` |
| SC-e89s07-P0-01 | MCP Project reads are organization-scoped and do not disclose another organization's Project. | P0 | HTTP integration | `components/mcp`, `components/auth` |
| SC-e89s07-P0-02 | `mcp:secrets:read` permits list/get but not set/delete; `mcp:secrets:write` is independently enforced per tool. | P0 | HTTP + stdio integration | `components/mcp/auth.go`, `components/mcp/mcp.go` |
| SC-e89s07-P0-03 | A Site deploy key cannot target another Site or read Project secrets without explicit policy. | P0 | HTTP integration | `components/mcp`, `components/auth`, `adapters.go` |
| SC-e89s07-P0-04 | Existing Site env tools use authenticated target ownership rather than caller-supplied Site IDs. | P0 | HTTP integration | `components/mcp`, `components/sites` |
| SC-e89s07-P0-05 | MCP internal failures return typed generic errors and never stringify SQL, key material, or plaintext. | P0 | HTTP + stdio integration | `components/mcp` |
| SC-e89s07-P1-06 | MCP list and mutation results remain masked/metadata-only, with bounded arguments and audit actor metadata. | P1 | Integration | `components/mcp`, `components/secrets` |

## 2. Level Strategy

- **Unit:** Pure encryption, AAD construction, key parsing, masking, redaction,
  precedence merge, scope validation, and response-shape serializers. These tests
  stay deterministic and do not require a running server.
- **Component integration:** Use real in-memory SQLite and `httptest` handlers for
  migrations, foreign keys, policy matrices, SecretManager operations, deploy
  resolver behavior, audit rows, MCP HTTP handlers, and backup/restore.
- **Race integration:** Run concurrent first-key creation and transaction tests with
  `-race`; include SQLite and the PostgreSQL implementation seam. A live PostgreSQL
  run is an environment-gated release check, not a substitute for SQLite coverage.
- **Browser E2E:** Use the existing root Playwright config and server lifecycle for
  `/secrets` masking, reveal, import, permission denial, destructive confirmation,
  keyboard navigation, and axe route coverage.
- **MCP stdio:** Exercise the same per-tool scope matrix through the stdio transport;
  HTTP-only coverage is insufficient because authentication context propagation differs.

## 3. Fixture Architecture & Isolation

### Go data factories

Add test-local factories, not production fixtures:

- `OrganizationFactory`: creates isolated organizations, users, memberships, and
  role combinations.
- `ProjectFactory`: creates an organization-owned Project, Environment, Folder, and
  optional Site attachment.
- `SecretFactory`: writes through `SecretManager`, returns metadata plus a separately
  held expected plaintext; tests must never derive plaintext from list responses.
- `MCPKeyFactory`: creates org keys with exact scope sets and Site deploy keys bound to
  one Site.
- `LegacyEnvFactory`: creates explicit plaintext or legacy ciphertext rows and an
  explicit migration mode.

Every factory receives `t *testing.T` and a fresh database. No shared global database,
root key, authentication token, or mutable fixture state.

### Database state

- Default: in-memory SQLite per test, with migrations run through the component
  lifecycle.
- Foreign-key and idempotency tests run migrations twice on the same isolated DB.
- Transaction/race tests use separate connections against one isolated DB and assert
  exactly one Project Data Key.
- PostgreSQL tests use the existing driver abstraction and an opt-in DSN. They must
  be skipped with an explicit reason when no DSN is configured, never silently passed
  as SQLite coverage.
- Backup/restore tests use temporary files and verify ciphertext, version metadata,
  key IDs, and decryptability without logging root keys or plaintext.

### HTTP and MCP fixtures

- Use `httptest.NewRecorder`/`httptest.NewRequest` for Go handlers.
- Attach authenticated organization, Project, Environment, Folder, and Site identity
  through context/auth middleware; never put trusted identity only in request paths.
- Use a table-driven action matrix for describe/read/create/update/delete/version
  permissions and expected 401/403/non-disclosing 404 results.
- MCP tests use both HTTP and stdio sessions with the same key fixture and assert tool
  result types, not only text fragments.

### UI fixtures

- Mock `fetch` at the API boundary with deterministic response fixtures; do not add
  MSW because the repository currently uses direct fetch mocks and React Testing
  Library.
- Keep separate `SecretMetadata` and `SecretValue` fixtures. The metadata fixture
  must not contain a `value` property.
- Browser tests seed data through API requests and use unique organization/project
  names per run. Submitted import values are test-only sentinels and must be asserted
  absent from rendered errors and page text.

## 4. NFR Verification

| NFR Type | Requirement | Verification Command |
|---|---|---|
| Confidentiality | No plaintext/ciphertext/root key in database responses, logs, audit rows, or UI errors. | `go test ./components/sites ./components/deploy ./components/secrets ./components/mcp ./components/internal/llm -run 'Test.*(Secret|Redact|Audit|Error)' -count=1` |
| Tenant isolation | No cross-organization Project/Secret/MCP/Site-key access. | `go test ./components/projects ./components/secrets ./components/mcp ./tests/contract -run 'Test.*(Org|Isolation|Binding|Secret)' -count=1` |
| Crypto integrity | AES-GCM, AAD scope binding, immutable versions, wrong-key failure, and nonce uniqueness. | `go test -race ./components/secrets ./components/internal/envcrypto ./kernel ./components/db -count=1` |
| Migration reliability | Idempotent schema creation, explicit legacy format, resumable migration, backup/restore verification. | `go test ./components/projects ./components/deploy ./components/backup ./components/secrets -run 'Test.*(Migration|Legacy|Backup|Restore)' -count=1` |
| Runtime reliability | Protected deployment does not start with missing/decryption-failed secrets; restart/resume preserves scope. | `go test ./components/deploy -run 'Test.*(Runtime|Resolver|Restart|Rollback|Injection)' -count=1` |
| Abuse resistance | Secret payloads/imports/list operations are bounded and mutations are rate-limited. | `go test ./components/sites ./components/secrets ./components/api -run 'Test.*(Limit|Rate|Payload|Import)' -count=1` |
| Browser accessibility | `/secrets` keyboard flow, accessible names, focus handling, and axe scan pass. | `npx playwright test tests/e2e/project-secrets-ui.spec.ts tests/e2e/axe-scan.spec.ts --config tests/e2e/playwright.config.ts --grep 'Project Secrets'` |
| Full release regression | All packages, race-sensitive components, UI build, and static analysis pass. | `go test -race ./components/sites/... ./components/deploy/... ./components/mcp/... ./components/auth/... ./components/secrets/... && go test ./... && go build ./... && go vet ./... && cd ui && npx tsc --noEmit && npm run build` |

No independent latency SLO is defined in the epic. The plan therefore gates bounded
work, rate limits, migration reliability, and regression commands rather than inventing
a response-time target.

## 5. Execution Order and Evidence

1. Implement P0 scenarios for e89s01, then run its task verifies and persist
   `specs/verifications/e89s01-verify.yaml`.
2. Implement e89s02 P1 scenarios, then e89s03 P0 storage scenarios. Do not open the
   REST/Deploy parallel wave until the SecretManager and transaction seams pass.
3. Run e89s04 and e89s06 test plans in separate worktrees. Merge only after each
   branch passes its story-level gates.
4. Run e89s05 UI and e89s07 MCP scenarios in parallel after the REST `/value` contract,
   `/secrets` route, and MCP principal/scope contracts are frozen.
5. Before release, run all NFR commands, security review, blind-spot/completeness
   checks, trace generation, then `gate-trace` last.
6. A failing P0 scenario blocks its story and every dependent wave. A failing P1
   scenario blocks that story's completion but does not invalidate independent
   foundation contracts.

## 6. Out of Scope

- Production test code, test implementation, or fixture implementation in this plan.
- External KMS/HSM, dynamic secrets, provider rotation, secret sync, PKI, Kubernetes,
  generated SDKs, secret scanning, approvals, sharing, and cross-project imports.
- Destructive legacy data re-encryption without an explicit operator-selected format.
- UI secret export or password generation.
