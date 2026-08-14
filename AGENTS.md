# BigBase — Engineering Guide (canonical)

This is the single source of engineering truth for AI agents on BigBase.
`CLAUDE.md` and `GEMINI.md` are thin adapters that import this file and add only
their own model-routing overrides. Edit rules HERE, not in the adapters.

Read CONVENTIONS.md before any GitHub or git operation.

## Project
Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC) architecture.
Stack: Go 1.26+ / ECC Kernel + Plugins / SQLite + PostgreSQL

## Commands
| Action | Command |
|--------|---------|
| Run    | `go run .` |
| Test   | `go test ./...` |
| Build  | `go build -o bigbase .` |
| Lint   | `golangci-lint run ./...` |
| Test coverage | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |

## Architecture
ECC pattern: Kernel (discovery, lifecycle, event bus, config merge) + pluggable components (proxy, auth, db, api, storage, git, forge, cici, functions, realtime, messaging, deploy, admin, monitoring). Components communicate via event hooks, not direct imports.

## Conventions
- Go standard layout: `kernel/`, `components/<name>/`, `config/`
- Component interface: `Init(ctx, config) → Start(ctx) → Stop(ctx)`
- Event bus for cross-component communication (no direct imports)
- TDD workflow: write test first, see it fail, implement, verify green
- All planning output in `specs/`

## Never
- Hardcode secrets, API keys, or tokens
- Mutate state directly — use spread/immutable patterns
- Commit to main without PR
- Expose internal errors or stack traces to API clients
- Use `any` in Go — prefer concrete types or interfaces

## Specs (bigpowers YAML)

Read `specs/state.yaml`, `specs/release-plan.yaml`, and the active file under `specs/epics/` before coding. Architecture: `specs/tech-architecture/tech-stack.md` (canonical; `specs/plans/TECH_STACK_LATEST.md` is superseded). Legacy docs: `specs/archive/` only.

## Agent Rules
- Read specs YAML (not `specs/archive/`) before writing code.
- All planning specifications MUST be written under `specs/` before code.
- Write minimum code that solves the stated problem.
- Run tests after every change. Show evidence before declaring done.

## sqz — Context Compression (READ FIRST)

sqz is installed in this project. It compresses tool output so large files, long
logs, and verbose command output cost far fewer tokens. There are **two ways**
sqz is wired in. Prefer each one in the situations below.

### Preferred tools (MCP)

The `sqz-mcp` server is registered in this project's MCP config. It exposes three
read-only tools that compress their output through the sqz pipeline:

- **`sqz_read_file`** — read a file from disk and return a compressed view.
  **PREFER this over a plain file read** for any file larger than ~2KB or any
  file you might read more than once in the same session. Repeat reads return a
  13-token `§ref:HASH§` reference instead of the full content.
- **`sqz_grep`** — search files for a literal string or regex. **PREFER this over
  a plain grep** for anything that might match more than a handful of lines. Caps
  at 200 matches by default; raise with `max_matches`.
- **`sqz_list_dir`** — list a directory, skipping `.git`, `node_modules`,
  `target`, `dist`, `build`, `vendor`, `__pycache__`.

Built-in read/grep/glob tools remain available. Use them for tiny config files
(<1KB), byte-exact reads you'll hash or diff (lockfiles, signatures), and
globbing (sqz has no glob tool).

### Bash commands (hooked automatically)

A PreToolUse hook rewrites shell commands to pipe output through `sqz compress`
(e.g. `git status`, `cargo test`, `docker ps`, `kubectl get pods`). Transparent;
skipped for interactive commands (`vim`, `ssh`, `python`), compound commands
(`a && b`, `a > file`), and anything already going through sqz.

### Escape hatch — when you see a `§ref:HASH§` token

To resolve a `§ref:a1b2c3d4§` token to full content: `sqz expand a1b2c3d4` (or
paste the whole token), or call the `expand` MCP tool with `{ "prefix": "a1b2c3d4" }`,
or prefix one command with `SQZ_NO_DEDUP=1`. If compressed output is making the
task harder, call the `passthrough` MCP tool for raw text.

### When NOT to use sqz tools
- Writing or editing files — use the built-in Write/Edit tools (sqz has no write tools).
- Running commands interactively or in watch mode.
- Reading very small files (<1KB) where compression can't help.

# RTK (Rust Token Killer) — Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it;
if not, it passes through unchanged — so `rtk` is always safe. This holds inside
`&&` chains too: `rtk git add . && rtk git commit -m "msg" && rtk git push`.

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build / check / clippy    # Rust build/lint output, grouped
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (60-99% savings)
```bash
rtk go test / cargo test / pytest / jest / vitest / playwright test / rspec
rtk test <cmd>          # Generic test wrapper — failures only
```

### Git (59-80% savings)
```bash
rtk git status / log / diff / show / add / commit / push / pull / branch / stash / worktree
```
Git passthrough works for ALL subcommands, even those not listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num> / gh pr checks / gh run list / gh issue list / gh api
```

### JS/TS Tooling (70-90%)
```bash
rtk pnpm list / outdated / install   ;   rtk npm run <script>   ;   rtk npx <cmd>   ;   rtk prisma
```

### Files & Search (60-75%)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%). Format flags (-c,-l,-L,-o,-Z) run raw.
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90%)
```bash
rtk err <cmd> / log <file> / json <file> / deps / env / summary <cmd> / diff
```

### Infra & Network (65-85%)
```bash
rtk docker ps / images / logs <c>    ;    rtk kubectl get / logs    ;    rtk curl <url> / wget <url>
```

### Meta
```bash
rtk gain [--history]    # token savings stats
rtk proxy <cmd>         # run without filtering (debug)
rtk discover            # find missed RTK usage in sessions
```

Overall: **60-90% token reduction** on common development operations.

## bts toolchain

`bts` is installed. Prefer its verbs over ad-hoc shell commands.

| Task | Command | Avoid |
|------|---------|-------|
| Search code | `bts find --print <pattern>` | grep / find / cat |
| Interactive search | `bts find <pattern>` | manual grep pipes |
| Compress for context | `bts compress <file>` or `cmd \| bts compress` | summarising by hand |
| Repo map | `bts map` | listing files by hand |
| Library docs | `bts docs <lib>` | guessing from training data |
| Package source | `bts src <pkg>` | git clone |
| Toolchain health | `bts doctor` | which / command -v |

**Rules**
- Search with `bts find` before opening files to locate a symbol or pattern.
- Pipe anything > 200 lines through `bts compress` before adding to context.
- Run `bts map` when asked for a repo overview.
- Use `bts docs <lib>` before answering questions about library APIs.
- If a tool is missing, say so and run `bts doctor` — do not silently substitute.

## ctxo — dependency-graph oracle (preferred when the index is fresh)

ctxo is a **dependency-graph and change-intelligence oracle, not a search
engine.** Its unique value is data you cannot get by reading files: reverse
dependencies, blast radius, revert/anti-pattern history, and PR risk. For
finding a symbol or a string, `grep`/`rg` and LSP are always correct. Reach for
ctxo when you need graph or history context, and prefer it *when the local
`.ctxo/` index is warm and fresh* (the git hooks below keep it that way). A
wrong answer from a stale index is worse than a slower correct one — when in
doubt, grep.

### Where ctxo earns its keep
- `get_blast_radius(symbol)` — what breaks if you change a symbol. Worth running
  before a non-trivial edit to a shared symbol.
- `get_why_context(symbol)` — revert history and known anti-patterns. Reverted
  code and past mistakes are invisible from the current tree alone.
- `get_pr_impact` / `get_context_for_task(taskType: fix|extend|refactor|understand)`
  — risk assessment with co-change analysis for a diff or a task.
- `find_importers(symbol)` — the reverse dependency graph, instead of tracing
  imports by hand.
- `get_architectural_overlay` / `get_symbol_importance` — orientation in an
  unfamiliar area.

### Searching with ctxo
- `search_symbols` takes a **`pattern`** parameter (substring or regex), with
  optional `kind` and `filePattern` — it is NOT a `query` parameter.
- `get_ranked_context` answers natural-language queries. Both are conveniences,
  not replacements: `grep`/`rg`/LSP remain correct and are the right call when
  the index may be cold or stale.

### Index freshness & known gaps
- The `.ctxo/` index is kept fresh by **versioned git hooks**:
  `.githooks/pre-commit` full-indexes staged source; `.githooks/post-commit`
  incrementally re-indexes changed files (`ctxo index --file`); and
  `.githooks/post-merge` runs `ctxo sync` after a pull. Run `./scripts/setup.sh`
  on a fresh clone to wire `core.hooksPath=.githooks`.
- **`verify-index` is upstream-broken here — skip it.** Its temporary comparison
  build runs with default config and can never match this repo's configured
  index (see `.ctxo/config.yaml`).
- A committed baseline index is intentionally NOT kept: every index JSON embeds
  a `lastModified` timestamp, so committing one would dirty all files on each
  rebuild. Freshness is hook-managed, per-clone.

## Orca Terminal Commands (from orca-cli skill)

When interacting with Orca-managed terminals:

```bash
orca terminal list --worktree active --json       # list live terminals
orca terminal show --terminal <handle> --json      # inspect metadata + preview
orca terminal read --terminal <handle> --json      # read current output (tail)
orca terminal read --terminal <handle> --cursor <cursor> --limit 1000 --json
orca terminal send --terminal <handle> --text "..." --enter --json
orca terminal wait --terminal <handle> --for exit --timeout-ms 5000 --json
orca terminal wait --terminal <handle> --for tui-idle --timeout-ms 300000 --json
```

### Key Rules
- `--terminal` is optional for most commands — omitted means the active terminal in the current worktree.
- **Run `terminal read` before `terminal send`** unless the next input is obvious.
- **Cursor-based paging for long output:** after the initial tail preview, page from `oldestCursor`, then keep advancing with `nextCursor` while `limited` is true and `nextCursor !== latestCursor`.
- Terminal handles are runtime-scoped — if Orca restarts or returns `terminal_handle_stale`, reacquire with `terminal list`.
- **Base64 encoding for scripts:** When sending multi-line scripts to Orca terminals via `--text`, `$VAR`, `$(…)` and `%` get interpreted by the local shell. Encode locally with `base64`, decode remotely: `echo '<b64>' | base64 -d > script.sh`.

## harden-vps — Production VPS Security (LOAD BEFORE VPS work)

**ALWAYS load `.agents/skills/harden-vps/SKILL.md` before SSHing into the production VPS.** The skill contains the three-layer hardening model, gotchas (crontab % escaping, fail2ban missing-log crash, BigBase alerts require SQLite insert), and the 8-gate verification matrix.

### VPS Quick Reference
- **IP:** 89.116.26.187 (vmi3338033)
- **User:** root (SSH key required)
- **Contabo Customer ID:** 15027696
- **BigBase service:** `systemctl status bigbase`
- **Health check:** `/opt/bigbase/scripts/healthcheck.sh`
- **Backups:** `/backup/bigbase-YYYYMMDD.db` (2AM daily, 90-day rotation)
- **Contabo creds:** `/opt/bigbase/.env` (CONTABO_CLIENT_ID, CONTABO_CLIENT_SECRET, CONTABO_API_USER, CONTABO_API_PASSWORD — deployed via GitHub Actions). Local dev: add to `.envrc`.

### VPS Verification (8 gates)
```bash
ufw status|grep -q active||echo FAIL:ufw
fail2ban-client status sshd>/dev/null 2>&1||echo FAIL:fail2ban
systemctl is-active unattended-upgrades|grep -q active||echo FAIL:unattended
sshd -T|grep -q 'permitrootlogin no'||echo FAIL:sshd
systemctl show bigbase -p NoNewPrivileges|grep -q yes||echo FAIL:systemd
systemctl is-active bigbase|grep -q active||echo FAIL:bigbase
sqlite3 /opt/bigbase/data/bigbase.db "SELECT count(*) FROM monitoring_alerts"|grep -q 3||echo FAIL:alerts
crontab -l|grep -q healthcheck&&crontab -l|grep -q bigbase.db&&crontab -l|grep -q contabo-snapshot||echo FAIL:crontab
echo ALL 8 GATES PASSED
```

## opensrc — Dependency Source Code (READ BEFORE reaching for other tools)

`opensrc` fetches and caches the actual source code of any dependency from npm, PyPI, crates.io, or GitHub/GitLab/Bitbucket at `~/.opensrc/`. It resolves registry metadata → shallow-clones at the correct version tag → gives you a local filesystem path. **Whenever your question is about what a library DOES internally, opensrc is the right tool.**

### Tool Selection: opensrc vs EVERYTHING else

| Your intent | RIGHT tool | WRONG tool (and why) |
|-------------|-----------|----------------------|
| How does `zod.parse()` handle this edge case? | `opensrc path zod` + `rg` | `web_search` — blog posts and docs won't show the actual code path |
| What does React's `useEffect` actually schedule internally? | `opensrc path react` + `rg` | `fetch_content` of react.dev — docs describe behavior, not implementation |
| Why does Prisma generate this SQL for my query? | `opensrc path prisma` + `rg` | `context7` MCP — gives API signatures, not query-generation internals |
| What does serde's `Deserialize` derive macro expand to? | `opensrc path crates:serde` + `rg` | `bts_find` — only searches our codebase, not the dependency |
| How does Express route matching work under the hood? | `opensrc path express` + `rg` | `read` — only reads files in our project, not node_modules |
| Study how Tailscale's DERP relay handles NAT traversal | `opensrc path tailscale/tailscale` + `rg` | `bts_map` — maps our repo structure, not theirs |
| Clone a repo to read its source locally | `opensrc path owner/repo` | `bash` + `git clone` — opensrc is shallow, cached, version-aware, and 10x faster |
| What does Flask's `@app.route` decorator actually register? | `opensrc path pypi:flask` + `rg` | `fetch_content` of flask.palletsprojects.com — docs show usage, not internals |
| Find how a Go library implements an interface | `opensrc path <pkg>` + `rg` | `web_search` for godoc — godoc shows signatures, not implementation |
| Check a dependency's package.json scripts or metadata | `cat $(opensrc path <pkg>)/package.json` | `read` on `node_modules/<pkg>/package.json` — opensrc gives the full repo context |
| Read a library's CHANGELOG to understand a breaking change | `fetch_content <repo-url>/blob/main/CHANGELOG.md` | `opensrc` — fetching a whole clone for one markdown file is overkill |
| Find API usage examples or community patterns | `web_search` or `fetch_content` | `opensrc` — source won't show how real users compose the API |
| Check if a library is actively maintained | `web_search` or `fetch_content <repo-url>` | `opensrc` — you want GitHub activity, not source files |
| Get the official docs for a function signature | `context7` MCP or `web_search` | `opensrc` — docs are faster than source for signatures |

### Quick Reference

```bash
# Core pattern: opensrc path gives a filesystem path, compose with any tool
rg "pattern"     $(opensrc path zod)           # search source
cat               $(opensrc path zod)/src/types.ts  # read a file
find              $(opensrc path zod) -name "*.test.ts"  # explore
ls                $(opensrc path zod)/src/      # list directory

# Any registry — same pattern
rg "dispatch"     $(opensrc path pypi:requests)
cat               $(opensrc path crates:serde)/src/lib.rs
grep -r "Router"  $(opensrc path vercel/next.js)/packages/next/src/

# Specific versions (matches YOUR lockfile version automatically for npm)
rg "ZodError"     $(opensrc path zod@3.22.0)
cat               $(opensrc path pypi:flask@3.0.0)/src/flask/app.py

# Pre-fetch to warm the cache (non-blocking, no path printed)
opensrc fetch zod react prisma @trpc/server
opensrc fetch pypi:requests pypi:fastapi
opensrc fetch crates:serde crates:tokio

# Cache management
opensrc list                # what's cached
opensrc list --json         # machine-readable
opensrc remove zod          # evict one package
opensrc clean --npm         # wipe npm cache
opensrc clean --pypi        # wipe PyPI cache
opensrc clean               # wipe everything
```

### Registry Map

| Registry | Spec | Example |
|----------|------|---------|
| npm | `<name>` or `npm:<name>` | `opensrc path zod` |
| npm (scoped) | `@scope/name` | `opensrc path @trpc/server` |
| PyPI | `pypi:<name>` / `pip:` / `python:` | `opensrc path pypi:requests` |
| crates.io | `crates:<name>` / `cargo:` / `rust:` | `opensrc path crates:serde` |
| GitHub | `owner/repo` or full URL | `opensrc path vercel/next.js` |
| GitLab | `gitlab:owner/repo` or URL | `opensrc path gitlab:gnome/gtk` |
| Bitbucket | `bitbucket:owner/repo` or URL | `opensrc path bitbucket:atlassian/python-bitbucket` |

### How It Works

1. Resolves package → git repo URL via registry APIs (npm/PyPI/crates.io/GitHub API)
2. For npm: auto-detects installed version from lockfiles (package-lock, pnpm-lock, yarn.lock)
3. Shallow-clones at `v<version>` tag (falls back to `<version>`, then default branch)
4. Caches at `~/.opensrc/repos/<host>/<owner>/<repo>/<version>/`
5. Records in `~/.opensrc/sources.json` (atomic writes, corrupt-safe)

Env vars: `OPENSRC_HOME` (cache location), `GITHUB_TOKEN` / `GITLAB_TOKEN` / `BITBUCKET_TOKEN` (private repos).

### The Rule

**opensrc is for reading code, not docs.** When your question is "how does X work internally?" or "what does this function actually do?", reach for opensrc first. The source is the ground truth — types, docs, and blog posts are approximations. Only fall back to `web_search`, `fetch_content`, or `context7` when you need documentation, usage examples, community patterns, or repo metadata that isn't in the source.
