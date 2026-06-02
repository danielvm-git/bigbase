# Slice 4: Auth — "See Login"

**type:** epic  
**status:** done  
**verify:** `curl -X POST /api/auth/login -d '{"email":"a@b.com","password":"x"}'` → JWT token

## Purpose

Email/password authentication with JWT tokens, bcrypt hashing, and bearer-token middleware for protecting API routes.

## Implementation

### components/auth/auth.go

- **Register** — `POST /api/auth/register` — creates user with bcrypt-hashed password
- **Login** — `POST /api/auth/login` — validates credentials, returns JWT
- **Middleware** — `Middleware(next http.Handler)` — extracts Bearer token, validates JWT, injects user info into request context
- **Auto-migration** — creates `users` table on first init (columns: `id`, `email`, `password_hash`, `created_at`)

### components/auth/jwt.go

- HS256 signing with configurable secret (default env `JWT_SECRET`)
- 24-hour expiry
- Claims: `user_id`, `email`, `exp`, `iat`

### HTTP Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | No | Create account |
| POST | `/api/auth/login` | No | Sign in, get JWT |
| GET | `/api/auth/me` | Yes | Get current user info |

### Security

- Passwords hashed with bcrypt (cost 10)
- No plaintext password storage
- Generic error messages ("invalid credentials") — no user enumeration
- Token expiry enforced server-side

## Configuration

```jsonc
{ "auth": { "jwt_secret_env": "JWT_SECRET" } }
```

## Verify

```bash
# Register
curl -X POST /api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"secret123"}'

# Login
curl -X POST /api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"secret123"}'

# Use token
TOKEN=$(curl -s -X POST /api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"secret123"}' | jq -r '.token')
curl -H "Authorization: Bearer $TOKEN" /api/auth/me
```

## Files

```
components/auth/
├── auth.go
└── jwt.go
```
