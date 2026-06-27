# e50s01: Persist JWT secret — env var with fallback
## Story ID: e50s01 | Epic: e50 | BCPs: 2 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
The JWT signing secret is currently auto-generated on every startup via `crypto/rand` when `auth.Options.Secret` is empty. This means tokens are invalidated on every restart. Production deployments need a persistent secret that survives restarts. The story adds `BIGBASE_JWT_SECRET` env var support with fallback to auto-generation.

## 3. Problem / Opportunity
- Restarting BigBase invalidates all existing JWTs (no persistent secret)
- No way to share a secret across multiple instances (future HA)
- Security: auto-generation is silent — operators may not realize tokens are ephemeral

## 4. Proposed solution
Add `resolveJWTSecret(logger Logger) []byte`:
- Reads `BIGBASE_JWT_SECRET` env var
- Validates min 32 bytes (HS256 requirement + entropy floor)
- Rejects known test/debug defaults (`test-secret-32-chars!!!` etc.)
- Logs "JWT secret loaded from configuration" on success
- Falls back to `crypto/rand` auto-generation with a warning log

## 5. Alternatives considered
- **CLI flag `--jwt-secret`**: Adds secret to process list (visible in `ps`). Env var is slightly less exposed and consistent with existing patterns (`GOOGLE_CLIENT_ID`, `RATE_LIMIT_*`).
- **File-based secret (e.g., `/etc/bigbase/jwt.secret`)**: Overcomplicated for single-binary. Env var is idiomatic for Go CLI tools.

## 6. Who are the users?
- **Operators** deploying BigBase in production
- **Developers** running locally who want persistent tokens across restarts

## 7. Dependencies
- `os.Getenv` (stdlib)
- `crypto/rand` (stdlib, already used)
- `encoding/hex` (stdlib, already used)
- Existing `auth.New()` and `auth.Options` struct

## 8. Assumptions
- 32 bytes minimum is sufficient entropy for HS256
- Env var pattern is acceptable (not a Kubernetes Secret for now)
- No live reload of secret (requires restart)

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Operator sets weak secret (< 32 bytes) | Medium | Low | Validate + reject with clear error |
| Operator uses known default | Low | High | Blacklist known test values |
| Secret leaks via env dump | Low | Medium | Document rotation procedure |

## 10. Non-goals
- Secret rotation (out of scope — requires token versioning)
- Multiple active secrets (HA multi-instance — out of scope)
- Vault/HashiCorp integration

## 11. Migration plan
**Not applicable** — no database migration. Existing auto-generation behavior is preserved when env var is unset. Backward compatible.

## 12. Wireframes / Diagrams
```
BIGBASE_JWT_SECRET set?  ──Yes──► Validate (≥32B, not known default) ──► Use it
        │                                    │
        No                                   ✗ Invalid ──► Fatal error
        │
        ▼
  Auto-generate via crypto/rand ──► WARN "JWT secret not configured, using auto-generated secret"
```

## 13. API / Data Model
No new API. Config change only:
- **Env var**: `BIGBASE_JWT_SECRET` (string, min 32 chars)
- **Log line (configured)**: `"JWT secret loaded from configuration"`
- **Log line (auto)**: `"WARN JWT secret not configured, using auto-generated secret. Set BIGBASE_JWT_SECRET for persistent tokens across restarts."`

## 14. Affected code
| File | Change |
|------|--------|
| `components/auth/auth.go` | Replace inline `rand.Read` in `New()` with `resolveJWTSecret()` call |
| `components/auth/auth_test.go` | Add `TestJWTSecret*` tests (env var, validation, fallback) |
| `main.go` | No changes (auth.Options.Secret already piped) |
| `specs/tech-architecture/tech-stack.md` | Document `BIGBASE_JWT_SECRET` |

## 15. Testing strategy
- **Unit**: Table-driven tests for resolveJWTSecret (set, unset, too short, known default)
- **Integration**: `TestNew` variants — with and without env var
- **No E2E**: Env var is process-level, not worth full E2E

## 16. Rollback plan
Unset `BIGBASE_JWT_SECRET` — reverts to auto-generation behavior. Tokens signed with old secret become invalid (by design — same as secret rotation).

## 17. Acceptance Criteria
```gherkin
Scenario: JWT secret loaded from env var
  Given BIGBASE_JWT_SECRET is set to a 64-char hex string
  When BigBase starts
  Then tokens signed with that secret validate successfully
  And "JWT secret loaded from configuration" is logged

Scenario: Auto-generated secret when not configured
  Given BIGBASE_JWT_SECRET is not set
  When BigBase starts
  Then a random 64-char hex secret is generated
  And a warning is logged

Scenario: Short secret rejected
  Given BIGBASE_JWT_SECRET is set to "short"
  When BigBase starts
  Then auth.New fatals with "BIGBASE_JWT_SECRET must be at least 32 bytes"

Scenario: Known default rejected
  Given BIGBASE_JWT_SECRET is set to "test-secret-32-chars!!!"
  When BigBase starts
  Then auth.New fatals with "BIGBASE_JWT_SECRET is a known default"
```

## 18. Implementation Steps (see e50s01-tasks.yaml)
1. Add `resolveJWTSecret` function with validation → verify: `go test -run 'TestJWTSecret' ./components/auth/ -v -count=1`
2. Wire into `auth.New` with log messages → verify: `go test -run 'TestNew.*Secret' ./components/auth/ -v -count=1`
3. Add comprehensive unit tests → verify: `go test -run 'TestJWTSecret' ./components/auth/ -v -count=1`
4. Document in tech-stack.md → verify: `rg 'BIGBASE_JWT_SECRET' specs/tech-architecture/tech-stack.md`
5. Full test suite regression → verify: `go test ./components/auth/... -count=1 && go vet ./components/auth/...`

## 19. Verification Script (for manual UAT)
1. `export BIGBASE_JWT_SECRET=$(openssl rand -hex 32)`
2. `go run . serve --port 9999 --db :memory:`
3. `curl http://localhost:9999/health` → 200
4. Register + login — save token
5. Stop server, restart with same `BIGBASE_JWT_SECRET`
6. Use saved token → still valid (proves persistence)

## 20. Out of scope
- Secret rotation
- Multiple active secrets
- Kubernetes Secret integration
- File-based secret loading
