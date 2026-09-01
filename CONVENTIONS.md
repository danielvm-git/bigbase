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
- For goroutines and supervisors, synchronize with a channel or poll helper before asserting state; never rely on a timing sleep alone.

### Async tests — poll, never assert on timing
After launching a goroutine, supervisor, or anything that changes state
asynchronously, **poll for the expected state before asserting** — never assert
once and hope the timing lined up. A bare `go func()` followed by
`require.Equal` is the recurring flake class in this repo (22 race/flake fixes
in history).

- Use a `waitForX` helper (or `require.Eventually` / a channel sync) that retries
  with a short interval up to a timeout, then assert.
- Canonical example: `waitForHostRegistration` in the deploy component tests
  (commit `336614804`) — copy that shape rather than sprinkling `time.Sleep`.
- Prove it: `go test -race -count=20 ./...` must run without flakes.

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

### Production Hardening (Contabo VPS)
The production VPS at Contabo (vmi3338033, 89.116.26.187) follows a three-layer hardening model. See `.agents/skills/harden-vps/SKILL.md` for the full guide and `.agents/skills/harden-vps/REFERENCE.md` for script templates.

**Layer 1 — Ubuntu OS:** UFW (22,80,443 only), fail2ban SSH jail (maxretry=3, bantime=1h), unattended-upgrades (security-only, 4AM reboot), SSH (no root, no password).

**Layer 2 — BigBase:** systemd unit runs as `bigbase` user with NoNewPrivileges, ProtectSystem=full, ProtectKernelTunables, RestrictAddressFamilies, LimitNOFILE=65536. Monitoring alerts (disk>80%, CPU>90%, RAM>85%) via SQLite. Daily SQLite backup at 2AM with 90-day rotation.

**Layer 3 — Contabo VPS:** Health check script every 5min (disk, RAM, BigBase alive, reboot flag). cntb CLI for API access. Monthly snapshot on the 1st at 4AM (reads credentials from `/opt/bigbase/.env` — same file used by BigBase systemd, deployed by GitHub Actions). See `.agents/skills/harden-vps/` for the full guide.

**Verification:** Run the 8-gate check in `harden-vps` SKILL.md after any VPS or configuration change.

## Git & Workflow

### Solo-git Mode
- The project follows a solo-git workflow pattern.
- Work is done in short-lived feature branches or worktrees.
- Direct pushes or fast-forward merges to `main` when CI passes. No complex PR reviews.

### Rebase before push
- Always `git pull --rebase --autostash` before pushing to `main`. The
  big-release bot commits to `origin/main` between your fetch and push
  (3 releases/day at peak), which causes non-fast-forward rejections.
- If the release bot beat you, **rebase** onto the new `main` and push again.
- **Never** use `--force`/`--force-with-lease` to win a race against the bot —
  that discards its release commit. Rebase, don't force.

### Conventional Commits
- All commits must follow the Conventional Commits format:
  - `feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`, etc.
  - Ex: `feat(auth): add project scoping to JWT`
- Breaking changes must include a `BREAKING CHANGE:` footer.
- The system uses [`big-release`](https://github.com/danielvm-git/big-release) to automate versioning based on these commit prefixes (config in `.big-release.yml`).
