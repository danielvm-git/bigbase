# e59s03: Better Auth Backend Engine

**Story ID:** e59s03 | **Epic:** e59 — Native Feature Port | **BCPs:** 3 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As a** developer using the `better-auth` JavaScript client library,
**I want** BigBase to expose the better-auth wire protocol endpoints,
**so that** I can use `better-auth`'s client SDK to authenticate against a self-hosted BigBase instance without writing a custom adapter.

## 3. Context

BigBase's Go auth component already provides login, register, JWT, OAuth, and session via its own routes (`/api/auth/login`, `/api/auth/register`, `/api/auth/me`). The `@bigbase/auth` TS client wraps these routes.

The `better-auth` npm library is a popular framework-agnostic JS auth library that expects a specific backend wire protocol:
- `POST /api/auth/sign-in/email` — accepts `{email, password}`, returns `{session, user}`
- `GET  /api/auth/get-session` — returns `{session, user}` or `null`
- `POST /api/auth/sign-out` — clears session

This story adds three compatibility handlers in `components/auth` that translate between BigBase's internal format and the better-auth wire format. No existing handlers are changed. No new JWT/session engine is introduced — the BigBase JWT *is* the session; the response shape is remapped.

### Zoom-Out Summary
- **Module purpose:** `components/auth` handles registration, login, JWT issuance, OAuth, and token validation. It is the trust root for all protected routes.
- **Callers:** `main.go` registers auth routes via `authComp.Handler()` and `authComp.ProtectedHandler()`. All other components consume JWTs issued here; none call auth methods directly.
- **Contracts preserved:** All existing `/api/auth/*` routes untouched. New routes added in `Handler()`. `Claims` struct, `createJWT`, `verifyJWT`, and `Middleware` are unchanged.

## 4. Domain Model

No new tables. No schema changes.

```
BetterAuthSession {
    id        string    // = SHA256(token)[:16] — stable per-token session ID
    userId    string    // = strconv.FormatInt(claims.UserID, 10)
    token     string    // = the JWT string
    expiresAt time.Time // = claims.ExpiresAt
    createdAt time.Time // = claims.IssuedAt
    updatedAt time.Time // = claims.IssuedAt
}

BetterAuthUser {
    id             string // = strconv.FormatInt(claims.UserID, 10)
    email          string
    name           string // = claims.Email (BigBase has no separate name field)
    emailVerified  bool   // = true (auth existence implies verification)
    createdAt      time.Time
    updatedAt      time.Time
}

BetterAuthResponse {
    session BetterAuthSession
    user    BetterAuthUser
}
```

## 5. Contract / Interface

```go
// components/auth/betterauth.go

// handleBetterAuthSignIn handles POST /api/auth/sign-in/email.
// Accepts {"email": string, "password": string}.
// Returns BetterAuthResponse JSON on success.
func (a *Auth) handleBetterAuthSignIn(w http.ResponseWriter, r *http.Request)

// handleBetterAuthGetSession handles GET /api/auth/get-session.
// Reads Bearer token from Authorization header or "better-auth.session_token" cookie.
// Returns BetterAuthResponse JSON or null.
func (a *Auth) handleBetterAuthGetSession(w http.ResponseWriter, r *http.Request)

// handleBetterAuthSignOut handles POST /api/auth/sign-out.
// Clears the session cookie and delegates to BigBase's logout path.
func (a *Auth) handleBetterAuthSignOut(w http.ResponseWriter, r *http.Request)

// betterAuthSession builds a BetterAuthResponse from a JWT token string and its claims.
func betterAuthResponse(token string, claims *Claims) map[string]any
```

New routes in `Handler()`:
```go
mux.HandleFunc("/api/auth/sign-in/email", a.handleBetterAuthSignIn)
mux.HandleFunc("/api/auth/get-session",   a.handleBetterAuthGetSession)
mux.HandleFunc("/api/auth/sign-out",      a.handleBetterAuthSignOut)
```

These do NOT replace the existing:
```
/api/auth/login    → handleLogin     (unchanged)
/api/auth/logout   → existing logout (unchanged)
/api/auth/me       → handleMe        (unchanged)
```

## 6. Implementation Strategy

1. Create `components/auth/betterauth.go` with the three handlers and `betterAuthResponse` helper.
2. `handleBetterAuthSignIn`: decode `{email, password}`, call the same credential-check logic used by `handleLogin` (extract into a shared `verifyCredentials(email, password) (token string, claims *Claims, err error)` helper if not already one, or call the DB query directly), build `betterAuthResponse`, write JSON 200.
3. `handleBetterAuthGetSession`: extract token from `Authorization: Bearer` header or cookie named `better-auth.session_token`; call `verifyJWT`; build `betterAuthResponse`; write JSON 200. If no valid token: write `null` JSON with 200.
4. `handleBetterAuthSignOut`: clear `better-auth.session_token` cookie (MaxAge=-1), write `{}` JSON 200.
5. Register the three routes in `Handler()`.
6. Write tests.

`betterAuthResponse` computes `sessionID` as `fmt.Sprintf("%x", sha256.Sum256([]byte(token)))[:16]`. Uses `crypto/sha256` from stdlib — no new dependencies.

## 7. Data Flow

```
POST /api/auth/sign-in/email
  → proxy middleware
  → auth.Handler (public, rate-limited when RL enabled)
  → handleBetterAuthSignIn
      → verifyCredentials(email, password)
      → betterAuthResponse(token, claims)
      → writeJSON 200 {session, user}

GET /api/auth/get-session
  → proxy middleware
  → auth.Handler
  → handleBetterAuthGetSession
      → extract token (Authorization header or cookie)
      → verifyJWT(token, secret)
      → if invalid/missing: writeJSON 200 null
      → betterAuthResponse(token, claims)
      → writeJSON 200 {session, user}
```

## 8. Error Handling

| Condition | HTTP Status | Body |
|-----------|-------------|------|
| Wrong credentials (sign-in) | 401 | `{"error": "invalid credentials"}` |
| Invalid JSON body | 400 | `{"error": "invalid request body"}` |
| Missing email or password | 400 | `{"error": "email and password are required"}` |
| Invalid/expired token (get-session) | 200 | `null` |
| Non-POST to sign-in or sign-out | 405 | `{"error": "method not allowed"}` |
| Non-GET to get-session | 405 | `{"error": "method not allowed"}` |

## 9. Testing Strategy

- **Unit — `betterAuthResponse`:** correct sessionID derivation, userId as string, expiresAt matches claims.
- **Integration (httptest) — sign-in/email:** valid credentials → 200 with `session.token`, `user.email`; wrong password → 401; missing fields → 400.
- **Integration — get-session:** valid Bearer token → 200 BetterAuthResponse; expired/invalid token → 200 `null`; cookie-based token → 200 BetterAuthResponse; no token → 200 `null`.
- **Integration — sign-out:** POST → 200 `{}`; `Set-Cookie` header clears `better-auth.session_token`.

## 10. Migration / Rollback

No schema changes. Rollback = remove three route registrations from `Handler()` and delete `betterauth.go`.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` to note better-auth wire protocol compatibility. Document the mapping (BigBase JWT ↔ better-auth session) for users who need to understand the translation.

## 12. Dependencies

- `crypto/sha256` — stdlib, no new dependency.
- No dependency on e57 (better-auth endpoints are org-level, not project-scoped, for this story).
- Assumes existing `verifyJWT` and credential-check logic in `components/auth/auth.go` are accessible within the same package.

## 13. Observability

- `logger.Info("better-auth sign-in", "user_id", claims.UserID)` on successful sign-in.
- `logger.Info("better-auth get-session", "valid", tokenValid)` on get-session.

## 14. Security

**Security level:** low

- Credential check reuses existing bcrypt comparison — no new auth logic.
- `verifyJWT` called for all get-session requests — forged tokens rejected.
- Session ID is a deterministic hash of the token — not a secret, does not enable token forgery.
- Cookie clearing on sign-out uses `MaxAge: -1, HttpOnly: true` (same pattern as existing logout).
- New routes fall under existing rate-limiting middleware (`rl.Middleware`) when rate limiting is enabled in `main.go`.

## 15. Performance

Three handler functions, no new DB queries beyond what existing login/me already do. `sha256.Sum256` is microsecond-range. No hot-path impact.

## 16. Alternatives Considered

- **Replace internal auth engine with better-auth Go library:** blast radius covers all existing auth handlers, JWT format, and every caller. 3 BCPs is far too narrow. Rejected.
- **New `@bigbase/better-auth` TS adapter package:** duplicates `packages/auth`, adds an npm package to maintain, and puts the compatibility burden on the client. Rejected — Go-side compatibility is cleaner.

## 17. Acceptance Criteria

```gherkin
Scenario: better-auth sign-in with valid credentials
  Given a registered user with email "user@test.com" and password "secret"
  When POST /api/auth/sign-in/email with {"email": "user@test.com", "password": "secret"}
  Then HTTP 200 with {session: {id, userId, token, expiresAt}, user: {id, email}}

Scenario: get-session with valid Bearer token
  Given a valid JWT obtained from sign-in
  When GET /api/auth/get-session with Authorization: Bearer <token>
  Then HTTP 200 with the same {session, user} shape

Scenario: get-session with no token returns null
  Given no Authorization header or cookie
  When GET /api/auth/get-session
  Then HTTP 200 with body null

Scenario: sign-out clears cookie
  When POST /api/auth/sign-out
  Then HTTP 200 with {} and Set-Cookie clearing better-auth.session_token

Scenario: wrong password rejected
  When POST /api/auth/sign-in/email with wrong password
  Then HTTP 401
```

## 18. Out of Scope

- Social/OAuth sign-in via better-auth protocol.
- better-auth plugin system or secondary storage adapters.
- Project-scoped sessions (deferred to e57 + future story).
- `rememberMe` field (BigBase JWT TTL is fixed at 24 h).

## 19. Risks

- better-auth client library may evolve its wire protocol. The three endpoints (`sign-in/email`, `get-session`, `sign-out`) are stable core paths as of better-auth v1.x — document the version targeted.
- Session ID derived from JWT hash is deterministic — a client that stores and compares session IDs across logins will see a new ID after each login (expected: BigBase issues a new JWT each time).

## 20. Verification Script

```bash
go build ./components/auth/ && go test ./components/auth/ -run TestBetterAuth -v -count=1
```
