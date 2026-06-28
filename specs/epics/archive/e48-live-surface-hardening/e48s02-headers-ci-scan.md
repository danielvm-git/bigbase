# e48s02: Add missing security headers and CI scanning

## 1. Story ID
e48s02

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
Add the missing `Permissions-Policy` and `Cache-Control: no-store` security headers to the proxy middleware. Integrate SAST (gosec), SCA (govulncheck), and secrets scanning (gitleaks) into the CI preflight pipeline. Create the `npm run preflight` scripts referenced in AGENTS.md that don't yet exist.

## 8. Problem Statement
Two categories of gaps:
1. **Missing HTTP security headers**: The proxy's `securityHeadersMiddleware` sets CSP, HSTS, X-Frame-Options, X-Content-Type-Options, and Referrer-Policy — but is missing `Permissions-Policy` (restricts browser feature access) and `Cache-Control: no-store` (prevents sensitive response caching on `/health` and `/api/*` routes).
2. **No automated security scanning in CI**: The project has no SAST, SCA, or secrets scanning integrated into the build pipeline. AGENTS.md references `npm run preflight` for the build gate, but these scripts don't exist in `package.json`.

## 9. Proposed Solution

### Headers
- Add `Permissions-Policy` header to `securityHeadersMiddleware` with a restrictive default: `accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()`
- Add `Cache-Control: no-store` to API and health routes via a new `cacheControlMiddleware` (or inline in `securityHeadersMiddleware` for JSON content-type responses)

### CI Scanning
- Install gosec `[OK]`, govulncheck `[OK]`, gitleaks `[OK]` (GPL-3.0, dev-tool use only — approved)
- Create `npm run preflight:go` script: `go vet ./... && go test ./... && gosec -quiet ./...`
- Create `npm run preflight:build`: `go build -o /dev/null .`
- Create `npm run preflight`: chains `preflight:go` + `preflight:ui` 
- Add gitleaks to `preflight:go` (or separate `preflight:secrets`)
- Document in `package.json` as proper npm scripts

## 10. Affected Modules

| Module | Purpose | Callers | Contracts |
|--------|---------|---------|-----------|
| `components/proxy/` | HTTP server, routing, security headers | All HTTP clients; kernel lifecycle | Middleware chain; `securityHeadersMiddleware` sets headers on every response |
| `package.json` (root) | Build scripts and CI entry points | CI (GitHub Actions), developer CLI, agent workflows (`/ship`) | `npm run preflight` must exit 0 on pass, non-zero on fail |

## 11. Dependencies
- **gosec** `[OK]` — standard Go SAST, MIT licensed, well-maintained by securego. Version: `go install github.com/securego/gosec/v2/cmd/gosec@latest`
- **govulncheck** `[OK]` — official Go team SCA tool, BSD licensed. Run via `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
- **gitleaks** `[OK]` — GPL-3.0, dev-tool CLI use only (not linked). Approved by user. Installed via `brew install gitleaks` or `go install github.com/gitleaks/gitleaks/v8@latest`

## 12. Implementation Steps

### Story e48s02: Security headers + CI scanning — Implementation Steps

**type:** feat
**context:** infra
**Context**: Add two missing security headers to the proxy's security middleware (Permissions-Policy and Cache-Control: no-store for sensitive routes), then wire automated security scanning into the build pipeline by creating the `npm run preflight` scripts and integrating gosec (SAST), govulncheck (SCA), and gitleaks (secrets detection).

## Steps

1. Add `Permissions-Policy` header to `securityHeadersMiddleware` with restrictive defaults → verify: `go test ./components/proxy/ -run TestPermissionsPolicy -v`

2. Add `Cache-Control: no-store` header to `/health` and `/api/*` responses in `securityHeadersMiddleware` (or dedicated middleware) → verify: `go test ./components/proxy/ -run TestCacheControl -v`

3. Run full proxy test suite to confirm no header regressions → verify: `go test ./components/proxy/ -v`

4. Install SAST/SCA/secrets tools: `gosec`, `govulncheck`, `gitleaks` → verify: `which gosec && which gitleaks && go run golang.org/x/vuln/cmd/govulncheck@latest -h > /dev/null 2>&1 && echo 'tools: OK'`

5. Create `npm run preflight:go` script in `package.json` (go vet + test + gosec + govulncheck) → verify: `npm run preflight:go`

6. Create `npm run preflight:secrets` script in `package.json` (gitleaks) → verify: `npm run preflight:secrets`

7. Create `npm run preflight` meta-script chaining go + ui + secrets → verify: `npm run preflight 2>&1 | tail -5`

8. Run gosec against full codebase, triage findings, add `// #nosec` annotations for intentional patterns → verify: `gosec -quiet ./...` exits 0

9. Run gitleaks against full git history, verify no secrets leaked → verify: `gitleaks detect --source . --verbose 2>&1 | grep -E 'no leaks found|leaks found: 0'`

## Verification Script (Step-by-Step)

1. **Verify Permissions-Policy header is present**:
   ```bash
   go run . serve --port 9999 &
   sleep 2
   curl -sI http://localhost:9999/ | grep -i permissions-policy
   # Expected: Permissions-Policy: accelerometer=(), camera=(), ...
   kill %1
   ```
2. **Verify Cache-Control: no-store on /health**:
   ```bash
   go run . serve --port 9999 &
   sleep 2
   curl -sI http://localhost:9999/health | grep -i cache-control
   # Expected: Cache-Control: no-store
   kill %1
   ```
3. **Verify Cache-Control: no-store on /api/version**:
   ```bash
   go run . serve --port 9999 &
   sleep 2
   curl -sI http://localhost:9999/api/version | grep -i cache-control
   # Expected: Cache-Control: no-store
   kill %1
   ```
4. **Run full preflight**:
   ```bash
   npm run preflight
   # Expected: all tasks pass, exit 0
   ```
5. **Verify gosec finds no issues**:
   ```bash
   gosec -quiet ./...
   # Expected: exit 0 with no output
   ```
6. **Verify govulncheck finds no known vulns**:
   ```bash
   go run golang.org/x/vuln/cmd/govulncheck@latest ./...
   # Expected: "No vulnerabilities found"
   ```
7. **Verify gitleaks finds no secrets**:
   ```bash
   gitleaks detect --source . --verbose
   # Expected: no leaks found
   ```

## Out of scope

- Nonce-based CSP (e48s05-e60 handles this)
- Customizable Permissions-Policy per-route
- SAST/SCA as GitHub Actions workflow (use `npm run preflight` in existing CI; dedicated scheduled scans are e48s03)
- trufflehog (gitleaks approved by user)

## Risks

- **gosec false positives**: May flag intentional patterns (e.g., `math/rand` for non-crypto use, file path inclusions). Mitigation: triage each finding, add `// #nosec` with justification comment for intentional patterns.
- **govulncheck network dependency**: Requires network access to fetch vulnerability database. Mitigation: CI already has network access; local dev can skip with `|| true` if offline.
- **gitleaks history scan**: May find previously-committed test secrets or example tokens. Mitigation: use `.gitleaks.toml` to whitelist test fixtures and example data.
- **Permissions-Policy breakage**: Overly restrictive policy could break embedded maps, payment forms, or camera/mic features if BigBase ever adds them. Mitigation: start restrictive, loosen per-route if needed later (documented as future work).

## 13. Definition of Done
- [x] `Permissions-Policy` header present on all responses with restrictive defaults
- [x] `Cache-Control: no-store` present on `/health` and `/api/*` responses
- [x] All existing proxy tests pass
- [x] `npm run preflight:go` script exists and passes
- [x] `npm run preflight:secrets` script exists and passes
- [x] `npm run preflight` meta-script exists and passes
- [x] gosec runs clean (zero HIGH/MEDIUM issues after triage)
- [x] govulncheck reports no known vulnerabilities
- [x] gitleaks reports no secrets in repo history

## 14. Acceptance Criteria (Gherkin)
```gherkin
Feature: Security headers and CI scanning

  Scenario: Permissions-Policy header present on home page
    When I request GET /
    Then the response includes Permissions-Policy header
    And the header value restricts accelerometer, camera, geolocation, gyroscope, magnetometer, microphone, payment, usb

  Scenario: Permissions-Policy header present on API routes
    When I request GET /health
    Then the response includes Permissions-Policy header

  Scenario: Cache-Control no-store on health endpoint
    When I request GET /health
    Then the response includes Cache-Control: no-store

  Scenario: Cache-Control no-store on API routes
    When I request GET /api/version
    Then the response includes Cache-Control: no-store

  Scenario: Home page does not get Cache-Control no-store
    When I request GET /
    Then the response does NOT include Cache-Control: no-store

  Scenario: npm run preflight:go passes
    When I run npm run preflight:go
    Then the command exits with status 0

  Scenario: gosec reports zero issues
    When I run gosec -quiet ./...
    Then the command exits with status 0

  Scenario: govulncheck reports no vulnerabilities
    When I run govulncheck ./...
    Then the output contains "No vulnerabilities found"

  Scenario: gitleaks finds no secrets
    When I run gitleaks detect --source .
    Then the output contains no leaks found
```

## 15. Non-Functional Requirements
- Header overhead: string constants set on every response (microseconds)
- gosec scan time: <30s on full codebase
- govulncheck scan time: <60s (network-dependent)
- gitleaks scan time: <10s
- Preflight total time: <2 minutes

## 16. Test Strategy
- **Unit**: `TestPermissionsPolicy` — verifies header value and content for home, /health, /api/version
- **Unit**: `TestCacheControl` — verifies no-store on /health and /api/*, absent on /
- **Integration**: Full `go test ./components/proxy/ -v`
- **Tool validation**: Each tool's verify command runs independently and as part of `npm run preflight`

## 17. Rollback Plan
- Remove `Permissions-Policy` header from `securityHeadersMiddleware`
- Remove `Cache-Control` logic from `securityHeadersMiddleware`
- Remove preflight scripts from `package.json`
- Uninstall gosec/gitleaks: `rm $(which gosec) $(which gitleaks)`
- No database migrations or state changes
