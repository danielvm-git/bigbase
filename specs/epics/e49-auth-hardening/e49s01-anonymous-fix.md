# e49s01: Fix anonymous tokens — add org_id context

## 1. Story ID
e49s01

## 2. Epic
e49 — Security: Auth Hardening (Anonymous, OAuth, Path Traversal)

## 3. Status
planned

## 4. BCPs
2

## 5. Type
fix

## 6. Context
domain

## 7. Summary
Anonymous tokens minted via `POST /api/auth/anonymous` always return 403 "no organization" because the JWT is created with `jwt.MapClaims` (no `org_id` field), but the auth middleware parses tokens into the `Claims` struct where `OrgID` defaults to `0`, which triggers the "fail closed" rejection at line 373. Fix by switching anonymous tokens to use the `Claims` struct with `role=anonymous` and adding a middleware bypass that lets anonymous tokens pass through without org isolation or email verification. Downstream handlers enforce read-only access via `UserRoleFromContext`.

## 8. Problem Statement
The `handleAnonymousToken` function in `anonymous.go` mints tokens with `jwt.MapClaims`:
```go
claims := jwt.MapClaims{
    "sub":  "anon-" + generateAnonymousID(),
    "role": "anonymous",
    "iat":  time.Now().Unix(),
    "exp":  time.Now().Add(1 * time.Hour).Unix(),
}
```

When the auth middleware (`Middleware()`) verifies this token via `verifyJWT`, it parses into the `Claims` struct:
```go
type Claims struct {
    UserID int64  `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    OrgID  int64  `json:"org_id"`
    jwt.RegisteredClaims
}
```

Since the anonymous token has no `org_id` claim, `claims.OrgID` is `0` (zero-value). The middleware then hits:
```go
// Fail closed: tokens missing org_id are rejected.
if claims.OrgID == 0 {
    writeJSON(w, http.StatusForbidden, map[string]string{"error": "no organization"})
    return
}
```

There are also no existing tests for anonymous tokens.

## 9. Proposed Solution
1. Switch `handleAnonymousToken` from `jwt.MapClaims` to the `Claims` struct, setting `Role: "anonymous"`, `OrgID: 0`
2. In `Middleware()`, after JWT verification and before email verification + org check, add:
   ```go
   if claims.Role == "anonymous" {
       ctx = context.WithValue(ctx, ctxUserRole, "anonymous")
       next.ServeHTTP(w, r.WithContext(ctx))
       return
   }
   ```
3. Downstream handlers that check `UserRoleFromContext` enforce read-only for role `"anonymous"` (existing behavior)
4. Anonymous tokens must NOT receive a refresh token (already enforced — anonymous endpoint doesn't issue refresh tokens)

## 10. Affected Modules

| Module | Purpose | Callers | Contracts |
|--------|---------|---------|-----------|
| `components/auth/` | Authentication, JWT minting/verification, middleware | Proxy (event hooks), API component (context values), all protected HTTP routes | `Middleware()` sets ctxUserID/Email/Role/OrgID; `Claims.OrgID == 0` rejected for non-anonymous; `verifyJWT` parses into `Claims` struct |

## 11. Dependencies
- **Zero new external packages** — uses existing `golang-jwt/jwt/v5` `[OK]` (already in use)

## 12. Implementation Steps

### Story e49s01: Fix anonymous tokens — Implementation Steps

**type:** fix
**context:** domain
**Context**: Anonymous tokens are broken in production because they lack `org_id` in the JWT claims, causing the auth middleware to reject them. Fix by migrating from `jwt.MapClaims` to the typed `Claims` struct and adding a role-based bypass in the middleware that allows `role=anonymous` tokens to skip org isolation and email verification checks.

## Steps

1. Add `TestAnonymousToken` — table-driven test covering: mint token (200), use token on GET /api/collections (200), POST rejected (403), no refresh token in response → verify: `go test ./components/auth/ -run TestAnonymousToken -v`

2. Switch `handleAnonymousToken` from `jwt.MapClaims` to `Claims` struct with `Role: "anonymous"`, `OrgID: 0` → verify: `go test ./components/auth/ -run TestAnonymousToken -v`

3. Add anonymous role bypass in `Middleware()`: skip email verification and org check when `claims.Role == "anonymous"` → verify: `go test ./components/auth/ -run TestAnonymousToken -v`

4. Run full auth test suite to confirm no regressions → verify: `go test ./components/auth/ -v`

## Verification Script (Step-by-Step)

1. **Start BigBase with anonymous auth enabled** (requires Google OAuth configured, or the bypass):
   ```bash
   go run . serve --port 9999 --google-client-id test --google-client-secret test
   ```
2. **Mint an anonymous token**:
   ```bash
   TOKEN=$(curl -s -X POST http://localhost:9999/api/auth/anonymous | jq -r '.token')
   echo "Token: ${TOKEN:0:20}..."
   # Expected: non-empty JWT string
   ```
3. **Verify no refresh token in response**:
   ```bash
   curl -s -X POST http://localhost:9999/api/auth/anonymous | jq '.refresh_token'
   # Expected: null
   ```
4. **Use anonymous token on a read endpoint**:
   ```bash
   curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9999/api/collections/items | jq '.'
   # Expected: 200 OK (not 403)
   ```
5. **Verify anonymous token rejected for write**:
   ```bash
   curl -s -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"test":1}' http://localhost:9999/api/collections/items
   # Expected: 403
   ```
6. **Verify anonymous token works without org context**:
   ```bash
   curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9999/api/version
   # Expected: 200 (no org isolation panic)
   ```

## Out of scope

- Configurable anonymous token TTL (hardcoded 1 hour, same as before)
- Rate limiting on anonymous token minting (handled by e47)
- Anonymous token revocation (no refresh token, so natural expiry sufficient)
- Per-collection anonymous access control (future feature)

## Risks

- **Downstream handler breakage**: Some handlers may panic if `OrgIDFromContext` returns 0 or `UserIDFromContext` returns 0 for anonymous tokens. Mitigation: the bypass sets context values inline; handlers that check `UserRoleFromContext` for "anonymous" should already handle this.
- **Email verification bypass**: The anonymous bypass also skips email verification (since anonymous users have no email). This is intentional but should be documented clearly in the code.
- **Token structure change**: Switching from `MapClaims` to `Claims` struct changes the JSON fields in the JWT payload (e.g., `sub` → `user_id`). Existing anonymous tokens will become invalid after deploy. Mitigation: anonymous tokens have 1-hour TTL, so any inflight tokens expire quickly.

## 13. Definition of Done
- [x] Anonymous tokens mint with `Claims` struct, including `role=anonymous`
- [x] Auth middleware bypasses org isolation and email verification for `role=anonymous`
- [x] Anonymous token accepted on read endpoints (200, not 403)
- [x] Anonymous token rejected on write endpoints (403)
- [x] Anonymous token response has no `refresh_token` field
- [x] `go test ./components/auth/ -run TestAnonymousToken -v` passes
- [x] Full auth test suite passes (`go test ./components/auth/ -v`)
- [x] No existing test regressions

## 14. Acceptance Criteria (Gherkin)
```gherkin
Feature: Anonymous token with org context

  Scenario: Anonymous token is minted successfully
    Given BigBase is running with Google OAuth configured
    When I POST /api/auth/anonymous
    Then the response status is 200
    And the response body contains a valid JWT in the "token" field
    And the response body does NOT contain "refresh_token"

  Scenario: Anonymous token accepted by middleware
    Given an anonymous token is minted via POST /api/auth/anonymous
    When the token is used to access GET /api/collections/items
    Then the response status is 200 OK

  Scenario: Anonymous user cannot create records
    Given an anonymous token
    When POST /api/collections/items is called
    Then the response status is 403

  Scenario: Anonymous user cannot modify records
    Given an anonymous token
    When PUT /api/collections/items/1 is called
    Then the response status is 403

  Scenario: Anonymous user cannot delete records
    Given an anonymous token
    When DELETE /api/collections/items/1 is called
    Then the response status is 403

  Scenario: Regular user tokens unaffected
    Given a registered user token
    When GET /api/collections/items is called
    Then the response status is 200

  Scenario: Anonymous token has 1-hour TTL
    Given an anonymous token is minted
    When the JWT is decoded
    Then the exp claim is iat + 3600 seconds
```

## 15. Non-Functional Requirements
- JWT payload size: comparable to MapClaims (Claims struct adds `user_id:0` and `org_id:0` fields, ~30 bytes extra)
- Middleware bypass: single string comparison (`claims.Role == "anonymous"`), O(1)
- No new database queries for anonymous tokens

## 16. Test Strategy
- **Unit**: `TestAnonymousToken` — table-driven test with subtests:
  - Mint token → 200, valid JWT, no refresh_token
  - Use token on GET → 200
  - Use token on POST → 403
  - Use token on PUT → 403
  - Use token on DELETE → 403
  - Expired token → 401
  - Tampered token → 401
- **Integration**: Use `httptest` with real auth component setup (same pattern as `oauth_redirect_test.go`)
- **Regression**: Full `go test ./components/auth/ -v`

## 17. Rollback Plan
- Revert `anonymous.go` to use `jwt.MapClaims`
- Remove the `claims.Role == "anonymous"` bypass from `Middleware()`
- Delete `TestAnonymousToken`
- No database changes — pure code rollback
