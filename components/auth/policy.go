package auth

import (
	"net/http"

	"github.com/danielvm/bigbase/kernel"
)

// policy.go implements the declarative Policy Gate for route access control
// (issue #43, "Deepen: Policy Gate for route access control").
//
// PROBLEM
//
// Auth in BigBase is hand-threaded as .Middleware() chains in main.go and the
// component ProtectedHandler() muxes, and individual handlers re-implement
// access checks (auth.OrgIDFromContext here, auth.UserRoleFromContext there).
// A new handler can silently skip an org-isolation or role check — nothing
// enforces completeness. This is the exact failure mode behind two critical
// IDOR regressions (see BUG-2026-07-28T000002 and the #180 cross-tenant route
// matrix). The "policy" today is just sequential function calls an author has
// to remember in the right order.
//
// SOLUTION
//
// A route instead carries a declarative Policy that names its access
// requirements up front:
//
//	authLevel: none | auth | admin   (PolicyNone / PolicyAuth / PolicyAdmin)
//	orgScoped: bool                  (PolicyOrgScoped)
//	scopes:    []string              (PolicyScopes / Policy.WithScopes)
//
// Policy.Wrap(handler) returns an http.Handler that enforces the declared
// policy at request time, using the EXISTING context helpers populated by
// auth.Middleware — UserIDFromContext, UserRoleFromContext, OrgIDFromContext,
// OrgKeyScopesFromContext. No new auth machinery is introduced; this layer is
// a declarative composition of the checks that already existed ad hoc.
//
// Why a Policy type rather than another middleware factory: the requirements
// live as DATA on the route registration, so the access rule for any route can
// be read in one place and a registration helper can refuse to mount a route
// that hasn't declared one. The 401/403 responses use the same structured
// {error, code} shape as the rest of the auth layer (kernel.WriteJSON).
//
// SCOPE (this change)
//
// The Policy Gate is ADDITIVE and OPTIONAL. It does not rewrite the existing
// routing table — that is a separate, larger migration tracked below. New and
// recently-changed routes opt in and get correct auth by construction; the
// existing hand-threaded routes keep working unchanged and stay guarded by the
// #180 cross-tenant route matrix test until they migrate. See the MIGRATION
// note at the bottom of this file for the plan.
//
// Relationship to #180: #43 PREVENTS missing checks at registration time (a
// Policy-decorated route cannot forget its check); #180 CATCHES them at test
// time (the matrix fails CI if an org-scoped route lacks an ownership check).
// Together they make the IDOR class structural rather than per-author.

// authLevel enumerates the authentication tiers a Policy can require.
type authLevel int

const (
	authNone   authLevel = iota // no auth required (public route)
	authRequired                // any authenticated caller
	authAdmin                   // authenticated caller with role "admin"
)

// Policy is a declarative description of a route's access requirements.
//
// Construct one with PolicyNone / PolicyAuth / PolicyAdmin /
// PolicyOrgScoped / PolicyScopes, then either call Wrap to get an enforcing
// http.Handler, or compose further with WithScopes before wrapping.
//
// A zero-value Policy is equivalent to PolicyNone() (everything allowed) so a
// registration helper that insists on a Policy for every route can still mount
// genuinely public routes by passing PolicyNone().
type Policy struct {
	level     authLevel
	orgScoped bool
	scopes    []string
}

// PolicyNone returns a Policy that imposes no access requirements. Use it for
// public routes when a registration helper mandates a Policy per route.
func PolicyNone() Policy { return Policy{level: authNone} }

// PolicyAuth returns a Policy that requires an authenticated caller
// (a user_id present in the request context, set by auth.Middleware). Any
// authenticated caller is accepted; role and org are not checked. Combine with
// PolicyOrgScoped / WithScopes to add org isolation or API-key scope limits.
func PolicyAuth() Policy { return Policy{level: authRequired} }

// PolicyAdmin returns a Policy that requires an authenticated caller whose
// role is "admin" (set from the JWT role claim by auth.Middleware). Use it for
// platform-wide administrative routes such as /api/sql.
func PolicyAdmin() Policy { return Policy{level: authAdmin} }

// PolicyOrgScoped returns a Policy that requires an authenticated caller with
// a nonzero org_id in the request context. This is the structural guard
// against the IDOR precondition: a tenant-scoped handler must never run for a
// request that has no resolved org. Combine with WithScopes for org write
// routes that are also gated by API-key scopes.
func PolicyOrgScoped() Policy { return Policy{level: authRequired, orgScoped: true} }

// PolicyScopes returns a Policy that enforces API-key scope restrictions: at
// least one of the required scopes must be present in the resolved API key's
// scopes. Requests authenticated via JWT or a site deploy key (no scopes in
// context) pass through, mirroring the legacy RequireScopes middleware
// contract for backward compatibility. Use this on org/site write routes.
func PolicyScopes(required ...string) Policy { return Policy{level: authNone, scopes: append([]string(nil), required...)} }

// WithScopes returns a copy of p that ALSO requires at least one of the given
// API-key scopes. It composes scope enforcement onto an auth/org policy, e.g.
// PolicyOrgScoped().WithScopes("orgs:write").Wrap(h) requires both org
// isolation and a permitted scope.
func (p Policy) WithScopes(required ...string) Policy {
	out := p
	out.scopes = append([]string(nil), p.scopes...)
	out.scopes = append(out.scopes, required...)
	return out
}

// Wrap returns an http.Handler that enforces p before delegating to next.
// Checks run in the order: auth -> role -> org -> scopes, so a denial reports
// the most specific failing precondition. Each failure writes a structured
// {error, code} JSON body via kernel.WriteJSON, matching the conventions used
// by auth.Middleware and the legacy RequireScopes/RequireAdmin middleware.
func (p Policy) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Authentication. A user_id in context means auth.Middleware (JWT or
		// org API key) accepted the caller. Its absence means unauthenticated.
		// For authNone policies we skip straight to scope checks (public route).
		userID, hasUser := UserIDFromContext(ctx)
		authenticated := hasUser && userID != 0
		if p.level != authNone && !authenticated {
			kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "authorization required",
				"code":  "auth_required",
			})
			return
		}

		// 2. Admin role. Only meaningful when auth was required.
		if p.level == authAdmin {
			role, ok := UserRoleFromContext(ctx)
			if !ok || role != "admin" {
				kernel.WriteJSON(w, http.StatusForbidden, map[string]string{
					"error": "forbidden",
					"code":  "forbidden",
				})
				return
			}
		}

		// 3. Org scoping. Reject requests that have no resolved org_id: a
		// tenant-scoped handler must never execute without tenancy context.
		// (auth.Middleware already fails closed on JWTs missing org_id, but
		// enforcing it here makes the guarantee local to the route, not just
		// to the global middleware chain an author has to remember to mount.)
		if p.orgScoped {
			orgID, ok := OrgIDFromContext(ctx)
			if !ok || orgID == 0 {
				kernel.WriteJSON(w, http.StatusForbidden, map[string]string{
					"error": "no organization",
					"code":  "no_organization",
				})
				return
			}
		}

		// 4. API-key scopes. Mirrors the legacy RequireScopes contract: only
		// scoped API keys (scopes present in context) are restricted; JWT and
		// site-key callers pass through.
		if len(p.scopes) > 0 {
			scopes, ok := OrgKeyScopesFromContext(ctx)
			if ok && len(scopes) > 0 {
				if !hasAnyScope(scopes, p.scopes) {
					kernel.WriteJSON(w, http.StatusForbidden, map[string]string{
						"error": "insufficient scopes",
						"code":  "insufficient_scopes",
					})
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// hasAnyScope reports whether any of the caller's scopes matches a required
// scope. Extracted from the inline loop in the legacy RequireScopes middleware
// so both code paths share one definition of "scope match".
func hasAnyScope(have, required []string) bool {
	for _, h := range have {
		for _, need := range required {
			if h == need {
				return true
			}
		}
	}
	return false
}

// Middleware returns p as a standard middleware function
// (func(http.Handler) http.Handler), so it composes with existing middleware
// chains in main.go (mComp.Middleware(authComp.Middleware(...))) and inside
// component muxes. This is the integration point for migrating hand-threaded
// RequireScopes / RequireAdmin chains to a declarative Policy: replace the
// ad-hoc middleware with the equivalent Policy constructor's Middleware.
func (p Policy) Middleware(next http.Handler) http.Handler { return p.Wrap(next) }

/*
MIGRATION PLAN (remaining routes)

The Policy Gate is adopted on representative routes in this change to prove it
composes with the existing auth context helpers:

  - main.go: /api/sql is now gated by auth.PolicyAdmin().Middleware(...) (was
    the bare auth.RequireAdmin middleware). Identical runtime behaviour; the
    access rule is now data on the registration.
  - components/auth/middleware.go (ProtectedHandler): the deploy-key creation
    route POST /api/sites/{id}/deploy-keys keeps its rate limiter but swaps the
    RequireScopes("sites:write","deploy") middleware for the equivalent
    Policy: auth.PolicyScopes("sites:write", "deploy").Middleware(...).

These two cover the two composition shapes the full migration needs: an
admin-gated platform route, and an org/site write route gated by scopes (and,
where relevant, org-scoping).

Migrating the rest is mechanical and deliberately deferred to avoid
destabilising the routing table in one pass:

  1. Org write routes (POST/PATCH/DELETE /api/orgs..., api-keys, invites) in
     components/auth/middleware.go: replace RequireScopes("orgs:write") with
     auth.PolicyOrgScoped().WithScopes("orgs:write").Middleware(...). The
     org-scoping check is currently implicit (auth.Middleware fails closed on
     JWTs missing org_id); making it explicit per route is the structural win.
  2. The remaining ProtectedHandler() routes (users, me, identities, org reads)
     migrate to auth.PolicyAuth() (or PolicyOrgScoped for org-scoped reads).
  3. Component handlers (sites, deploy, git, forge, functions, messaging) that
     mount their own mux behind authComp.Middleware adopt PolicyOrgScoped on
     their org-scoped sub-routes. The per-handler SQL ownership checks
     (requireSiteOwnership / verifyDeploymentOwnership) STAY — the Policy Gate
     is an access precondition, not a replacement for row-level ownership
     probes. It removes the chance of a handler running with no tenancy
     context at all; the ownership check still enforces cross-tenant denial.

BACKSTOP UNTIL MIGRATION: the #180 cross-tenant route matrix
(components/auth/route_matrix_test.go) fails CI if any org-scoped route lacks
an ownership check, so unmigrated routes remain guarded at test time. The
matrix and the Policy Gate are complementary: #43 prevents missing checks at
registration time; #180 catches them at test time.

A future hardening (out of scope here) can add a registration helper that
REFUSES to mount a route without a declared Policy, turning "optional" into
"required by construction" once the migration is complete.
*/
