# Slice 1: CLI — "BigBase Breathes"

**type:** epic  
**status:** done  
**verify:** `go run . version` → `bigbase version v0.1.0`

## Purpose

Single binary boots, discovers components, and responds to CLI commands. Proves the ECC kernel works end-to-end.

## Commands

| Command | Action | Verify |
|---------|--------|--------|
| `bigbase version` | Print version | `go run . version` |
| `bigbase status` | Show kernel + component table | `go run . status` |
| `bigbase components list` | List registered components | `go run . components list` |
| `bigbase serve [--port] [--db]` | Start HTTP server | `go run . serve --port 9999` |

## Implementation

### kernel/

- `component.go` — `Component` interface (`Name`, `Version`, `Dependencies`, `ConfigSchema`, `Init`, `Start`, `Stop`, `Hooks`), `Context`, `Event`, `HookDef`
- `eventbus.go` — `Subscribe`, `Emit` (sorted by priority), `SubscriberCount`
- `kernel.go` — `Register`, `Start` (topo-sort resolve), `Stop` (reverse order), `ListComponents`

### main.go

- Flag parsing for `serve` subcommand (`--port`, `--db`)
- CLI routing: `version`, `status`, `components list`, `help`, `serve`
- `printTable` helper using `text/tabwriter`

## Dependencies

None. Kernel has zero external dependencies.

## Files

```
kernel/
├── component.go
├── eventbus.go
├── kernel.go
└── kernel_test.go
main.go
```
