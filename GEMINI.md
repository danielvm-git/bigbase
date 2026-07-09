# BigBase — Gemini CLI

Read CONVENTIONS.md before any GitHub or git operation.

## Project
Single-binary, component-based BaaS platform using Entity-Component-Construct (ECC) architecture.
Stack: Go 1.22+ / ECC Kernel + Plugins / SQLite + PostgreSQL

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

Read `specs/state.yaml`, `specs/release-plan.yaml`, and the active epic under `specs/epics/`. Architecture: `specs/plans/TECH_STACK_LATEST.md`.

## Agent Rules
- Read specs YAML (not `specs/archive/`) before writing code.
- All planning specifications MUST be written under `specs/` before code.
- Write minimum code that solves the stated problem.
- Run tests after every change. Show evidence before declaring done.

## Model Routing Matrix (Gemini)

### 1. Model Matrix & Allocation

| Task Category | Optimal Model | Selection Reason |
| :--- | :--- | :--- |
| **Global Planning & ADRs** | `Gemini 3.1 Pro (High)` | Deep reasoning, complex design trade-offs, and architectural planning. |
| **Codebase Context Search** | `Gemini 3.1 Pro (Low)` | 2M+ context window. Best for reading large file trees and cross-component structures. |
| **Feature Coding & TDD Loops** | `Gemini 3.1 Pro (High)` | Precision coding, exact syntax execution, and deep test writing. |
| **Verification & Utility Tasks** | `Gemini 3.5 Flash (High/Medium)` | Ultra-low latency, cheap tokens. Best for running linters, tests, and compiling. |
| **Browser UI Testing** | `Gemini 3.5 Flash (Low)` | Fast image/visual processing for browser subagents. |
| **Structured Docs & Summaries** | `Gemini 3.5 Flash (Medium)` or `Gemini 3.1 Pro (Low)` | Strong at structured prose, reports, and YAML/JSON synthesis. |

### 2. Dynamic Delegation Protocol

When spawning sub-agents (via `delegate-task`, `dispatch-agents`, or `browser_subagent`):
- **For file system audits, logs inspection, and linting**: Use `Gemini 3.5 Flash (Low)` to keep token costs minimal.
- **For code generation/refactoring sub-tasks**: Use `Gemini 3.1 Pro (High)`.
- **Context Shaving**: Never pass entire files to sub-agents unless they are targets for modification. Pass only specific function signatures or YAML spec blocks to keep input tokens low.


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

<!-- ctxo-rules-start -->
## ctxo MCP Tool Usage (MANDATORY)

**ALWAYS use ctxo MCP tools before reading source files or making code changes.** The ctxo index contains dependency graphs, git intent, anti-patterns, and change health that cannot be derived from reading files alone. Skipping these tools leads to blind edits and broken dependencies.

### Before ANY Code Modification
1. Call `get_blast_radius` for the symbol you are about to change — understand what breaks
2. Call `get_why_context` for the same symbol — check for revert history or anti-patterns
3. Only then read and edit source files

### Before Starting a Task
| Task Type | REQUIRED First Call |
|---|---|
| Fixing a bug | `get_context_for_task(taskType: "fix")` |
| Adding/extending a feature | `get_context_for_task(taskType: "extend")` |
| Refactoring | `get_context_for_task(taskType: "refactor")` |
| Understanding code | `get_context_for_task(taskType: "understand")` |

### Before Reviewing a PR or Diff
- Call `get_pr_impact` — single call gives full risk assessment with co-change analysis

### When Exploring or Searching Code
- Use `search_symbols` for name/regex lookup — DO NOT grep source files for symbol discovery
- Use `get_ranked_context` for natural language queries — DO NOT manually browse directories

### Orientation in Unfamiliar Areas
- Call `get_architectural_overlay` to understand layer boundaries
- Call `get_symbol_importance` to identify critical symbols

### NEVER Do These
- NEVER edit a function without first calling `get_blast_radius` on it
- NEVER skip `get_why_context` — reverted code and anti-patterns are invisible without it
- NEVER grep source files to find symbols when `search_symbols` exists
- NEVER manually trace imports when `find_importers` gives the full reverse dependency graph
<!-- ctxo-rules-end -->
