# Key Rules

source: AGENTS.md
references: [AGENTS.md]

### Key Rules
- `--terminal` is optional for most commands — omitted means the active terminal in the current worktree.
- **Run `terminal read` before `terminal send`** unless the next input is obvious.
- **Cursor-based paging for long output:** after the initial tail preview, page from `oldestCursor`, then keep advancing with `nextCursor` while `limited` is true and `nextCursor !== latestCursor`.
- Terminal handles are runtime-scoped — if Orca restarts or returns `terminal_handle_stale`, reacquire with `terminal list`.
- **Base64 encoding for scripts:** When sending multi-line scripts to Orca terminals via `--text`, `$VAR`, `$(…)` and `%` get interpreted by the local shell. Encode locally with `base64`, decode remotely: `echo '<b64>' | base64 -d > script.sh`.
