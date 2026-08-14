## ctxo MCP Tool Usage

Use ctxo MCP tools before source changes when the index is available and fresh. If ctxo is unavailable or stale, continue with LSP and repository search; do not block development on an external index.

### Before Code Modification (when ctxo is available)
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

### Guidelines
- For high-risk symbols, call `get_blast_radius` before editing when ctxo is available.
- Check `get_why_context` for high-risk or historically unstable symbols.
- Prefer `search_symbols` for indexed symbol lookup, but use LSP or grep when the index is stale or the target is plain text/configuration.
- Use `find_importers` for dependency tracing when available; verify critical callsites with LSP or repository search.

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
