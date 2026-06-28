# e58s05: Astro SDK Refactor

**Story ID:** e58s05 | **Epic:** e58 — Native Feature Port | **BCPs:** 2 | **Status:** planned

## 1. Type & Context

**type:** fix + refactor
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As a** developer using `@bigbase/auth-astro` in an Astro project,
**I want** the Astro middleware to correctly validate and expose the authenticated session,
**so that** server-side rendering and API routes can reliably access the current user without silent authentication failures.

## 3. Context

`packages/auth-astro/src/index.ts` contains a bug: the `createBigBaseAuth` middleware extracts a token from the request cookie but never passes it to the auth client before calling `client.getSession()`. The internal `storedToken` in the `@bigbase/auth` client remains `null`, so `getSession()` calls `GET /api/auth/me` without an `Authorization` header — returning `null` for every authenticated request.

```typescript
// Current broken code (packages/auth-astro/src/index.ts):
const token = cookieHeader.match(/token=([^;]+)/)?.[1] ?? null;
if (token) {
  const session = await client.getSession(); // BUG: storedToken is null, token never set
```

Additionally:
1. `requireAuth` claims to return `AstroAuth` but can return `Response` (from `astro.redirect('/login')`) — type unsafety.
2. There is no `setToken` or `createAuthClientWithToken` API on `@bigbase/auth`, so the middleware cannot inject the extracted token without reaching into private state.
3. `AstroAuth` exposes `session: any` — untyped.

This story fixes the bug, tightens types, and adds a `setToken(token: string): void` method to `@bigbase/auth` so the Astro middleware can legitimately set the token before calling `getSession()`.

### Zoom-Out Summary
- **Module purpose:** `packages/auth-astro` is the Astro framework adapter for BigBase auth — a middleware function and helper utilities for Astro SSR routes.
- **Callers:** End-user Astro projects that add `createBigBaseAuth(...)` as Astro middleware. No BigBase server components depend on this package.
- **Contracts:** `AuthClient` interface in `@bigbase/auth` gains `setToken` — additive, no breaking change to existing callers (no existing caller calls `setToken`). `requireAuth` return type changes from `AstroAuth` to `AstroAuth | Response` — technically a broadening, but existing correct usage is unaffected.

## 4. Domain Model

No database changes. TypeScript-only changes across two packages.

```typescript
// packages/auth/src/index.ts additions:
interface AuthClient {
  // ... existing methods ...
  setToken(token: string): void  // NEW: sets storedToken for subsequent requests
}

// packages/auth-astro/src/index.ts:
interface AstroAuthLocals {  // NEW: for App.Locals extension
  auth: AstroAuth | null;
}

type AstroAuth = {
  user: AuthUser | null;
  session: AuthSession | null;  // typed instead of any
  token: string | null;
}
```

## 5. Contract / Interface

**In `packages/auth/src/index.ts`:**
```typescript
// Add to AuthClient interface:
setToken(token: string): void;

// Add to createAuthClient implementation:
setToken(token: string): void {
  storedToken = token;
},
```

**In `packages/auth-astro/src/index.ts`:**
```typescript
// Fixed createBigBaseAuth middleware:
export function createBigBaseAuth(options: { baseURL: string; cookieSecret?: string }) {
  const client = createAuthClient({ baseURL: options.baseURL });
  return async (context: any, next: () => Promise<Response>) => {
    const cookieHeader = context.request?.headers?.get('cookie') ?? '';
    const token = cookieHeader.match(/token=([^;]+)/)?.[1] ?? null;
    if (token) {
      client.setToken(token);  // FIX: inject extracted token before getSession()
      try {
        const session = await client.getSession();
        context.locals.auth = session
          ? { user: session.user, session, token }
          : null;
      } catch {
        context.locals.auth = null;
      }
    } else {
      context.locals.auth = null;
    }
    return next();
  };
}

// Fixed requireAuth return type:
export function requireAuth(astro: any): AstroAuth | Response

// New export:
export type { AstroAuthLocals };
```

## 6. Implementation Strategy

1. Add `setToken(token: string): void` to the `AuthClient` interface and the `createAuthClient` return object in `packages/auth/src/index.ts`.
2. Fix `createBigBaseAuth` in `packages/auth-astro/src/index.ts` to call `client.setToken(token)` before `client.getSession()`.
3. Fix `AstroAuth.session` type from `any` to `AuthSession | null`.
4. Fix `requireAuth` return type annotation to `AstroAuth | Response`.
5. Export `AstroAuthLocals` type.
6. Build both packages and run tests.

## 7. Data Flow

```
Astro SSR request
  → createBigBaseAuth middleware
      → extract token from cookie
      → client.setToken(token)          ← FIX: was missing
      → client.getSession()
          → GET /api/auth/me with Authorization: Bearer <token>
          → returns AuthSession or null
      → context.locals.auth = { user, session, token } | null
  → next() → SSR route handler
      → getSession(Astro) → context.locals.auth
```

## 8. Error Handling

- `getSession()` throwing (network error, non-200 response) → `context.locals.auth = null` (unchanged).
- `requireAuth` returning a redirect `Response` on missing auth — callers must check the return type (no exception thrown).

## 9. Testing Strategy

- **Unit — `setToken`:** call `setToken("abc")`, then verify `getJWT()` returns `"abc"` without making an HTTP call.
- **Unit — `createBigBaseAuth` middleware:** mock fetch; with cookie `token=jwt123`, verify fetch is called with `Authorization: Bearer jwt123`; without cookie, verify `context.locals.auth` is `null`.
- Run: `cd packages/auth && npm test` and `cd packages/auth-astro && npm run build`.

## 10. Migration / Rollback

TypeScript packages — no migration needed. Rollback = revert the `setToken` addition and the middleware fix. `setToken` is purely additive; existing code not calling it is unaffected.

## 11. Documentation

Update `packages/auth-astro/README.md` (if it exists) or add a JSDoc comment to `createBigBaseAuth` explaining the cookie-to-token extraction and the `AstroAuthLocals` type usage.

## 12. Dependencies

- No new npm packages in either package.
- `@bigbase/auth` is already a dependency of `@bigbase/auth-astro`.

## 13. Observability

N/A — client-side TypeScript package.

## 14. Security

**Security level:** none

- Token is extracted from an HttpOnly cookie (set by BigBase backend). The middleware reads it server-side — no client-side exposure.
- `setToken` is an internal SDK method; it does not introduce a new attack surface since the caller (Astro middleware) already has access to the raw request cookies.

## 15. Performance

Negligible. One additional method call (`setToken`) per request.

## 16. Alternatives Considered

- **Direct fetch in auth-astro instead of using the client:** avoids the `setToken` addition, but duplicates the `GET /api/auth/me` call that `getSession()` already encapsulates. Rejected — DRY principle.
- **`createAuthClientWithToken(options, token)` factory:** heavier API surface; a `setToken` method is simpler and idiomatic for stateful auth clients.

## 17. Acceptance Criteria

```gherkin
Scenario: Middleware injects authenticated session when cookie is present
  Given a BigBase instance and an Astro app with createBigBaseAuth middleware
  When a request arrives with cookie "token=<valid-jwt>"
  Then context.locals.auth.user is the authenticated user
  And context.locals.auth.token equals the JWT

Scenario: Middleware sets auth to null when no cookie
  Given no token cookie in the request
  Then context.locals.auth is null

Scenario: setToken enables getSession to work with externally-extracted token
  Given an AuthClient with no stored token
  When client.setToken("my-jwt") is called
  Then client.getJWT() returns "my-jwt"

Scenario: requireAuth redirects unauthenticated users
  Given context.locals.auth is null
  When requireAuth is called
  Then a Response (redirect to /login) is returned
```

## 18. Out of Scope

- Cookie signing/verification (`cookieSecret` option is accepted but not yet used — out of scope for this story).
- Astro session store integration.
- Server-side token refresh.

## 19. Risks

- `storedToken` in `createAuthClient` is closure-local. If the same client instance is shared across concurrent requests (SSR), `setToken` could cause a race: request A's token overwrites request B's. Mitigated by the middleware creating a new client per request via `createBigBaseAuth` → `createAuthClient` is called once at middleware setup, not per-request. Since Node.js/Astro SSR handles requests serially in a single-threaded event loop, no concurrent mutation occurs for a single server process. Document this assumption.

## 20. Verification Script

```bash
cd packages/auth && npm test && npm run build && cd ../auth-astro && npm run build
```
