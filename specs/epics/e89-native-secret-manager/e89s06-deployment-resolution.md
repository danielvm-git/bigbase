# e89s06 — Project and Site Secret Deployment Resolution

**type:** feat
**risk:** P0
**context:** infra
**BCPs:** 5

## 1. Type

Deployment integration and migration.

## 2. Context

BigBase build resolution uses EnvResolver, while runtime uses the legacy
FetchSiteEnvVars path and appends manifest values separately.

## 3. Summary

Make EnvResolver the only build/runtime path, add Project Environment sources,
and migrate legacy Site values through dual-read compatibility.

## 4. Problem

Duplicated resolution causes precedence drift, secret leakage, and fail-open
startup when a value cannot be decrypted.

## 5. Users

All deployed Node, Go, Python, PHP, and static applications.

## 6. Solution

Resolve platform baseline, manifest configuration, Project secrets, Site
compatibility secrets, and reserved system values through one deterministic seam.
Inject only the requested build/runtime scope.

## 7. Alternatives

- Add a second project fetch helper: rejected because it repeats the existing defect.
- Migrate all values before dual-read: rejected because it creates downtime and ambiguity.

## 8. Dependencies

e89s01, e89s02, e89s03, existing Deploy Engine/Orchestrator, backup/restore.

## 9. Assumptions

Site compatibility values override Project values on collision. Static serving does
not receive secrets.

## 10. Risks

The current production database may contain plaintext/no-op rows. Migration must
require an explicit legacy storage mode.

## 11. Migration Plan

Native-first dual-read: native Project secrets are resolved first, legacy Site
values are read as compatibility values, and the resolver reports conflicts without
logging values. A resumable command later re-encrypts/moves legacy rows.

## 12. Data Model

No new tables beyond e89s02/e89s03. Add migration checkpoints and legacy-format
selection to the operator command.

## 13. API

No new public API. Deploy uses an injected SecretResolver/SecretManager seam.

## 14. Affected Code

`components/deploy/envresolver.go`, `env_vars.go`, `engine.go`, `orchestrator.go`,
`manifest.go`, `backup`, Site/Project lookup, integration tests.

## 15. Testing Strategy

Test all runtime scopes, precedence, reserved variables, redaction, decryption
failure, dual-read conflicts, restart/resume, and child-process environment behavior.

## 16. Rollback Plan

Disable native resolution through an explicit compatibility mode. Keep legacy
reads and prevent destructive migration until verification completes.

## 17. Acceptance Criteria

```gherkin
Scenario: [SC-e89s06-P0-01] Site secret overrides Project secret
  Given Project secret DB_URL=project and Site secret DB_URL=site
  When the deployment resolves runtime values
  Then DB_URL=site is injected

Scenario: [SC-e89s06-P0-02] Build-only secret is absent at runtime
  Given a Secret is marked build-only
  When the app starts
  Then the Secret is not present in the runtime environment

Scenario: [SC-e89s06-P0-03] Runtime logs redact resolved values
  Given an app prints a resolved Secret value
  When deployment logs are collected
  Then the plaintext value is absent from logs and diagnostics

Scenario: [SC-e89s06-P0-04] Legacy values remain deployable during migration
  Given a Site has only a legacy site_env_vars value
  When the Site deploys
  Then the resolver supplies the value through the Compatibility Layer

Scenario: [SC-e89s06-P0-05] Invalid legacy format is explicit
  Given a legacy row cannot be classified as plaintext or ciphertext
  When migration runs without a format mode
  Then migration stops with an actionable error and changes no rows
Scenario: [SC-e89s06-P1-06] Restart and rollback preserve resolution invariants
  Given a deployment is restarted, resumed, or rolled back
  When its child process receives environment values
  Then scope, precedence, static-secret exclusion, and redaction remain unchanged
```

## Requirements
+
#### MODIFIED: Deployment environment resolution
**Before:** Build resolution uses EnvResolver, while runtime uses FetchSiteEnvVars and appends manifest values separately; legacy values can be skipped after decryption errors.
**After:** One EnvResolver resolves platform, manifest, Project, Site compatibility, and reserved runtime values for both build and runtime scopes; protected decryption failures stop delivery; dual-read migration preserves legacy Sites.

## 18. Implementation Steps

1. Extend EnvResolver with Project Environment and Site compatibility sources → verify: `go test ./components/deploy -run 'TestEnvResolver.*Project|Test.*Precedence' -count=1`
2. Replace startApp/legacy fetch with the resolver and reserved-value policy → verify: `go test ./components/deploy -run 'TestEnvVarInjection|Test.*Runtime' -count=1`
3. Redact build, runtime, diagnostic, and LLM log paths → verify: `go test ./components/deploy ./components/internal/llm -run 'Test.*Redact|Test.*Secret' -count=1`
4. Add native-first dual-read and resumable legacy migration command → verify: `go test ./components/deploy ./components/backup -run 'Test.*Legacy|Test.*Migration|Test.*Backup' -count=1 && echo 'no new security findings in affected paths'`
5. Run all runtime and restart/rollback deployment integration coverage → verify: `go test ./components/deploy ./components/backup -run 'Test.*(Runtime|Restart|Rollback|Resume|Legacy|Backup)' -count=1 && echo 'no new security findings in affected paths'`

## 19. Verification Script

1. Create Project and Site secrets with overlapping keys.
2. Deploy a test app for each supported runtime.
3. Verify build-only/runtime-only behavior.
4. Print known values and inspect logs.
5. Remove native values and confirm legacy compatibility resolution.
6. Run migration in explicit plaintext and ciphertext modes.

## 20. Out of scope

Dynamic secrets, external KMS, provider rotation, secret syncs, containers, and
runtime isolation beyond existing systemd/process-group controls.
## 21. Zoom-Out Check

- **Purpose:** Deploy EnvResolver owns build/runtime environment composition, child-process injection, redaction, restart/resume, and rollback behavior; backup owns explicit migration/restore tooling.
- **Callers:** `buildApp`, `startApp`, orchestrator resume, rollback, diagnostics/LLM paths, supported runtime processes, and the composition root.
- **Contracts:** platform → manifest → Project → Site compatibility → reserved precedence, Site-over-Project collision behavior, build/runtime scope isolation, static secret exclusion, fail-closed decryption, explicit resumable legacy format, and the s01 redaction seam.
