package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (a *Auth) ProtectedHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/users", a.handleUsers)
	mux.HandleFunc("DELETE /api/auth/users/{id}", a.handleUserByID)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)
	mux.HandleFunc("PATCH /api/auth/me", a.handleUpdateMe)
	mux.HandleFunc("GET /api/auth/me/identities", a.handleListIdentities)
	mux.HandleFunc("POST /api/auth/me/identities", a.handleLinkIdentity)
	mux.HandleFunc("DELETE /api/auth/me/identities/{provider}", a.handleUnlinkIdentity)
	mux.Handle("POST /api/orgs", RequireScopes("orgs:write")(http.HandlerFunc(a.handleCreateOrg)))
	mux.HandleFunc("GET /api/orgs", a.handleListOrgs)
	mux.HandleFunc("GET /api/orgs/{id}", a.handleGetOrg)
	mux.Handle("PATCH /api/orgs/{id}", RequireScopes("orgs:write")(http.HandlerFunc(a.handleUpdateOrg)))
	mux.Handle("DELETE /api/orgs/{id}", RequireScopes("orgs:write")(http.HandlerFunc(a.handleDeleteOrg)))
	mux.Handle("POST /api/orgs/{id}/invites", RequireScopes("orgs:write")(http.HandlerFunc(a.handleCreateInvite)))
	mux.HandleFunc("POST /api/orgs/{id}/invites/{token}/accept", a.handleAcceptInvite)
	mux.HandleFunc("GET /api/orgs/{id}/members", a.handleListMembers)
	mux.Handle("POST /api/orgs/{id}/api-keys", RequireScopes("orgs:write")(http.HandlerFunc(a.handleCreateAPIKey)))
	mux.HandleFunc("GET /api/orgs/{id}/api-keys", a.handleListAPIKeys)
	mux.Handle("DELETE /api/orgs/{id}/api-keys/{keyID}", RequireScopes("orgs:write")(http.HandlerFunc(a.handleDeleteAPIKey)))
	// Deploy-key routes with rate-limited POST (10 per site per hour per user)
	rl := NewRateLimiter(RateLimiterConfig{
		IPLimit:      3,
		IPWindow:     time.Minute,
		UserLimit:    10,
		UserWindow:   time.Hour,
		CleanupEvery: 10 * time.Minute,
	})
	mux.Handle("POST /api/sites/{id}/deploy-keys", rl.Middleware(RequireScopes("sites:write", "deploy")(http.HandlerFunc(a.handleCreateSiteKey))))
	mux.HandleFunc("GET /api/sites/{id}/deploy-keys", a.handleListSiteKeys)
	mux.Handle("DELETE /api/sites/{id}/deploy-keys/{keyID}", RequireScopes("sites:write")(http.HandlerFunc(a.handleRevokeSiteKey)))
	mux.HandleFunc("POST /api/auth/logout-all", a.handleLogoutAll)
	return a.Middleware(mux)
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			if c, err := r.Cookie("token"); err == nil {
				token = c.Value
			}
		}
		if token == "" {
			kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization required"})
			return
		}

		// API key authentication — resolve org_id and skip JWT checks.
		if strings.HasPrefix(token, "bb_") {
			if strings.HasPrefix(token, "bb_dep_") {
				siteID, err := a.ResolveSiteKey(token)
				if err != nil {
					a.logger.Error("site key resolution failed", "error", err)
					kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid site key"})
					return
				}
				ctx := kernel.WithSiteID(r.Context(), siteID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			orgID, scopes, err := a.ResolveOrgKey(token)
			if err != nil {
				a.logger.Error("api key resolution failed", "error", err)
				kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
				return
			}
			ctx := context.WithValue(r.Context(), ctxOrgID, orgID)
			ctx = kernel.WithOrgID(ctx, orgID)
			ctx = context.WithValue(ctx, ctxOrgKeyScopes, scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := verifyJWT(token, a.secret)
		if err != nil {
			a.logger.Error("jwt verification failed", "error", err)
			kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxUserEmail, claims.Email)
		ctx = context.WithValue(ctx, ctxUserRole, claims.Role)
		ctx = context.WithValue(ctx, ctxOrgID, claims.OrgID)
		if claims.OrgID != 0 {
			ctx = kernel.WithOrgID(ctx, claims.OrgID)
		}

		// Anonymous tokens bypass org isolation and email verification
		// but are restricted to read-only methods.
		if claims.Role == "anonymous" {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r.WithContext(ctx))
			default:
				kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "write access not allowed for anonymous users"})
			}
			return
		}

		// Block unverified users when email verification is enabled.
		verified, vErr := a.isEmailVerified(ctx, claims.UserID)
		if vErr != nil {
			a.logger.Error("check email verified", "user_id", claims.UserID, "error", vErr)
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if !verified {
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "email not verified"})
			return
		}

		// Fail closed: tokens missing org_id are rejected.
		if claims.OrgID == 0 {
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "no organization"})
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxUserID).(int64)
	return id, ok
}

func UserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(ctxUserEmail).(string)
	return email, ok
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(ctxUserRole).(string)
	return role, ok
}

func OrgIDFromContext(ctx context.Context) (int64, bool) {
	orgID, ok := ctx.Value(ctxOrgID).(int64)
	return orgID, ok
}

// WithOrgID returns a new context with the given org_id value.
// Useful for testing handlers that depend on org-scoped auth context.
// Also sets kernel.WithOrgID so components that read kernel.OrgIDFromContext
// (storage, monitoring) see the same org as auth middleware in production.
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	ctx = context.WithValue(ctx, ctxOrgID, orgID)
	return kernel.WithOrgID(ctx, orgID)
}

// WithUserRole returns a new context with the given user role, mirroring what
// the JWT auth middleware sets from the token's role claim. Useful for tests
// that exercise role-gated behaviour (e.g. the legacy org_id=0 site visibility
// carve-out, which is now admin-only — see BUG-2026-07-28T000002).
func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxUserRole, role)
}

// OrgKeyScopesFromContext extracts scopes from the org API key that
// the auth middleware injected via ResolveOrgKey. Returns an empty
// slice and false when no scopes are present in the context.
func OrgKeyScopesFromContext(ctx context.Context) ([]string, bool) {
	scopes, ok := ctx.Value(ctxOrgKeyScopes).([]string)
	return scopes, ok
}

// RequireScopes returns middleware that enforces API key scope restrictions.
// At least one of the required scopes must be present in the request's
// resolved API key scopes. Requests authenticated via JWT or site key
// (no scopes in context) pass through for backward compatibility.
// Requests from scoped API keys missing all required scopes get 403.
func RequireScopes(required ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, ok := OrgKeyScopesFromContext(r.Context())
			if !ok || len(scopes) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			for _, have := range scopes {
				for _, need := range required {
					if have == need {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient scopes"})
		})
	}
}

// RequireAdmin returns middleware that rejects non-admin requests with 403.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := UserRoleFromContext(r.Context())
		if !ok || role != "admin" {
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
