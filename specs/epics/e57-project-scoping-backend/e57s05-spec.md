### Story e57s05: Deepen: EnvResolver seam for projects — Implementation Steps

**type:** refactor
**context:** domain
**Context**: Issue #41 requires a central `EnvResolver` seam to handle environment variable resolution so that we can support project-level secrets (e61) and preview environments (e65) without duplicating logic across the build path and start path. The resolver owns: layering, conflict precedence, and a `RedactedView()` for log consumption. This story creates the interface in `components/deploy/env.go` and wires it into the `Deploy` component.
**Reason for Depth**: `EnvResolver` centralizes precedence rules and redaction logic so that build and start paths don't duplicate secret-handling logic, eliminating a major source of accidental leakage in logs.

## Steps

1. Define `EnvResolver` interface in `components/deploy/env.go` with methods `BuildEnv(home string) []string`, `RuntimeEnv() []string`, and `RedactedView(env []string) []string`. Create `DefaultEnvResolver` that implements this using existing `BuildEnv` logic and a basic redaction check (masking keys containing `SECRET`, `KEY`, `TOKEN`, `PASSWORD`). Tag package dependencies as `[OK]`. → verify: `go build ./components/deploy/...`
2. Update `Options` in `components/deploy/deploy.go` to accept an `EnvResolver`. Update `New()` to use `DefaultEnvResolver` if nil, and store it on the `Deploy` struct. Update `Deploy.buildCmdEnv()` to delegate to the resolver. → verify: `go build -o bigbase .`
3. Write `TestEnvResolver` in `components/deploy/env_test.go` to assert that `DefaultEnvResolver` correctly filters `HOME` and `NPM_CONFIG_CACHE` and successfully redacts sensitive values in `RedactedView`. → verify: `go test -v -run TestEnvResolver ./components/deploy`

## Verification Script (Step-by-Step)

1. Run `go test ./components/deploy/...` to ensure `DefaultEnvResolver` works and old logic is preserved.
2. Run `go build -o bigbase .` to verify that the kernel and deploy component compile without errors.

## Out of scope

- Implementing actual project-level secrets database fetching (e61).
- Implementing preview-level environment overrides (e65).

## Risks

- Unintentional leakage if the redaction logic misses a key pattern. Mitigated by explicit `TestEnvResolver` tests covering variations of sensitive names.
