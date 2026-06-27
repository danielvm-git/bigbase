# e50s03: Refresh token revocation — per-user invalidation
## Story ID: e50s03 | Epic: e50 | BCPs: 1 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
The auth component already implements refresh token rotation (single-use tokens with family-based replay detection). However, there's no way for a user to explicitly revoke all their sessions — e.g., after a password change or security concern. The story adds a `POST /api/auth/logout-all` endpoint that marks all refresh tokens for the authenticated user as used.

## 3. Problem / Opportunity
- No "log out everywhere" capability — existing sessions continue until token expiry
- Security best practice: after password reset, all sessions should be invalidated
- Follows Supabase/Auth0 pattern: short-lived access tokens + refresh token rotation + explicit revocation

## 4. Proposed solution
Add `invalidateAllUserTokens(ctx, userID)` that performs:
```sql
UPDATE refresh_tokens SET used = 1 WHERE user_id = ?
```

Add `POST /api/auth/logout-all` handler that:
- Extracts `userID` from authenticated context
- Calls `invalidateAllUserTokens`
- Returns `{"ok": true}` with status 200
- Does NOT invalidate the current access token (max 24h remaining)

## 5. Alternatives considered
- **Token blacklist (JWT blocklist)**: Requires Redis or in-memory list, complex, and most JWT libraries advise against it. Token-routing pattern (short-lived access + revocable refresh) is simpler.
- **Per-token revocation**: Over-engineering. "Logout everywhere" covers the primary use case.

## 6. Who are the users?
- **End users** who want to secure their account after a device loss or password change
- **Admins** who need to force-logout a compromised user

## 7. Dependencies
- Existing `refresh_tokens` table (created by `migrateRefreshTokens`)
- `UserIDFromContext` (existing, in auth.go)
- `a.db.ExecContext` (existing DBer interface)

## 8. Assumptions
- Access tokens remain valid for their remaining lifetime (up to 24h). This is the Supabase/Auth0 pattern.
- `logout-all` is a protected endpoint (requires valid access token)

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Access token still valid after logout-all | Medium | Low | Max 24h window; documented behavior |
| DB write fails silently | Low | Medium | Return 500 on error |

## 10. Non-goals
- Immediate access token invalidation (requires token blocklist)
- Admin-initiated logout of specific user (future story)
- Password-change-triggered automatic revocation (future story — hooks into password-reset flow)

## 11. Migration plan
No database migration needed — the `refresh_tokens` table already exists with a `used` column. Backward compatible.

## 12. Wireframes / Diagrams
```
POST /api/auth/logout-all (Authorization: Bearer <token>)
  │
  ├─ Extract userID from JWT claims
  │
  ├─ UPDATE refresh_tokens SET used = 1 WHERE user_id = ?
  │
  └─ Return 200 { "ok": true }

After logout-all:
  ┌────────────────────────────────────┐
  │ Original access token              │
  │ ├─ Still valid (max 24h window)    │
  │ └─ Cannot be refreshed (401)       │
  └────────────────────────────────────┘
```

## 13. API / Data Model
**Endpoint**: `POST /api/auth/logout-all`
**Auth**: Bearer token required
**Response 200**:
```json
{ "ok": true }
```
**Response 401** (unauthenticated):
```json
{ "error": "authorization required" }
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/auth/refreshtoken.go` | Add `invalidateAllUserTokens(ctx, userID)` |
| `components/auth/auth.go` | Add `handleLogoutAll` handler; register in `Handler()` and `ProtectedHandler()` |
| `components/auth/auth_test.go` | Add `TestLogoutAll*` tests |
| `components/auth/refreshtoken_test.go` | Add `TestLogoutAll` in refresh token test |

## 15. Testing strategy
- **Unit**: Single refresh token invalidated by user
- **Unit**: Multiple refresh tokens all invalidated
- **Unit**: Subsequent refresh after logout-all returns 401
- **Unit**: Unauthenticated request returns 401
- **Unit**: Access token still valid after logout-all
- **Unit**: No refresh tokens for user → idempotent 200

## 16. Rollback plan
Disable the endpoint by removing the route registration. No data changes to undo.

## 17. Acceptance Criteria
```gherkin
Scenario: Logout all invalidates refresh tokens
  Given user has 3 active refresh tokens
  When POST /api/auth/logout-all is called
  Then all 3 refresh tokens are marked as used
  And subsequent refresh attempts return 401

Scenario: Access token still works after logout-all
  Given user has a valid access token
  When POST /api/auth/logout-all is called
  Then the access token remains valid for protected endpoints

Scenario: Unauthenticated request rejected
  Given no Authorization header
  When POST /api/auth/logout-all is called
  Then 401 is returned
```

## 18. Implementation Steps (see e50s03-tasks.yaml)
1. Add `invalidateAllUserTokens` → verify: `go test -run 'TestTokenRevocation' ./components/auth/ -v -count=1`
2. Add `handleLogoutAll` handler + route → verify: `go test -run 'TestLogoutAll' ./components/auth/ -v -count=1`
3. Full test coverage → verify: `go test -run 'TestLogoutAll' ./components/auth/ -v -count=1`
4. Update epic.yaml manifest + sync → verify: `bash scripts/sync-status-from-epics.sh`
5. Full test suite → verify: `go test ./components/auth/... -count=1 && go vet ./components/auth/...`

## 19. Verification Script (for manual UAT)
1. `go run . serve --port 9999 --db :memory:`
2. Register + login user `test@e50.com` → save access_token and refresh_token
3. Login again (from another "device") → get second refresh_token
4. `curl -X POST http://localhost:9999/api/auth/logout-all -H "Authorization: Bearer $access_token"`
5. Verify response `{"ok":true}`
6. Try to refresh with OLD refresh_token → **401**
7. Try to use access_token for protected route → **200** (still valid)

## 20. Out of scope
- Admin-initiated logout of specific users
- Immediate access token invalidation (token blocklist)
- Automatic revocation on password change
