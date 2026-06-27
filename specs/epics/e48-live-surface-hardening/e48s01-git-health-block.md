# e48s01: Block .git exposure and harden /health endpoint

## 1. Story ID
e48s01

## 2. Epic
e48 — Security: Live Surface Hardening

## 3. Status
planned

## 4. BCPs
2

## 5. Type
feat

## 6. Context
infra

## 7. Summary
Block `/.git/*` path access in the proxy middleware chain (returning 404) and add Bearer-token authentication gating to the `/health` endpoint. The `/health` endpoint currently exposes the full 19-component architecture map publicly — including component names, versions, running status, dependencies, and hooks. This is an information disclosure vulnerability confirmed on the live deployment at bigbase.click.

## 8. Problem Statement
Two live-surface vulnerabilities:
1. **`.git` exposure**: `/.git/config` and similar paths are not blocked by the proxy, allowing attackers to probe for repository metadata or accidentally-exposed `.git` directories.
2. **`/health` information leak**: The health endpoint returns the complete component architecture map (19 components × {name, version, running, dependencies, hooks}) to unauthenticated clients. This aids reconnaissance.

## 9. Proposed Solution
- Add a `gitPathBlockMiddleware` to the proxy middleware chain that checks for `/.git` in the request path and returns 404.
- Add a `HEALTH_TOKEN` field to `Options` and a `healthToken` field to `Proxy`. In `handleHealth`, require `Authorization: Bearer <token>` when `HEALTH_TOKEN` is configured; omit the check when the token is empty (backward-compatible for dev/test).
- Both changes live entirely within `components/proxy/` — no cross-component imports, preserving ECC contracts.

## 10. Affected Modules

| Module | Purpose | Callers | Contracts |
|--------|---------|---------|-----------|
| `components/proxy/` | HTTP server, routing, landing page, security headers | All external HTTP clients; kernel lifecycle; Deploy component (host registry via `Handle()`) | Middleware chain order preserved; `kernel.Component` interface; `/health` returns JSON component map |

## 11. Dependencies
- **Zero new external packages** — all changes use `net/http` standard library only.
- `HEALTH_TOKEN` is read from environment at application startup via existing flag/env plumbing.

## 12. Implementation Steps

### Story e48s01: Block .git exposure and harden /health — Implementation Steps

**type:** feat
**context:** infra
**Context**: Add two security middleware layers to the proxy component: (1) a `.git` path blocker that returns 404 for any request path containing `/.git`, and (2) optional Bearer-token authentication on `/health` gated by a `HEALTH_TOKEN` environment variable. Both changes follow the established proxy middleware pattern and add corresponding test coverage.

## Steps

1. Add `HealthToken` field to `proxy.Options` and `healthToken` field to `Proxy` struct; wire it in `New()` → verify: `go build ./components/proxy/`

2. Add `gitPathBlockMiddleware` that returns 404 for paths containing `/.git`; insert into middleware chain in `Handler()` after `requestIDMiddleware` → verify: `go test ./components/proxy/ -run TestGitPathBlocked -v`

3. Add Bearer-token auth check to `handleHealth`: when `p.healthToken != ""`, require `Authorization: Bearer <token>` header matching exactly; return 401 with JSON body `{"error":"unauthorized"}` on mismatch → verify: `go test ./components/proxy/ -run TestHealthAuth -v`

4. Update `TestProxyHealthEndpoint` to work with no token set (backward-compatible: when HEALTH_TOKEN is empty, /health remains public) → verify: `go test ./components/proxy/ -run TestProxyHealthEndpoint -v`

5. Run full proxy test suite to confirm no regressions → verify: `go test ./components/proxy/ -v`

## Verification Script (Step-by-Step)

1. **Start the server without HEALTH_TOKEN** (backward-compatible mode):
   ```bash
   go run . serve --port 9999
   ```
2. **Verify /health is accessible**:
   ```bash
   curl -s http://localhost:9999/health | jq .status
   # Expected: "ok"
   ```
3. **Stop the server; restart with HEALTH_TOKEN**:
   ```bash
   HEALTH_TOKEN=secret123 go run . serve --port 9999
   ```
4. **Verify /health without token returns 401**:
   ```bash
   curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/health
   # Expected: 401
   ```
5. **Verify /health with correct token returns 200**:
   ```bash
   curl -s -H "Authorization: Bearer secret123" http://localhost:9999/health | jq .status
   # Expected: "ok"
   ```
6. **Verify /.git/config returns 404**:
   ```bash
   curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/.git/config
   # Expected: 404
   ```
7. **Verify /.git/HEAD returns 404**:
   ```bash
   curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/.git/HEAD
   # Expected: 404
   ```
8. **Verify normal routes unaffected**:
   ```bash
   curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/
   # Expected: 200
   ```

## Out of scope

- Rate limiting on /health (handled by e47)
- Permissions-Policy and Cache-Control headers (handled by e48s02)
- DAST scanning (handled by e48s03)
- Obfuscating component details in health response (the HEALTH_TOKEN gate is sufficient for now; if a deeper defense is needed, file a separate story)

## Risks

- **Token leak**: If HEALTH_TOKEN is logged or exposed in config, the health endpoint becomes accessible again. Mitigation: document in SECURITY.md that HEALTH_TOKEN should be rotated regularly and never committed.
- **Health probe breakage**: If a load balancer or monitoring tool probes /health without the token, it will get 401 and mark the service as down. Mitigation: the token is optional; deployments that use external health probes can leave HEALTH_TOKEN unset.
- **Middleware ordering**: The `.git` middleware must run before the mux dispatches the request. Current chain insertion point (after requestIDMiddleware, before deploymentHostMiddleware) is correct because it runs before any route handler. Risk is low — verified by integration test.

## 13. Definition of Done
- [x] `/.git/*` paths return 404 from proxy (any path containing `/.git`)
- [x] `/health` requires `Authorization: Bearer <token>` when `HEALTH_TOKEN` env var is set
- [x] `/health` returns 401 with JSON `{"error":"unauthorized"}` without valid token
- [x] `/health` remains publicly accessible when `HEALTH_TOKEN` is empty (backward-compatible)
- [x] All existing proxy tests pass (no regressions)
- [x] `go test ./components/proxy/ -run TestGitPathBlocked -v` passes
- [x] `go test ./components/proxy/ -run TestHealthAuth -v` passes
- [x] `npm run preflight:go` passes (Go vet + tests)

## 14. Acceptance Criteria (Gherkin)
```gherkin
Feature: Block sensitive paths and harden health endpoint

  Scenario: .git paths return 404
    Given BigBase is running
    When I request GET /.git/config
    Then the response status is 404

  Scenario: .git paths with nested segments return 404
    When I request GET /.git/HEAD
    Then the response status is 404

  Scenario: .git paths with double slashes return 404
    When I request GET /.git//config
    Then the response status is 404

  Scenario: /health gated behind token
    Given BigBase is configured with HEALTH_TOKEN=secret123
    When I request GET /health without Authorization header
    Then the response status is 401
    And the response body contains {"error":"unauthorized"}

  Scenario: /health available with correct token
    Given BigBase is configured with HEALTH_TOKEN=secret123
    When I request GET /health with Authorization: Bearer secret123
    Then the response status is 200
    And the response includes component status data

  Scenario: /health available without token when HEALTH_TOKEN is empty
    Given BigBase is configured without HEALTH_TOKEN
    When I request GET /health without Authorization header
    Then the response status is 200

  Scenario: Other routes unaffected by .git middleware
    When I request GET /
    Then the response status is 200

  Scenario: Other routes unaffected by health auth when token set
    Given BigBase is configured with HEALTH_TOKEN=secret123
    When I request GET / without Authorization header
    Then the response status is 200
```

## 15. Non-Functional Requirements
- Middleware overhead: `strings.Contains` check per request (~nanoseconds)
- Token comparison: constant-time via `crypto/subtle.ConstantTimeCompare` to prevent timing attacks
- No new external dependencies

## 16. Test Strategy
- **Unit**: `TestGitPathBlocked` — table-driven test covering `/.git/config`, `/.git/HEAD`, `/.git//objects`, `/not-git`, `/api/health`
- **Unit**: `TestHealthAuth` — table-driven test covering: no token set (200), token set + no header (401), token set + wrong token (401), token set + correct token (200)
- **Integration**: Existing `TestProxyHealthEndpoint` and `TestPerComponentHealth` updated to pass with optional token
- **Regression**: Full `go test ./components/proxy/ -v` must pass

## 17. Rollback Plan
- Remove `gitPathBlockMiddleware` from the `Handler()` chain
- Remove the `if p.healthToken != ""` block from `handleHealth`
- Revert `Options` and `Proxy` struct changes
- No database migrations or state changes — pure stateless HTTP middleware
