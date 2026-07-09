### Story e71s01: Site auth policy schema — Implementation Steps

**type:** feat
**risk:** P2
**context:** domain
**Context**: To enforce host-level authentication on deployed sites, we must first store the routing auth policy for each site. This story introduces the `auth_policy` column to the `sites` database table, updates the `Site` struct representation with a structured `AuthPolicy` sub-struct, and implements `GET` and `POST` API handlers at `/api/sites/:id/auth-policy` to view and update the policy.

## Steps

1. Add `auth_policy TEXT NOT NULL DEFAULT '{}'` to the `sites` table schema in `components/sites/sites.go` `Start()` migration. Add a schema migration block `ALTER TABLE sites ADD COLUMN auth_policy TEXT NOT NULL DEFAULT '{}'` (handled gracefully if already exists). Tag Go standard library and driver package dependencies as `[OK]`. → verify: `go test -v -run TestDomain ./components/sites/...`
2. Define the `AuthPolicy` struct inside `components/sites/sites.go`:
   ```go
   type AuthPolicy struct {
       Default        string   `json:"default"`          // "public" (default) or "protected"
       ProtectedPaths []string `json:"protected_paths"`  // e.g., ["/books/*"]
       PublicPaths    []string `json:"public_paths"`     // e.g., ["/login"]
       Accept         []string `json:"accept"`           // e.g., ["jwt", "site_key"]
   }
   ```
   Add `AuthPolicy *AuthPolicy` (or `AuthPolicy AuthPolicy`) field to the `Site` struct with JSON tag `auth_policy,omitempty`. Update database loading and site creation/updating logic to parse/serialize this field. → verify: `go test -v -run TestSiteAuthPolicyStruct ./components/sites`
3. Add REST API endpoints `GET /api/sites/:id/auth-policy` and `POST /api/sites/:id/auth-policy` to retrieve and update the site auth policy in `components/sites/sites.go`. In options, define an optional callback `UpdateAuthPolicy func(siteID string, policyJSON string)` that is invoked on successful policy updates to notify the proxy/router immediately. Write tests in `components/sites/sites_test.go` to verify policy retrieval and updates. → verify: `go test -v -run TestSiteAuthPolicyAPI ./components/sites`

## Verification Script (Step-by-Step)

1. Run `go test ./components/sites/...` to verify all domain/site schema and API tests pass.
2. Run `go build -o bigbase .` to verify compilation.

## Out of scope

- Enforcing the policy at the proxy layer (e71s02).
- Forwarding user identity headers to passthroughs (e71s03).
- Configuring the policy via MCP tools (e71s04).

## Risks

- DB schema lock or migration failures on production databases. Mitigated by using standard `ALTER TABLE ... ADD COLUMN` which is non-destructive and idempotent.
