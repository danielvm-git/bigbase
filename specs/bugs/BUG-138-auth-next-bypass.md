---
bug_id: BUG-138
status: fixed
severity: critical
scope: packages/auth-next
title: "Auth Bypass in @bigbase/auth-next SDK middleware"
github_issue: 138
---

# BUG-138: Auth Bypass in @bigbase/auth-next SDK middleware

## Summary

`bigBaseAuthMiddleware` in `packages/auth-next/src/index.ts` never calls `client.setToken(token)` before checking the session. Since `getSession()` swallows errors internally and returns `null`, any non-empty `token` cookie passes — regardless of validity.

## Root Cause

In the middleware (line 21-25):
```typescript
const token = request.cookies.get('token')?.value;
if (token) {
  try {
    await client.getSession();  // BUG: setToken never called!
    return NextResponse.next();  // ALWAYS reached — getSession doesn't throw
  } catch {
    request.cookies.delete('token');  // DEAD CODE — never reached
  }
}
```

### Why it fails

1. `createAuthClient({ baseURL })` creates a client with `storedToken = null`
2. `getSession()` calls `request('/api/auth/me')` which uses `storedToken` for the Authorization header
3. Without `setToken()`, no Authorization header is sent → server returns 401
4. `getSession()` CATCHES the error internally and returns `null` (doesn't throw!)
5. The try block completes → `NextResponse.next()` is always called
6. **Any non-empty token cookie (e.g. "x") bypasses authentication**

### Correct implementation (auth-astro)

```typescript
client.setToken(token);  // ← THIS LINE IS MISSING IN auth-next
const session = await client.getSession();
```

## Fix

1. Call `client.setToken(token)` before `getSession()`
2. Check the return value of `getSession()` — if null, redirect to login
3. Add regression test asserting garbage token is redirected

## Affected Files

- `packages/auth-next/src/index.ts` — the buggy middleware
- `packages/auth-next/src/index.test.ts` — needs regression test

## Verification

- [x] Garbage token cookie → redirected to login
- [x] Valid token cookie → passes through
- [x] No token cookie → redirected to login
- [x] All auth-next tests pass (8/8)
- [x] TypeScript compiles cleanly
- [x] Go tests pass (pre-existing TestResumeSvelteKitStaticDeployment failure unrelated)
