# Component Interface

source: CONVENTIONS.md
references: [CONVENTIONS.md]
enforced_by: [audit-code, plan-work, verify-work]

### Component Interface
```go
type Component interface {
    Name() string
    Version() string
    Dependencies() []string
    ConfigSchema() json.RawMessage
    Init(ctx *Context, config json.RawMessage) error
    Start(ctx *Context) error
    Stop(ctx *Context) error
    Hooks() []HookDef
}
```

Components communicate via events, not direct imports:
- `proxy` emits `onRequest` → `auth` validates, `api` routes, `monitoring` logs
- `db` emits `onMutation` → `realtime` notifies, `functions` triggers, `messaging` sends
- `git` emits `onPush` → `cici` runs workflow, `deploy` previews
