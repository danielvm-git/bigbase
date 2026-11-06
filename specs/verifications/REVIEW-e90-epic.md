# Independent Review — Epic e90

**type:** review-report
**context:** execution
**epic:** e90
**target:** `git diff main...HEAD` on `feat/e90-dev-environment-hardening`
**reviewer:** fresh-context reviewer stance (no shared reasoning with the build pass)
**verdict:** APPROVE WITH FOLLOW-UPS (0 blocking, 3 accepted with logged follow-ups, 2 fixed)

> Note on method: this harness exposes no subagent dispatch tool, so `request-review`
> could not hand the diff to a genuinely separate agent. The review was performed from
> a clean read of the diff against the epic's acceptance criteria and CONVENTIONS.md,
> deliberately looking for what the build pass would have rationalised. The limitation
> is recorded in `specs/state.yaml` `decisions[]`.

## What was reviewed

1. Deploy state machine: `forceStatus`, the drain guard, transition ordering.
2. Gate scripts and their fixture tests; whether the gates can regress silently.
3. CI wiring: do the new gates actually block a merge?
4. Spec hygiene: statuses, verify evidence, trace markers, deleted flat epic shard.

## Findings

### F-1 (fixed) — Trace markers pointed at files that do not obviously implement the story

`tests/e2e/axe-scan.spec.ts` was tagged `e90s04` and `components/deploy/state_lifecycle_test.go` `e90s09`
with no explanation, which reads as tag-padding. A traceability reader cannot tell whether the
link is real.

**Response:** both markers now state the relationship ("frontend test gates (test-ui / test-e2e)",
"polls via waitForDeploymentTerminal, never a bare sleep"). Fixed.

### F-2 (fixed) — A drift gate could pass silently when it cannot see the files

`check-version-drift.sh` used relative paths with no root override; run outside the repo it read
nothing and exited 0 — a green gate that checked nothing.

**Response:** the script now resolves the repo root (override or `git rev-parse`) and fails loudly
instead of silently passing; the behaviour is documented in the header. Fixed.

### F-3 (accepted, logged) — `forceStatus` is a second mutation path around the state machine

`stopDeployment` can now write a status the machine would reject (for example `running → stopped`),
recording history as it goes. That is intentional (force-stop of an orphan has no legal path), but it
weakens the "one validated path" property that `TransitionState` provides.

**Reason accepted:** the alternative — teaching the state machine that every state may jump to
`stopped` — would legalise the very transition the machine exists to prevent, and the force path is
reachable only from `stopDeployment` (2 callers, both intentional stops). It is logged in
`state.yaml` as a follow-up: if a third caller appears, move it onto a `ForceState` method on
`StateMachine` so the escape hatch is declared next to the rules.

### F-4 (accepted, logged) — Status and history are still two writes

`TransitionState` (and now `forceStatus`) commit the status first and append `status_history` after.
A reader can observe the new status before its history entry. The tests now codify this as eventual
consistency and poll, which is correct for readers, but it means "status X implies history contains X"
is not a durable invariant.

**Reason accepted:** making it atomic means a single UPDATE that read-modify-writes the JSON history
inside the same statement — a schema/statement change well outside a dev-environment hardening epic.
Logged as a follow-up in `state.yaml`, and the contract is now documented in CONVENTIONS.md so future
tests poll instead of asserting once.

### F-5 (accepted) — The new gate tests run under the `verify` job only

`node --test scripts/*.test.cjs` runs in `verify`, which is gated on `code == 'true'` and on PR/push-to-main.
A docs-only change therefore does not exercise the gate tests.

**Reason accepted:** the gate scripts themselves only run in `verify` too, so the tests cover exactly
the paths that can regress in a given run; adding a second job would cost ~1 minute per run for no
extra signal.

## Checks that came back clean

- The CI gates do block: `verify` needs `test-race`; `release` needs `verify`, `test-ui`, `test-race`;
  `build` needs `release`.
- No new dependency, no secrets, no new network call, no `@ts-ignore`/`eslint-disable`.
- `gosec`, CodeQL, gitleaks, go vet and the full Go suite are green; UI lint/typecheck/vitest(755)/build green.
- Conventional Commits respected; no AI attribution in any commit.
- The deleted flat shard (`specs/epics/e90-dev-environment-hardening.yaml`) has no remaining references;
  `release-plan.yaml` points at the capsule.
- Deploy package stable under `-race` (full package twice) and the new regression test stable at `-count=5`.
