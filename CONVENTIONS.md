# BigBase Conventions

## Architecture

### Entity-Component-Construct (ECC)
- **Entity** = The running BigBase server
- **Component** = Independent submodule with its own lifecycle (auth, db, proxy...)
- **Construct** = The config that decides which components run together

### Kernel Responsibilities
- Component discovery and registration
- Dependency resolution
- Lifecycle management: Init → Start → Stop
- Event bus for hook-based communication
- Config merge: defaults + user overrides

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

## Code Quality

### Readability First
- Clear variable/function names in English
- Self-documenting code over comments

### KISS
- Simplest solution that works
- No premature optimization

### DRY
- Extract shared logic
- No copy-paste

### YAGNI
- Don't build what isn't needed yet

## Go Conventions

### Naming
- `camelCase` for unexported, `PascalCase` for exported
- Acronyms all-caps: `HTTP`, `URL`, `API`, `JSON`
- File names: `snake_case.go`

### Error Handling
```go
// Always check errors
if err != nil {
    return fmt.Errorf("context: %w", err)
}

// Use sentinel errors for expected failures
var ErrNotFound = errors.New("not found")
```

### Immutability
- Prefer value receivers for small structs
- Return copies, don't mutate inputs

### Testing
- Table-driven tests with `t.Run`
- Package name: `package_test`
- Test files: `*_test.go` co-located with source

## Project Structure
```
bigbase/
├── main.go
├── kernel/
│   ├── kernel.go        — discovery, lifecycle
│   ├── component.go     — Component interface
│   ├── eventbus.go      — hook system
│   ├── config.go        — config merge
│   └── registry.go      — component registration
├── components/
│   ├── proxy/
│   ├── auth/
│   ├── db/
│   ├── api/
│   ├── storage/
│   ├── git/
│   ├── forge/
│   ├── cici/
│   ├── functions/
│   ├── realtime/
│   ├── messaging/
│   ├── deploy/
│   ├── admin/
│   └── monitoring/
├── config/
│   ├── defaults.jsonc
│   └── profiles/
├── specs/               — planning documents
│   ├── adr/             — architecture decision records
│   ├── CONTEXT.md       — domain context
│   ├── RELEASE-PLAN.md  — epics and stories
│   └── ...
└── ui/                  — Admin UI (React)
```

## Specs Convention
All planning and specifications live in `specs/`. Before writing any code, read `specs/` and write a plan to `specs/PLAN.md`.

## Defensive Code Categories
- **Rate limit** — all API endpoints
- **Retry with backoff** — external calls (DB, OAuth, S3)
- **Timeout** — all network operations
- **Graceful degradation** — if Redis/dependency is down, fall back

## Security
- No secrets in code. Use env vars.
- `httpOnly` + `Secure` + `SameSite=Strict` cookies
- Parameterized queries only (no SQL concatenation)
- Validate all input with schemas
- Generic error messages to clients, full details in logs

## Git & Workflow

### Solo-git Mode
- The project follows a solo-git workflow pattern.
- Work is done in short-lived feature branches or worktrees.
- Direct pushes or fast-forward merges to `main` when CI passes. No complex PR reviews.

### Conventional Commits
- All commits must follow the Conventional Commits format:
  - `feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`, etc.
  - Ex: `feat(auth): add project scoping to JWT`
- Breaking changes must include a `BREAKING CHANGE:` footer.
- The system uses `semantic-release` to automate versioning based on these commit prefixes.
