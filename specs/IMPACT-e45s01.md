# Impact Report — e45s01: Graceful Connection Draining

## Target
Zero-downtime drain on deployment switchover. Adds `draining` state and defers old deployment shutdown until new deployment passes health check.

## Changed Files
- `components/deploy/state_machine.go` — Add `StateDraining`, `StateStopped` valid transitions
- `components/deploy/deploy.go` — Refactor `stopPreviousDeployments()` to defer shutdown; add drain timeout; integrate into `runDeployment()` lifecycle
- `components/deploy/deploy_test.go` — New drain tests

## Dependents (7)
- `deploy.go:565` — `stopPreviousDeployments` called from `Trigger()`
- `deploy.go:614` — `d.apps` map read in `stopDeployment`
- `deploy.go:630` — `hostRouter.UnregisterDeploymentHost` in `stopDeployment`
- `rollback.go:78` — `stopDeploymentWithTransition` calls `stopDeployment`
- `deploy.go:1351` — `hostRouter.RegisterDeploymentHost` in `finalizeDeploymentURL`
- `deploy.go:1061` — `hostRouter.UnregisterDeploymentHost` in `startApp` (on process exit)
- `deploy.go:1655` — `apps` map read in `handleDeployByID`

## Affected Stories
- **e45s01**: Graceful connection draining (this story) — Primary
- **e45s02**: Proxy integration — smooth traffic migration (uses the drain infrastructure built here)
- **e44s01/e44s02**: Rollback — `stopDeploymentWithTransition` may need alignment

## Test Coverage
- `state_machine_test.go:7` — `TestStateMachineValidTransitions` — will need `draining` + `stopped` states
- `state_machine_test.go:105` — `TestStateConstants` — will need new constants
- `state_lifecycle_test.go:14` — `TestStateLifecycle` — will need drain sequence case
- `deploy_test.go:2002` — orphaned process recovery (exercises stopDeployment path)
- **Gap**: No existing drain tests — new tests required:
  - `TestConnectionDrain`: drain timeout, smooth transition, force-kill fallback
  - `TestDrainLifecycle`: full lifecycle with drain phase

## Risk: Medium (Score: 7/10)
- Fan-in moderate: `stopPreviousDeployments` has 1 caller, but drain affects `apps` map (7+ access points) and `hostRouter` (5+ call sites)
- Core lifecycle change: deferring old deployment shutdown is a behavioral shift in the Trigger flow
- Rollback uses `stopDeploymentWithTransition` which bypasses the drain — must verify compatibility
- Existing state machine tests provide a safety net

## Recommended action
Proceed. Add dedicated drain tests before modifying production code (TDD). Verify rollback path still works.
