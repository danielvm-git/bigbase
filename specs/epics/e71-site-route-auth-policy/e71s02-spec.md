### Story e71s02: Proxy JWT validation for protected static paths — Implementation Steps

**type:** feat
**risk:** P0
**context:** domain
**Context**: Once auth policies are stored, the proxy layer must enforce them. When intercepting requests for a registered host, the proxy must resolve the site's policy, determine if the requested path is protected, and validate the credentials if needed. Validation supports either a session JWT (from the `Authorization: Bearer` header or the `token` cookie) or a site deploy key (`bb_dep_`). If validation fails, it returns `401 Unauthorized`.

## Steps

1. In `components/proxy/hosts.go`, add `AuthPolicy *AuthPolicy` to the `hostInfo` struct. Update `RegisterDeploymentHost` to read the `auth_policy` column from the database for the given `siteID` and store the parsed policy. Update all tests and callers of `RegisterDeploymentHost` to ensure they compile. Tag database driver and json packages as `[OK]`. → verify: `go test -v -run TestRegisterDeploymentHost ./components/proxy/...`
2. Add the following callbacks to `Options` in `components/proxy/proxy.go`:
   ```go
   ValidateToken   func(token string) (userID int64, role string, err error)
   ValidateSiteKey func(siteID string, token string) error
   ```
   Add a path matching helper in `components/proxy/hosts.go`:
   ```go
   func matchPath(pattern, path string) bool {
       if strings.HasSuffix(pattern, "/*") {
           prefix := strings.TrimSuffix(pattern, "/*")
           return strings.HasPrefix(path, prefix)
       }
       return pattern == path
   }
   ```
   Implement `isPathProtected(policy *AuthPolicy, path string) bool` that checks:
   - If `path` matches any pattern in `policy.PublicPaths` → return `false`
   - If `path` matches any pattern in `policy.ProtectedPaths` → return `true`
   - Fallback to `policy.Default == "protected"` → verify: `go test -v -run TestPathMatching ./components/proxy/...`
3. Update `deploymentHostMiddleware` in `components/proxy/hosts.go` to enforce the authentication policy before proxying the request:
   - Retrieve the `AuthPolicy` from the host's `hostInfo`. If nil or empty, bypass authentication.
   - If the path is protected:
     - Read the token from the `Authorization: Bearer <token>` header or the `token` cookie.
     - Extract/validate the credentials based on `policy.Accept`:
       - If `accept` includes `jwt` and token is valid → allow request.
       - If `accept` includes `site_key` and token has `bb_dep_` prefix → validate using `ValidateSiteKey`.
     - If credentials are missing or invalid, respond with `401 Unauthorized` (`{"error":"unauthorized"}`).
   - Write comprehensive unit tests in `components/proxy/hosts_test.go` covering the entire auth policy matrix. → verify: `go test -v -run TestProxyAuthPolicy ./components/proxy/...`

## Verification Script (Step-by-Step)

1. Run `go test ./components/proxy/...` to verify all proxy host routing and auth policy checks pass.
2. Run `go build -o bigbase .` to verify compilation.

## Out of scope

- Forwarding user identity headers to downstream apps (e71s03).
- MCP tool integration (e71s04).

## Risks

- Over-aggressive path blocking that breaks public static assets (e.g. `/assets/*`). Mitigated by explicit `public_paths` allow-list matching first.
- Stale policy updates on active connections. Mitigated by `UpdateAuthPolicy` callback in `e71s01` which modifies the policy in memory instantly.
