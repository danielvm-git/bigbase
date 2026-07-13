# e81 Guardrails — Anti-Patterns from 82 Prior Bug Fixes

> Auto-generated from specs/bugs/registry.yaml analysis.
> Apply these checks BEFORE writing e81 code to avoid known failure patterns.

## A. Command Construction (11 prior bugs)

| Rule | Bug Source |
|------|-----------|
| cmd.Dir MUST be set on EVERY code path returning *exec.Cmd | BUG-004 |
| Never literal $PORT in command args — pass int, format with fmt.Sprintf | BUG-004 |
| exec.LookPath BEFORE building for every runtime binary | BUG-2026-07-12T171000 |

## B. Security Hardening (14 prior bugs)

| Rule | Bug Source |
|------|-----------|
| -- separator before user-controlled args in all exec.Command calls | BUG-2026-07-11T032535 |
| filepath.Clean + containment check on filesystem paths from user input | BUG-2026-07-11T032549 |
| Generic error messages in MCP/API responses — never raw Go errors | BUG-2026-07-10T160005 |
| errors.Is/errors.As for sentinel checks — never switch on err.Error() | BUG-2026-07-10T160105 |

## C. Type Safety & Extensibility (5 prior bugs)

| Rule | Bug Source |
|------|-----------|
| Update AllAppTypes() when adding any AppType constant | BUG-2026-07-10T160201 |
| Use FlagOrEnv helpers in config — never manual os.Getenv | BUG-2026-06-27T131500 |
| Pre-compile regex at init() — never regexp.Compile per-request | BUG-2026-07-10T160107 |

## D. Testing & Observability (6 prior bugs)

| Rule | Bug Source |
|------|-----------|
| Write regression tests for ALL 6 e81s06 gaps before story done | e81s06 |
| Test with actual VPS runtimes via exec.LookPath guards in tests | BUG-2026-07-12T171000 |
| Verify orphaned process cleanup — persist PID in DB, kill on stop | BUG-2026-06-21T120000 |

## E. Infrastructure (4 prior bugs)

| Rule | Bug Source |
|------|-----------|
| Verify bigbase user can execute new runtimes after install | e81s00 |
| Restart BigBase service so exec.LookPath picks up new binaries | e81s00 |
| Add ensurePIDColumn migration for new process types | BUG-2026-06-21T120000 |
