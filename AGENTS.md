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
