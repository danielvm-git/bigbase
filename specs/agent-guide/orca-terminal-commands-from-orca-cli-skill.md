# Orca Terminal Commands (from orca-cli skill)

source: AGENTS.md
references: [AGENTS.md]

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
