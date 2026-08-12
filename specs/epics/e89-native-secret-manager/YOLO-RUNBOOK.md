type: execution-runbook
context: domain
epic: e89
mode: yolo
status: blocked-by-security-gate

# e89 YOLO Execution Runbook

## Operating rule

YOLO means automatic progression between gates with no human checkpoint after each
story. It does **not** mean bypassing Bigpowers hard gates. A failed security, preflight,
verify, audit, traceability, CI, or release gate stops the affected wave and routes back
to the owning story.

Current blocker: `specs/security/epics/e89/REVIEW.md` is NOT READY with four HIGH baseline
findings. Do not start RED commits until e89s01 remediation is implemented and the
security review is refreshed against the implementation diff.

## Start conditions

1. Confirm `main` is clean at the current state hash recorded in `specs/state.yaml`.
2. If user-owned source changes are present, stop. Do not stash, reset, overwrite, or
   include them in the e89 checkpoint; reconcile the active flow first.
3. Create coordinator-owned `specs/agent-locks.yaml` with `locks: []` if absent.
4. Run repository preflight. Any red baseline routes to `quick-fix`/`fix-bug` first.
5. Create `feat/e89-integration` from the clean main checkpoint.
6. Run build-epic step 0 security review. The current pre-implementation report is a
   threat model, not a passing implementation review.

## Story loop

For every story, the coordinator automatically runs:

1. `security-review` against the story diff and threat model.
2. `survey-context`; write `started_at` under the story in `execution-status.yaml`.
3. `assess-impact --lightweight`; if risk exceeds 7, run `grill-me` before code.
4. `plan-work`; preserve the story's scenario IDs, risk/security/Allure fields, and
   failing ledger.
5. `kickoff-branch`; acquire the story lock and verify a green baseline.
6. `develop-tdd`; for every task: RED test-only commit → GREEN implementation commit →
   optional refactor. Flip the task to `passing` only after its verify command exits 0.
7. `verify-work`; run mechanical gates, P0 security/NFR/UAT, and persist verification.
8. `audit-code --gate`; failure resets the story to develop-tdd. Run `enforce-first`.
9. `commit-message`; check Conventional Commits and no AI attribution.
10. Merge the accepted story branch into `feat/e89-integration`, update state/status,
    release the story lock, and open the next dependency wave.

## Parallel waves

```text
s01 → s02 → s03 → (s04 || s06) → (s05 || s07) → integrated release
```

- s01 is the security foundation: canonical key wiring, fail-closed resolution, MCP
  Site binding, metadata-only writes, and redaction.
- s02 owns Project/Environment schema and Site compatibility attachment.
- s03 owns encrypted immutable storage and freezes the SecretManager/transaction seam.
- s04 and s06 branch from the accepted s03 integration point. s04 owns REST policy/audit;
  s06 owns Deploy resolution/migration. Neither edits `main.go`.
- s05 and s07 branch after s04's REST contract. s05 owns UI; s07 owns MCP. Neither
  edits shared Go composition wiring.
- The coordinator applies all `main.go` composition-root changes serially after branch
  review. Never merge parallel `main.go` edits.

## Automatic integration checkpoint

After each story merge, run:

```text
affected story verify commands
go test ./affected/packages -count=1
go vet ./affected/packages
```

After s03, verify the public SecretManager contract before opening s04/s06. After s04,
verify the REST `/value` contract, role/action matrix, and response shapes before opening
s05/s07.

## Final release gate sequence

1. All 30 task entries are `status: passing`.
2. Story verification YAML and audit artifacts exist.
3. Run `scripts/trace-stories.sh --json`; if missing, report `trace skipped`.
4. Run blind-spot and completeness checks.
5. Refresh security review, then `audit-code --gate`, then `enforce-first --quick`.
6. Run `gate-trace` last, after trace inputs exist.
7. Run `commit-message`, CI wait, and `release-branch`.
8. Run OKF refresh scripts when present; otherwise report `OKF wiki refresh skipped`.
9. Archive the capsule only after all seven stories are done and CI is green.

## Non-bypassable stop conditions

- Any HIGH security finding with confidence ≥8.
- Dirty worktree, missing story lock, or red preflight.
- Failed task verify, P0 NFR/UAT gap, audit failure, or F.I.R.S.T. violation.
- Contract drift between s03 and s04/s06, or REST/MCP/UI response-shape drift.
- Traceability FAIL, unresolved CI failure, non-conventional commit, or AI attribution.

## Required artifacts

- `specs/security/epics/e89/THREAT_MODEL.md`
- `specs/security/epics/e89/REVIEW.md`
- `specs/IMPACT-e89-e89sYY.md`
- `specs/verifications/e89sYY-verify.yaml`
- `specs/verifications/AUDIT-e89-e89sYY.md`
- `specs/state.yaml`
- `specs/execution-status.yaml`
