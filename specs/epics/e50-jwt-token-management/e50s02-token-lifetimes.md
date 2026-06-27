# e50s02: Configurable token lifetimes
## Story ID: e50s02 | Epic: e50 | BCPs: 2 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
Access and refresh token lifetimes are currently hardcoded: 24 hours for access tokens, 30 days for refresh tokens. Operators need to tune these for their security posture and use cases. Additionally, the auth response JSON doesn't expose expiry information, forcing clients to decode JWT payloads.

## 3. Problem / Opportunity
- No way to shorten token lifetimes for high-security environments
- No way to extend lifetimes for low-risk internal deployments
- Clients must decode JWT to determine remaining validity (`expires_at`, `expires_in` missing from response)

## 4. Proposed solution
1. Add `AccessExpiry` and `RefreshExpiry` fields to `auth.Options` (defaults: 24h, 30d)
2. Thread through `createJWT` and `issueRefreshToken` signatures
3. Add `--jwt-access-expiry` and `--jwt-refresh-expiry` CLI flags with env fallback (`BIGBASE_JWT_ACCESS_EXPIRY`, `BIGBASE_JWT_REFRESH_EXPIRY`)
4. Add `expires_at` (RFC3339) and `expires_in` (seconds) to `writeAuthResponse` and `handleRefresh` response JSON

## 5. Alternatives considered
- **Seconds-based config**: Natural for computers but less human-readable. Duration strings (`1h`, `30d`) are more user-friendly.
- **Per-client-type lifetimes**: Over-engineering for current scope. Single global config is sufficient.

## 6. Who are the users?
- **Operators** configuring security policy
- **Client SDK developers** who need expiry metadata in responses

## 7. Dependencies
- `time.ParseDuration` (stdlib)
- `golang-jwt/jwt/v5` (already used)
- Existing `auth.Options`, `createJWT`, `writeAuthResponse`, `refreshTokenExpiry`

## 8. Assumptions
- Duration format `1h30m` / `24h` / `30m` is acceptable (Go `time.ParseDuration` format)
- Defaults are 24h access, 30d refresh (current behavior preserved)
- Negative/zero durations are rejected

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Operator sets 1s access expiry | Low | High | Reject durations < 1 minute |
| Misparsed duration format | Medium | Low | Parse validation at startup, fatal on invalid |
| Refresh < Access expiry | Low | Medium | Validate refresh ≥ access at startup |

## 10. Non-goals
- Per-client or per-role token lifetimes
- Token refresh window / sliding expiry
- JWT `nbf` (not-before) claims

## 11. Migration plan
Backward compatible — defaults match current hardcoded values. No database changes. Existing tokens continue to work for their remaining lifetime.

## 12. Wireframes / Diagrams
```
CLI flag --jwt-access-expiry=1h ──► auth.Options.AccessExpiry = 1h
                                      │
  createJWT(userID, email, role, orgID, secret, ttl) ──► exp = now + ttl
                                      │
  writeAuthResponse(...)             ──► { token, refresh_token, expires_at, expires_in, user }
```

## 13. API / Data Model
Response changes to `/api/auth/register`, `/api/auth/login`, `/api/auth/refresh`, `/api/auth/oauth/google/callback`, and all passwordless/phone/anonymous endpoints:
```json
{
  "token": "eyJ...",
  "refresh_token": "abc...",
  "expires_at": "2026-06-28T12:00:00Z",
  "expires_in": 86400,
  "user": { "id": 1, "email": "a@b.com" }
}
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/auth/auth.go` | Add `AccessExpiry`/`RefreshExpiry` to `Options`; wire into `createJWT` call; add expiry fields to `writeAuthResponse` |
| `components/auth/jwt.go` | Add TTL parameter to `createJWT`; update `expires_in`/`expires_at` in response struct |
| `components/auth/refreshtoken.go` | Use `Options.RefreshExpiry` instead of const |
| `components/auth/passwordless.go` | Update callers of `createJWT` |
| `components/auth/phone.go` | Update callers of `createJWT` |
| `components/auth/anonymous.go` | Update callers of `createJWT` |
| `components/auth/magiclink.go` | Update callers of `createJWT` |
| `main.go` | Add `--jwt-access-expiry`/`--jwt-refresh-expiry` flags |
| `specs/tech-architecture/tech-stack.md` | Document new flags |

## 15. Testing strategy
- **Unit**: Custom lifetimes produce correct `exp` claim, response includes `expires_at` and `expires_in`
- **Unit**: Default lifetimes match current 24h/30d
- **Unit**: Invalid duration strings rejected (zero, negative, malformed)
- **Integration**: Login with custom expiry → token validates, has correct exp

## 16. Rollback plan
Remove `--jwt-access-expiry` and `--jwt-refresh-expiry` flags from CLI args. Defaults restore current 24h/30d behavior.

## 17. Acceptance Criteria
```gherkin
Scenario: Custom token lifetimes
  Given --jwt-access-expiry=1h
  When a user logs in
  Then the JWT expires 1 hour from now
  And the response includes expires_at and expires_in

Scenario: Default lifetimes when not configured
  Given no --jwt-access-expiry flag
  When a user logs in
  Then the JWT expires 24 hours from now (current behavior)

Scenario: Invalid duration rejected
  Given --jwt-access-expiry=0h
  When BigBase starts
  Then it fatals with "jwt-access-expiry must be at least 1 minute"
```

## 18. Implementation Steps (see e50s02-tasks.yaml)
1. Add expiry fields to Options → verify: `go test -run 'TestTokenLifetimes' ./components/auth/ -v -count=1`
2. Thread TTL through createJWT + all callers → verify: `go test ./components/auth/... -count=1`
3. CLI flags + env fallback in main.go → verify: `go run . serve --help 2>&1 | rg 'jwt-access-expiry'`
4. Response expiry fields → verify: `go test -run 'TestTokenExpiryResponse' ./components/auth/ -v -count=1`
5. Unit tests for lifetimes → verify: `go test -run 'TestToken' ./components/auth/ -v -count=1`
6. Tech-stack.md docs + full test suite → verify: `go test ./components/auth/... -count=1 && go vet ./components/auth/...`

## 19. Verification Script (for manual UAT)
1. `go run . serve --port 9999 --db :memory: --jwt-access-expiry=5m`
2. `curl -s -X POST http://localhost:9999/api/auth/register -H 'Content-Type: application/json' -d '{"email":"test@e50.com","password":"password123"}' | jq`
3. Verify `expires_at` is ~5 minutes from now
4. Verify `expires_in` is ~300
5. Wait 5 minutes, try using the token → 401

## 20. Out of scope
- Per-client token lifetimes
- Token refresh windows / sliding expiry
- JWT `nbf` (not-before) claims
