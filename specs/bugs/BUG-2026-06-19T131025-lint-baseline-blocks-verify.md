# BUG-2026-06-19T131025: Full lint baseline blocks verify-work

## Problem

During `verify-work` for story `e36s01`, the story-specific backend implementation verified successfully, but the full mechanical lint gates failed.

- **Actual behavior:** Full Go lint and UI lint return non-zero exit codes even though the active story scope passes lint and tests.
- **Expected behavior:** Repository-level verification gates should be green, or failures should identify only changes introduced by the active story.
- **How to reproduce:**
  1. Run `golangci-lint run ./...` from the repository root.
  2. Run `cd ui && npm run lint`.
  3. Observe both commands fail before audit/release can proceed.

Relevant prior bug history:

- Related to `BUG-2026-06-04T100000`, which fixed reviewer-found UI quality and lint-hardening issues in specific UI components.
- This is a broader and novel baseline issue: lint gates now fail across multiple unrelated backend and frontend modules, not just one reviewed UI slice.

## Root Cause Analysis

### Reproduce

The failure reproduces on the current feature branch with these commands:

- `golangci-lint run ./...` exits non-zero and reports unchecked errors, an ineffectual assignment, and diagnostics from a vendored frontend dependency package that Go package discovery includes.
- `cd ui && npm run lint` exits non-zero and reports a mix of unused variables, empty blocks, missing plugin-rule definitions, Fast Refresh export rules, and new React Hooks compiler-style rules across existing UI modules.

Story-scoped isolation passes:

- `golangci-lint run ./components/sites/...` passes with `0 issues`.
- The site deletion tests and full Go tests pass.

### Isolate

The issue is not caused by the new Site deletion backend behavior. It is isolated to repository-wide lint configuration and existing baseline violations:

- The Go lint command scans every Go package under the repository, including a Go package inside the frontend dependency tree.
- The repository has no root lint configuration constraining package scope or documenting intentional exclusions.
- Several existing Go modules contain unchecked return values or an ineffectual assignment that stricter linting rejects.
- The UI lint configuration applies the newest recommended React Hooks and React Refresh rules across all TypeScript/TSX files, while the existing UI codebase still contains pre-existing patterns those rules now reject.
- Some test files reference a lint rule from an import plugin that is not installed/configured.

### Hypothesize

Ranked hypotheses and falsification tests:

1. **Hypothesis:** The active Site deletion story introduced the lint failure.
   - **Falsification test:** Run lint only for the active story's backend package.
   - **Result:** Falsified. Story-scoped Go lint passes with `0 issues`.

2. **Hypothesis:** Repository-wide Go lint is failing because lint scope/configuration includes unrelated modules and vendored frontend dependency code.
   - **Falsification test:** Inspect the lint output and Go package list for non-story modules and frontend dependency packages.
   - **Result:** Confirmed. Failures occur in unrelated backend modules and a frontend dependency package discovered by Go tooling.

3. **Hypothesis:** UI lint is failing because the configured rule set is stricter than the current baseline, not because of this backend story.
   - **Falsification test:** Run lint against an existing UI page untouched by this story.
   - **Result:** Confirmed. Untouched UI pages fail under the current rules.

### Verify

Verified root cause: the project lacks a clean, maintained repository-wide lint baseline. Full lint gates are enforcing rules over unrelated existing code and dependency directories, while story-scoped code remains clean. This blocks `verify-work` despite the active story's implementation passing build, tests, and scoped lint.

Risk level: **High** — this blocks all future release/audit flows that require full mechanical gates, and it can obscure real regressions by mixing new-story failures with unrelated baseline noise.

## TDD Fix Plan

1. **RED:** Run the full Go lint gate and capture the current failing baseline as the regression target.
   **GREEN:** Add/adjust root Go lint configuration so dependency directories are excluded intentionally, then fix existing unchecked-error and ineffectual-assignment findings in first-party Go code without changing behavior.
   **verify:** `golangci-lint run ./...`

2. **RED:** Run the UI lint gate and capture the current failing baseline as the regression target.
   **GREEN:** Align the UI lint configuration with the project's adopted rules: exclude generated/build/dependency artifacts, remove stale references to unavailable lint rules, and either fix or explicitly downgrade newly introduced React compiler-style rules that the codebase has not adopted yet.
   **verify:** `cd ui && npm run lint`

3. **RED:** Run the full release verification command set after the lint fixes.
   **GREEN:** Make the smallest additional cleanup needed so build, vet/typecheck, lint, and tests pass together.
   **verify:** `go build -o /tmp/bigbase-verify . && go vet ./... && go test ./... && cd ui && npm run build && npm run lint`

**REFACTOR:** If the lint fixes reveal repeated UI data-loading patterns, extract or document one approved pattern instead of suppressing rules ad hoc across files.

## Acceptance Criteria

- [ ] `golangci-lint run ./...` passes from the repository root.
- [ ] `cd ui && npm run lint` passes.
- [ ] Frontend dependency/build directories are not treated as first-party lint targets.
- [ ] Story-scoped Site deletion tests still pass.
- [ ] Full Go tests still pass.
- [ ] Verification evidence for `e36s01` can be updated from blocked to passed.

## Resolution

<!-- filled in by validate-fix -->
