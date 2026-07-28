# Sweep Environment Capture

**Pinned commit:** `cedb0fc51cca0436c709295c79dd251906536513` (main)
**Branch:** `chore/bug-certification-sweep`
**Date:** 2026-07-27
**Working tree:** dirty at sweep start (11 entries). Uncommitted changes are EXCLUDED
from the certification baseline — all `git show HEAD:<path>` checks read the pinned commit.

## Toolchain

| Tool | Version | Path | Notes |
|---|---|---|---|
| go | 1.26.3 darwin/arm64 | system | |
| git | 2.54.0 | system | |
| node | 26.3.0 | /opt/homebrew/bin/node | |
| npm | 11.16.0 | /opt/homebrew/bin/npm | present — deploy tests should NOT skip |
| bun | 1.3.14 | /opt/homebrew/bin/bun | present |
| pnpm | present | /opt/homebrew/bin/pnpm | |
| uv | 0.11.14 | ~/.local/bin/uv | python deploy path |
| python3 | 3.13.13 | pyenv shim | |
| golangci-lint | installed | /opt/homebrew/bin/golangci-lint | |
| gosec | installed | ~/go/bin/gosec | |

**Shell caveat:** `npm`/`bun`/`pnpm` are shell-function wrappers that fail in
non-interactive shells (`_bsm_wrap: command not found`). All sweep commands must
`export PATH="/opt/homebrew/bin:$PATH"` so tests resolve the real binaries.
Without this, `components/deploy` tests silently `t.Skip("npm not in PATH")`.

## CI leg coverage

| Leg | Status | Reason |
|---|---|---|
| sqlite (`BIGBASE_DB_DRIVER=sqlite BIGBASE_DB_DSN=:memory:`) | RUN | |
| postgres (`postgres:16`) | **NOT RUN** | `pg_isready localhost:5432` → no response; docker unavailable on this host |

Per the sweep rubric, DB-sensitive scopes (auth, sites, deploy) cap at
`CERTIFIED-WEAK` with `leg_coverage: sqlite_only` because the postgres leg
could not be executed. This is recorded, not silently dropped.

## Tier-0 results

| Gate | Result |
|---|---|
| `go build ./...` | PASS |
| `go test ./... -json` (sqlite) | **1226 pass / 0 fail / 1 skip**; 27 packages pass, 2 no-test. Skip is `components/deploy TestEnsureNodePackageManager_MissingBun` (intentional negative-path test) |
| `ui` vitest under **Node 22** | **549 pass / 0 fail** — authoritative |
| `ui` vitest under Node 26.3.0 | 543 pass / 6 fail — **environment artifact, NOT regressions** |
| `packages/auth`, `packages/auth-next` vitest | **NOT RUN** — no `node_modules` installed locally |

### Node 26 false-failure finding

Running the UI suite on the local default Node (26.3.0) fails 6 tests in
`ui/src/Layout.test.tsx` and `ui/src/context/ThemeContext.test.tsx` with
`TypeError: Cannot read properties of undefined (reading 'getItem'/'clear')` —
`window.localStorage` is undefined. Node 26 ships an experimental built-in
`localStorage` that requires `--localstorage-file` and interacts badly with the
jsdom environment.

The identical suite passes **549/549 on Node 22.22.3**, which matches CI
(`ci-cd.yml` pins `node-version: '20'`; `release-deploy.yml` uses `22`).

These 6 failures are therefore **not** code regressions and must not produce
`REGRESSED` verdicts. The scorer consumes `ui-vitest-node22.json` only.
The Node 26 incompatibility is itself a real forward-compatibility defect and is
filed as a new bug rather than being discarded.

## Denylisted test scripts (zero evidence value)

These `packages/*` test scripts are stubs and must never count as passing:

- `packages/auth-react` → `echo ok`
- `packages/auth-vue` → `echo ok`
- `packages/auth-astro` → `echo ok`
- `packages/auth-ui-svelte` → `echo ok`
- `packages/auth-svelte` → `echo 'tests pass'`

Only `packages/auth`, `packages/auth-next`, and `ui/` run real vitest suites.
