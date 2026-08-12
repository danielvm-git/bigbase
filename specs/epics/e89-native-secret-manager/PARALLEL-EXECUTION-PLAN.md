type: execution-plan
context: domain
id: e89
status: ready-for-gated-execution

# e89 Parallel Execution Plan

## Decision

Do not run seven implementation branches at once. e89 has a dependency chain through
its storage contracts, but it also has two safe parallel waves. Use one worktree per
story, with a coordinator-owned integration branch. Only the coordinator edits
`specs/state.yaml`, `specs/execution-status.yaml`, release metadata, and final trace
artifacts.

The epic remains security-first: no story is marked done until its task ledger is
passing, its P0/P1 verification evidence exists, and the story audit gate passes.

## Current facts

- Epic: e89 Native Infisical-Inspired Secret Manager; 29 BCPs; tier 1; WSJF 8.4.
- Dependencies: e18, e23, e38, e41.
- Security model: `specs/security/epics/e89/THREAT_MODEL.md`; CRITICAL before s01 and HIGH during project-secret rollout.
- Existing ledgers: all seven are `status: failing`, as required before TDD.
- Current baseline observed: `main` at `b535145d1`; the working tree contains the e89 planning specs plus an untracked non-spec helper directory `scripts/lib/`. Kickoff is blocked until the coordinator reviews and checkpoints the spec artifacts and separately classifies/preserves or commits the helper; do not silently treat the tree as spec-only.
- No implementation worktree exists. `scripts/land-branch.sh` exists; `scripts/trace-stories.sh` and the review-worktree helper are absent and must be reported as skipped if still absent.
- No new external dependency is planned. Slopcheck: `[OK]` for the existing Go/React/SQLite stack; do not add a package without a new plan-work review.

## Dependency DAG and waves

```text
W0  threat model + impact gates + contract review (read-only/coordinator)
     |
W1  e89s01  harden existing Site secrets and delivery
     |
W2  e89s02  Projects, Environments, Site compatibility attachment
     |
W3  e89s03  SecretFolder/Secret/SecretVersion + KeyHierarchy
     |
W4  +-----------------------------+
     |                             |
     v                             v
  e89s04 REST/policy/audit       e89s06 Deploy resolver/dual-read
     |                             |
     +--------------+--------------+
                    v
              W5  e89s05 UI  ||  e89s07 MCP tools
                    |              |
                    +------+-------+
                           v
W6  integrated security, traceability, full verification, audit, release
```

The capsule's nominal implementation order remains the review order. The only
out-of-order execution is e89s06 after e89s03 and alongside e89s04; it consumes the
public SecretManager seam and does not edit REST or UI files. If the release tooling
enforces a strict story order, queue the e89s06 merge, not its isolated work, until
e89s04 is accepted.

## Worktree protocol

1. On `main`, run the repository preflight from `CLAUDE.md`/`AGENTS.md`. A red baseline routes to `quick-fix`/`fix-bug`; it does not become an e89 exception.
2. Resolve the dirty-tree gate: checkpoint the current spec artifacts, and explicitly review `scripts/lib/plan-consistency-check.sh` before deciding whether it belongs in the checkpoint. Do not mix source changes into that checkpoint.
3. Create the integration branch/worktree from the clean checkpoint:
   `feat/e89-integration` at `main`.
4. For each story, create a short-lived worktree from the latest accepted integration
   branch: `feat/e89s01`, `feat/e89s02`, etc. Never branch two stories from stale `main`.
5. Acquire the story lock before tests. The repository currently has no
   `specs/agent-locks.yaml`; the coordinator should create it with `locks: []` and own
   lock release. Agents must not edit shared state concurrently.
6. Each story agent owns only its story code, tests, and its own `e89sYY-tasks.yaml`.
   Each agent runs RED → GREEN → optional REFACTOR with separate conventional commits.
7. The coordinator reviews the diff, runs the story verify commands, merges/cherry-picks
   the branch into `feat/e89-integration`, updates state/status, and only then opens
   the next dependent wave.
8. Worktree names are deterministic: `../bigbase-e89s01`, …, `../bigbase-e89s07`,
   plus `../bigbase-e89-integration`. Remove them only after release verification.

## Ownership matrix

| Story | Primary owned paths | Must not edit in parallel |
|---|---|---|
| s01 | `main.go`; `components/internal/envcrypto`; `components/internal/llm` redaction seam; `components/sites/env_vars.go`, `sites.go`; `components/deploy/envresolver.go`, `env_vars.go`, `engine.go`; existing MCP adapter wiring and tests | shared auth/composition code after the branch is cut |
| s02 | new `components/projects` component (Project/Environment schema, handlers, dependency contract), Site attachment in `components/sites/sites.go`, related API/auth/kernel scope tests, and coordinator composition wiring | s01 Site migration/key files; s07 MCP files; do not edit `main.go` in the story branch |
| s03 | new `components/secrets` storage/key hierarchy/manager; optional transaction seam in `kernel`/DB implementations; DB/backup schema tests; crypto tests | s04 handlers and s06 resolver consumers; consume s01 `envcrypto` contract rather than editing it; freeze the public seam before W4 |
| s04 | Secret REST adapter/handlers, policy helper, rate limits, audit event adapter, contract tests | `components/deploy`, UI, and `main.go`; export constructor/options for coordinator wiring; do not change the s03 storage core |
| s05 | new Project Secrets UI page/components/lib/types/tests plus route/navigation registration | Go code, MCP code, REST handlers |
| s06 | `components/deploy/envresolver.go`, `env_vars.go`, `engine.go`, `orchestrator.go`, `manifest.go`, backup migration command/tests | `components/internal/llm` (consume s01 redaction seam), `components/secrets`, REST/UI/MCP handlers, and `main.go`; export resolver/deploy options for coordinator wiring |
| s07 | `components/mcp/auth.go`, `mcp.go`, MCP adapters, narrow auth-scope/principal wiring, MCP/e2e tests | UI and deploy resolver; `main.go` composition wiring is coordinator-only; rebase from accepted s04 contract |

`main.go` is coordinator-owned after s01. The coordinator applies the s02/s03/s04/s06/s07
composition-root patches one at a time after each branch is accepted; no parallel story
branch edits it. This removes the hidden overlap reported by the wave reviews.

The s02 project component is intentional: it gives Project/Environment migrations a
kernel dependency boundary instead of relying on nondeterministic component registration.
Reason for depth: Sites and SecretManager need deterministic schema availability, and
an explicit ECC component is simpler to verify than cross-component startup ordering.

Shared-file rule: if a story needs a path outside its matrix, stop and ask the
coordinator to assign ownership. Never solve an overlap by concurrent edits.

## Contract freeze before W4

e89s03 must publish and test these seams before e89s04/e89s06 work starts:

- A typed SecretManager interface for metadata listing, explicit value reads, writes,
  immutable version listing, deletion, and scoped deployment resolution.
- Metadata/value separation: list and mutation results never contain plaintext or
  ciphertext; explicit value reads use a distinct result type.
- REST value reads use the explicit route `GET /api/projects/{project}/environments/{env}/secrets/{key}/value`.
  s04 owns the route and s05/s07 consume this frozen shape; no adapter may infer reveal
  behavior from list or mutation responses.
- Scope identity is authenticated organization → Project → Environment → Folder →
  Secret. Callers cannot supply tenant identity as an authorization substitute.
- Envelope encryption uses the canonical base64 32-byte root key, per-Project data
  keys, AES-256-GCM, fresh nonces, version/key identifiers, and AAD containing stable
  scope metadata. The legacy hex key is migration input only. s03 consumes the s01
  `envcrypto` contract; any primitive extension is a coordinator-reviewed handoff.
- Project and SecretManager components declare explicit kernel dependencies so project
  migrations precede Site attachment and secret storage. If atomic first-key creation
  cannot be expressed with existing `DBer`, add an optional `kernel.TxBeginner` seam
  implemented by SQLite and PostgreSQL without widening every component interface.
- Exact startup dependency edges are part of s02/s03 acceptance: `projects -> db`,
  `sites -> projects`, and `secrets -> projects`. Deploy and REST/MCP adapters consume
  the already-started SecretManager through composition-root injection; registration
  order must not be the schema-order mechanism.
- EnvResolver precedence is explicit and deterministic: platform baseline, manifest
  configuration, native Project Environment values, Site compatibility values (Site
  wins on collision), then reserved runtime values. Build-only values never enter
  runtime; protected decryption failures are fatal.
- MCP principal context supports both organization keys and Site deploy keys. Scope
  names are `mcp:secrets:read` and `mcp:secrets:write`; target ownership is resolved
  before every operation. Site deploy keys remain Site-bound.

Any contract change after the freeze requires a coordinator decision and a focused
ADR update before dependent branches continue.

- Freeze the Admin UI route as `/secrets` for the first release. s05 updates `App.tsx`,
  `Layout.tsx`, `PAGE_TITLES`, and the Playwright/axe route inventory together. The UI
  route is separate from the existing Site environment tab.
- `get_project_secret` is metadata-only; explicit value reads use the separate
  `read_project_secret_value` MCP action and the REST `/value` endpoint. Both require
  the read-value scope/policy, are audited, and return a distinct value response.
- s07 must add a principal abstraction that can represent authenticated organization
  keys and Site deploy keys. Existing `OrgKeyAuthenticator` only resolves `bb_` keys
  to org ID/scopes, while Site-key resolution returns a Site ID; the coordinator must
  wire both without allowing a caller-supplied Site ID to override authenticated
  binding.
- The W5 coordinator wiring is not optional: s05 needs a working REST route, and s07
  needs `mcp.Options` to receive the SecretManager/ownership adapter. Both branches
  expose seams; the coordinator patches `main.go` after each branch is reviewed.
- REST policy responses use the existing `{"error":"..."}` JSON shape: 401 for
  unauthenticated requests, 403 for authenticated callers without the action, and a
  non-disclosing 404 for cross-organization targets. s05 renders these value-free
  errors and does not infer permission from client state; authorization remains server
  authoritative.


## Story gates

Every story follows the complete build-epic cycle, not a reduced parallel shortcut:

0. `security-review`: use the e89 threat model; update the story risk notes.
1. `survey-context`: confirm the active story and write `started_at` in
   `execution-status.yaml`.
2. `assess-impact --lightweight`: write `specs/IMPACT-e89-e89sYY.md`; risk > 7 requires
   `grill-me` before implementation. Run plan consistency before code.
3. `plan-work`: preserve the existing countable steps, `risk`, `security`, `allure`,
   and failing-ledger statuses. Carry BCPs into `state.yaml`.
4. `kickoff-branch`: clean checkpoint, story lock, preflight green.
5. `develop-tdd`: public-contract tests first; separate RED and GREEN commits; update
   the task ledger to `passing` only after its verify command exits 0.
6. `verify-work`: P0 stories require full mechanical/UAT/security/NFR evidence. P1 UI
   uses standard verification. Persist `specs/verifications/e89sYY-verify.yaml`.
7. `audit-code --gate`: write `specs/verifications/AUDIT-e89-e89sYY.md`; failure loops
   to develop-tdd. Run `enforce-first --quick` after audit passes.
8. Coordinator runs trace/wiki refresh, marks the story done with `completed_at`,
   syncs status, and merges the accepted branch.

### e89s01 — security foundation

Single owner. Keep compatibility behavior while changing the source of truth:
canonical root-key validation, explicit development plaintext opt-in, metadata-only
Site writes, fatal selected decryption errors, build/runtime resolver parity, log
redaction, and ownership-aware existing MCP paths. Do not add Project tables here.

s01 owns the `components/internal/llm` redaction integration as part of the shared
redaction contract. s06 consumes that seam and must not edit the LLM package in its
parallel branch.

s01 is the runtime-resolver foundation. s06 must not reimplement its fatal-error or
redaction behavior; it extends the frozen resolver with native Project and Site
compatibility layers and consumes the s01 error/redaction contract.

Required task verifies are the five commands in `e89s01-tasks.yaml`, followed by the
affected package regression command. This story is the first release blocker because
all later stories rely on its fail-closed and masking invariants.

### e89s02 — tenant-scoped structure

Single owner after s01 is integrated. Migrations must be idempotent for SQLite and
PostgreSQL: existing Sites receive exactly one compatibility Project and production
Environment; new Sites receive the same association; deletion is blocked while a Site
is attached. No secret values move in this story.

### e89s03 — encrypted immutable storage

Single owner after s02. Keep metadata separate from ciphertext, enforce uniqueness and
foreign keys, serialize first Project Data Key creation, bind AAD to scope, preserve
old versions, and test wrong-key/wrong-scope, backup/restore, and race behavior. This
story owns the contract freeze required by the two W4 worktrees.

### e89s04 — REST policy surface

Run in parallel with s06 from the accepted s03 integration point. Use authenticated
org context, one policy helper for describe/read/mutate/version actions, masked list
and mutation results, bounded input/rate limits, non-disclosing cross-org errors, and
value-free audit records. Do not duplicate crypto or storage logic.

### e89s06 — deployment resolution

Run in parallel with s04 from the accepted s03 integration point. Consume only the
frozen SecretManager seam and the s01 runtime resolver/error/redaction contract. Add
only the native Project layer, deterministic precedence, and explicit dual-read
migration; do not reimplement s01 runtime failure or redaction behavior. Replace
runtime `FetchSiteEnvVars` and manifest append logic with one resolver, preserve
Site-over-Project precedence, keep static serving secret free, redact logs/diagnostics/
LLM output through the shared primitive, and make legacy migration explicit and
resumable. Do not modify REST policy or MCP handlers.

### e89s05 and e89s07 — adapter wave

After s04's REST contract is accepted, run the UI and MCP worktrees concurrently. The
UI uses separate metadata/value TypeScript types, explicit reveal, version history,
safe `.env` import, and value-free errors. MCP adds narrow scopes, target ownership
checks, masked defaults, explicit value reads, safe typed errors, and real HTTP
cross-org/site-key tests. Both adapters consume the same SecretManager policy; neither
reimplements authorization.
- s07 tests must exercise per-tool `mcp:secrets:read` versus `mcp:secrets:write`
  enforcement on both HTTP and stdio paths; declaring scope constants without a
  per-tool check is not sufficient and must not regress existing tier-read behavior.

## Integration and final release gates

After every merge: run that story's task verifies, `go test` for affected packages,
`go vet` for affected Go packages, and the UI typecheck/build when UI files changed.
After W5: run the integrated e89 scenario matrix across REST, Deploy, MCP, UI, SQLite,
and PostgreSQL paths. Then run, from the integration worktree:

```text
go test -race ./components/sites/... ./components/deploy/... ./components/mcp/... ./components/auth/... ./components/secrets/...
go test ./...
go build ./...
go vet ./...
cd ui && npx vitest run src/lib/secretsData.test.ts src/components/ProjectSecretsTab.test.tsx
cd ui && npx tsc --noEmit && npm run build
npx playwright test tests/e2e/project-secrets-ui.spec.ts --config tests/e2e/playwright.config.ts
npx playwright test tests/e2e/axe-scan.spec.ts --config tests/e2e/playwright.config.ts --grep "Project Secrets"
go test ./components/mcp -run 'TestMCP.*Secret|Test.*Secret.*MCP' -count=1

The s05 branch must add the named browser test and add `/secrets` to the axe route
inventory; the root Playwright config owns the server lifecycle because `ui` has no
Playwright script. The s07 branch must add the named MCP HTTP integration test; its
task-level test alone is not release evidence.

Run final quality gates in this order:

1. All seven ledgers passing and story verification artifacts persisted.
2. `bash scripts/trace-stories.sh --json` when present; this generates the matrix
   consumed by downstream trace checks. If absent, report `trace skipped`.
3. `bash scripts/check-blind-spots.sh` and `bash scripts/lib/completeness-critic.sh`.
4. `security-review`, then `audit-code --gate`, then `enforce-first --quick`.
5. `gate-trace` last, after trace and blind-spot inputs exist.
6. `commit-message`, CI verification, and `release-branch`.

Run `bash scripts/decompose-conventions.sh` and `bash scripts/generate-agent-guide.sh`
when present; if absent or failing, report `OKF wiki refresh skipped` visibly. Do not
hide missing tooling.

Release only after all seven story ledgers are passing, verification artifacts exist,
there are no unresolved HIGH security findings, traceability is not FAIL, CI is green,
and the coordinator has run `commit-message`. Use solo-git `scripts/land-branch.sh` or
team-PR flow according to `state.yaml`; archive the capsule only after all seven
stories are done.

## Coordinator stop conditions

- Any red baseline, story verify, security, audit, traceability, or CI gate stops the
  next wave and routes back to the owning story.
- A shared-file request pauses that branch until ownership is reassigned.
- A contract change pauses both W4 branches, updates the contract/ADR, and rebases
  them from the new integration point.
- Never claim parallel completion from unmerged worktrees; only the integrated branch
  is evidence.
