# ADR 002: JWT + bcrypt Auth Component

type: adr
context: BigBase needs authenticated access for the auto-CRUD API. Auth must be a self-contained kernel component with no external service dependency.

## Decision

Implement authentication as a kernel component using bcrypt for password hashing and HS256 JWTs for session tokens.

## Rationale

- **Self-contained**: SQLite users table + in-memory JWT secret. No Redis, no external auth provider.
- **bcrypt** (`golang.org/x/crypto/bcrypt`): Industry-standard adaptive hashing with built-in salt. DefaultCost (10) is adequate for v0.1; configurable later via Options.
- **HS256 JWT** (`github.com/golang-jwt/jwt/v5`): Stateless token verification — no DB lookup on every request. Secret auto-generated as 64-char hex string on first start (32 random bytes hex-encoded). Configurable via Options.Secret for production deployments.
- **24-hour token expiry**: Standard trade-off between UX (infrequent re-login) and security (limited window if token leaked). Can be shortened via Options in future.
- **Email normalized to lowercase at rest**: Prevents duplicate accounts due to case differences (e.g. `User@Example.com` vs `user@example.com`).
- **Min 6-char password**: Prevents trivially weak passwords without external validation. Can be strengthened with zxcvbn or similar later.

## Consequences

- **No refresh tokens** — clients must re-authenticate after 24h. Acceptable for v0.1; refresh token rotation can be added as Slice 8+.
- **Secret lost on restart unless configured** — auto-generated secret is in-memory only. All existing JWTs become invalid on restart unless `Options.Secret` is set explicitly. Acceptable for dev; production deploys must set the secret.
- **No email verification** — accounts are immediately usable after registration. Deferred to Slice 7 (Email verification).
- **No OAuth/SSO** — deferred to Slice 9 (Social login). Current approach supports email/password only.
- **No password reset** — deferred to Slice 7 (Forgot password flow).
- **Login not rate-limited** — vulnerable to brute force at the application layer. Mitigated partially by the per-IP rate limiting at the proxy level (future work). ADR 003 should address auth-specific rate limiting.
- **Rows returned for non-existent user vs wrong password** — both paths return the same generic "invalid email or password" error to prevent user enumeration.

## Status

Accepted (Slice 4, commit dbbb7e2).
