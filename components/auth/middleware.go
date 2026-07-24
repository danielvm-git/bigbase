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
	mux.HandleFunc("POST /api/orgs", a.handleCreateOrg)
	mux.HandleFunc("GET /api/orgs", a.handleListOrgs)
	mux.HandleFunc("GET /api/orgs/{id}", a.handleGetOrg)
	mux.HandleFunc("PATCH /api/orgs/{id}", a.handleUpdateOrg)
	mux.HandleFunc("DELETE /api/orgs/{id}", a.handleDeleteOrg)
	mux.HandleFunc("POST /api/orgs/{id}/invites", a.handleCreateInvite)
	mux.HandleFunc("POST /api/orgs/{id}/invites/{token}/accept", a.handleAcceptInvite)
	mux.HandleFunc("GET /api/orgs/{id}/members", a.handleListMembers)
	mux.HandleFunc("POST /api/orgs/{id}/api-keys", a.handleCreateAPIKey)
	mux.HandleFunc("GET /api/orgs/{id}/api-keys", a.handleListAPIKeys)
	mux.HandleFunc("DELETE /api/orgs/{id}/api-keys/{keyID}", a.handleDeleteAPIKey)
	// Deploy-key routes with rate-limited POST (10 per site per hour per user)
	rl := NewRateLimiter(RateLimiterConfig{
		IPLimit:      3,
		IPWindow:     time.Minute,
		UserLimit:    10,
		UserWindow:   time.Hour,
		CleanupEvery: 10 * time.Minute,
	})
	mux.Handle("POST /api/sites/{id}/deploy-keys", rl.Middleware(http.HandlerFunc(a.handleCreateSiteKey)))
	mux.HandleFunc("GET /api/sites/{id}/deploy-keys", a.handleListSiteKeys)
	mux.HandleFunc("DELETE /api/sites/{id}/deploy-keys/{keyID}", a.handleRevokeSiteKey)
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
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	return context.WithValue(ctx, ctxOrgID, orgID)
}

// OrgKeyScopesFromContext extracts scopes from the org API key that
// the auth middleware injected via ResolveOrgKey. Returns an empty
// slice and false when no scopes are present in the context.
func OrgKeyScopesFromContext(ctx context.Context) ([]string, bool) {
	scopes, ok := ctx.Value(ctxOrgKeyScopes).([]string)
	return scopes, ok
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
