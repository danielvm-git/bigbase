# Refactor Plan: Pass All Grimoire Checks

## Problem Statement

The BigBase repository is being monitored by Grimoire (a self-hostable GitHub repository monitoring dashboard). Currently, 4 out of 9 checks are failing:

1. **CI/CD Pipeline Audit** (error severity) — Fails because the CI/CD workflow structure doesn't follow the solo-dev pipeline pattern expected by the audit script.
2. **CI/CD Migration Status** (warning severity) — Related to workflow naming/structure gaps.
3. **License Exists** (warning severity) — The LICENSE file is missing despite README claiming MIT license.
4. **Lint Passes** (warning severity) — Lint issues exist in the Go source code.

The goal is to refactor the codebase to pass all Grimoire checks while maintaining code quality and test coverage.

## Solution

Address each failing check systematically:

1. **Create LICENSE file** with standard MIT license text.
2. **Fix lint issues** identified by golangci-lint.
3. **Restructure CI/CD workflows** to pass the CI/CD Pipeline Audit check — add missing `needs` chains, concurrency blocks, permissions blocks, artifact reuse, and post-deploy verification.
4. **Run full test suite** and verify all Grimoire checks pass.

## Commits

1. Add MIT LICENSE file → verify: `test -f LICENSE && head -3 LICENSE`
2. Fix lint issues in Go source files → verify: `golangci-lint run ./...`
3. Add permissions block to all workflow files → verify: `grep -l 'permissions:' .github/workflows/*.yml`
4. Add concurrency block with cancel-in-progress to CI/CD workflow → verify: `grep -A2 'concurrency:' .github/workflows/ci-cd.yml`
5. Add explicit `needs` dependencies between CI/CD jobs (lint → test → build) → verify: `grep -A1 'needs:' .github/workflows/ci-cd.yml`
6. Add post-deploy health check step to release-deploy workflow → verify: `grep -i 'health\|smoke\|verify' .github/workflows/release-deploy.yml`
7. Pin unpinned third-party actions to SHA or version → verify: `grep 'uses:' .github/workflows/*.yml | grep -v '@v[0-9]'`
8. Run full test suite → verify: `go test ./...`
9. Run CI/CD Pipeline Audit script locally → verify: `python3 data/checks/ci-cd-pipeline-audit.yaml 2>&1 || true`

## Decision Document

### Modules to Modify
- Root directory: Add LICENSE file
- Go source files: Fix lint issues
- `.github/workflows/ci-cd.yml`: Add permissions, concurrency, job dependencies, caching
- `.github/workflows/release-deploy.yml`: Add health check, permissions, rollback
- `.github/workflows/codeql.yml`: Add permissions block
- `.github/workflows/pr-review.yaml`: Add permissions block

### Interfaces to Modify
- No Go interface changes required — this is a configuration, documentation, and workflow refactor.

### Technical Clarifications
- The CI/CD Pipeline Audit check (`data/checks/ci-cd-pipeline-audit.yaml`) is a Python script that parses workflow YAML files and checks for specific patterns. It expects:
  - Sequential gates: lint → test → build → release → deploy
  - `concurrency` with `cancel-in-progress: true` on CI/CD pipeline
  - Deploy concurrency must NOT have `cancel-in-progress: true`
  - `permissions` block on all workflows
  - Third-party actions pinned to SHA or version tag
  - Dependency caching (`actions/cache`, `cache: true`, etc.)
  - Post-deploy health/smoke check
  - Rollback path in deploy workflow
  - Build artifact upload/download reuse
  - No hardcoded secrets

### Architectural Decisions
- Keep existing workflow separation (`ci-cd.yml` for CI, `release-deploy.yml` for release/deploy)
- Add missing workflow elements to pass audit checks without changing the fundamental pipeline design
- Maintain backward compatibility with existing CI/CD processes
- Treat lint issues as errors (per user preference)

### Schema Changes
- None

## Testing Decisions

### What Makes a Good Test
- Test external behavior, not implementation details
- Verify that changes don't break existing functionality
- Ensure CI/CD workflows execute correctly after restructuring

### Modules to Test
- Go source files after lint fixes
- CI/CD workflow YAML syntax validation
- LICENSE file presence and content

### Prior Art for Tests
- Existing test suite in `*_test.go` files
- GitHub Actions workflow validation via `actionlint` or similar
- golangci-lint configuration in `.golangci.yml`

## Out of Scope

- Major architectural changes to the CI/CD pipeline design
- Adding new CI/CD stages not currently present (e.g., staging environment)
- Changing the deployment mechanism (SSH to Contabo VPS)
- Modifying test infrastructure or test frameworks
- Adding new Grimoire check definitions

## Further Notes

### CI/CD Pipeline Audit Requirements

The audit script (`ci-cd-pipeline-audit.yaml`) checks the following categories. Items marked with status need attention:

| Category | Check | Status |
|----------|-------|--------|
| File Structure | Pipeline file naming | PASS |
| File Structure | Separate deploy workflow | PASS |
| File Structure | No staging leftovers | PASS |
| File Structure | Kebab-case filenames | PASS |
| Triggers | Push trigger on all workflows | NEEDS CHECK |
| Triggers | Pull request trigger on CI/CD | NEEDS CHECK |
| Triggers | Concurrency + cancel-in-progress: true | NEEDS CHECK |
| Triggers | Deploy cancel-in-progress: false | NEEDS CHECK |
| Job Sequencing | Jobs use explicit needs | NEEDS CHECK |
| Job Sequencing | test needs lint | NEEDS CHECK |
| Job Sequencing | build needs test | NEEDS CHECK |
| Job Sequencing | release needs build | NEEDS CHECK |
| Job Sequencing | No continue-on-error on gates | NEEDS CHECK |
| Job Sequencing | Deploy chained to pipeline | NEEDS CHECK |
| Build Artifact | Artifact uploaded | NEEDS CHECK |
| Build Artifact | Version/SHA traceable | NEEDS CHECK |
| Deploy Stage | Deploy verifies artifact | NEEDS CHECK |
| Deploy Stage | environment: production | NEEDS CHECK |
| Deploy Stage | Uses GitHub secrets | NEEDS CHECK |
| Deploy Stage | No hardcoded secrets | PASS |
| Deploy Stage | Rollback path exists | NEEDS CHECK |
| Post-Deploy | Smoke/health check exists | NEEDS CHECK |
| Post-Deploy | Verification is separate step | NEEDS CHECK |
| Post-Deploy | Failure surfaced | NEEDS CHECK |
| Supply Chain | Actions pinned to SHA/version | NEEDS CHECK |
| Supply Chain | Dependency caching used | NEEDS CHECK |
| Permissions | Permissions block defined | NEEDS CHECK |
| Permissions | No secret echoing | PASS |

### License File

The README.md references MIT license but the LICENSE file is missing. Need to create a standard MIT LICENSE file with:
- Copyright notice: `Copyright (c) 2026 Daniel`
- Standard MIT license text

### Lint Issues

The earlier exploration found lint passes clean (`golangci-lint run ./...` shows "No issues found"). The Grimoire "Lint Passes" check may be running a different lint configuration or checking something else. Need to investigate what the Grimoire lint check actually runs. The `Lint Passes` check definition needs to be located in the Grimoire instance to understand its exact script.
