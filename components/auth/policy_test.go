package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
)

// This file tests the declarative Policy Gate introduced by issue #43
// ("Policy Gate for route access control"). The Policy type lets a route
// declare its access requirements (auth / admin / org-scoped / scopes) at
// registration time; Policy.Wrap then enforces them, so a handler that forgets
// to call a check is caught structurally rather than relying on each author
// remembering to thread middleware by hand (the exact failure mode behind two
// critical IDOR regressions — see BUG-2026-07-28T000002 and the #180 matrix).
//
// These are PURE UNIT tests for the Policy enforcement layer. They do not boot
// a database or the full JWT middleware; instead they use the exported test
// helpers auth.WithUserRole / auth.WithOrgID / auth.WithUserID to populate the
// request context the same way auth.Middleware does in production, then assert
// the policy's enforcement decision (allow vs 401/403 with a structured code).
// This is the leverage the issue calls out: handler-level policy tests don't
// need to fake real auth tokens.
//
// Composition with the existing cross-tenant route matrix (#180) is covered by
// the integration adoption: /api/sql (RequireAdmin) and the deploy-key creation
// route (RequireAuth + scopes), which are migrated to Policy.Wrap in main.go.

// okHandler is the downstream handler a Policy wraps. It sets a sentinel
// header so tests can tell whether the policy ALLOWED the request through.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Policy-Passed", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
}

// decodePolicyErr pulls the structured error body the policy emits on denial.
func decodePolicyErr(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode error body %q: %v", body, err)
	}
	return m
}

// withContext returns a request whose context carries the given values, mirroring
// what auth.Middleware sets in production (role via WithUserRole, org via
// WithOrgID). A nil userID means "no authenticated user" (unauthenticated case).
func withContext(r *http.Request, userID int64, role string, orgID int64) *http.Request {
	ctx := r.Context()
	if userID != 0 {
		ctx = auth.WithUserID(ctx, userID)
	}
	if role != "" {
		ctx = auth.WithUserRole(ctx, role)
	}
	if orgID != 0 {
		ctx = auth.WithOrgID(ctx, orgID)
	}
	return r.WithContext(ctx)
}

// --- RequireAuth -------------------------------------------------------------

func TestPolicy_RequireAuth_AllowsAuthenticated(t *testing.T) {
	h := auth.RequireAuth().Wrap(okHandler())

	req := withContext(httptest.NewRequest(http.MethodGet, "/x", nil), 42, "user", 7)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RequireAuth: authenticated caller should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Policy-Passed") != "1" {
		t.Fatalf("RequireAuth: downstream handler not reached")
	}
}

func TestPolicy_RequireAuth_RejectsUnauthenticated(t *testing.T) {
	h := auth.RequireAuth().Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("RequireAuth: unauthenticated caller should get 401, got %d (body=%s)", w.Code, w.Body.String())
	}
	if e := decodePolicyErr(t, w.Body.Bytes()); e["code"] != "auth_required" {
		t.Fatalf("RequireAuth: want code=auth_required, got %q (body=%s)", e["code"], w.Body.String())
	}
}

// --- RequireAdmin ------------------------------------------------------------

func TestPolicy_RequireAdmin_AllowsAdmin(t *testing.T) {
	h := auth.RequireAdmin().Wrap(okHandler())

	req := withContext(httptest.NewRequest(http.MethodGet, "/x", nil), 1, "admin", 1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RequireAdmin: admin should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestPolicy_RequireAdmin_RejectsNonAdmin(t *testing.T) {
	h := auth.RequireAdmin().Wrap(okHandler())

	req := withContext(httptest.NewRequest(http.MethodGet, "/x", nil), 2, "user", 7)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("RequireAdmin: non-admin should get 403, got %d (body=%s)", w.Code, w.Body.String())
	}
	if e := decodePolicyErr(t, w.Body.Bytes()); e["code"] != "forbidden" {
		t.Fatalf("RequireAdmin: want code=forbidden, got %q", e["code"])
	}
}

func TestPolicy_RequireAdmin_RejectsUnauthenticated(t *testing.T) {
	h := auth.RequireAdmin().Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("RequireAdmin: unauthenticated should get 401 (auth checked before admin), got %d", w.Code)
	}
}

// --- RequireOrgScoped --------------------------------------------------------

func TestPolicy_RequireOrgScoped_AllowsWithOrg(t *testing.T) {
	h := auth.RequireOrgScoped().Wrap(okHandler())

	req := withContext(httptest.NewRequest(http.MethodGet, "/x", nil), 5, "user", 9)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RequireOrgScoped: caller with org_id should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestPolicy_RequireOrgScoped_RejectsMissingOrg(t *testing.T) {
	h := auth.RequireOrgScoped().Wrap(okHandler())

	// Authenticated (user_id + role set) but NO org_id — the IDOR precondition.
	req := withContext(httptest.NewRequest(http.MethodGet, "/x", nil), 5, "user", 0)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("RequireOrgScoped: caller with no org_id should get 403, got %d (body=%s)", w.Code, w.Body.String())
	}
	if e := decodePolicyErr(t, w.Body.Bytes()); e["code"] != "no_organization" {
		t.Fatalf("RequireOrgScoped: want code=no_organization, got %q", e["code"])
	}
}

func TestPolicy_RequireOrgScoped_RejectsUnauthenticated(t *testing.T) {
	h := auth.RequireOrgScoped().Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("RequireOrgScoped: unauthenticated should get 401, got %d", w.Code)
	}
}

// --- Composition (RequireAuth + scopes via RequireScopes) --------------------

func TestPolicy_RequireScopes_AllowsMatchingScope(t *testing.T) {
	h := auth.RequireScopes("sites:write").Wrap(okHandler())

	ctx := auth.WithOrgKeyScopes(context.Background(), []string{"sites:write"})
	req := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RequireScopes: matching scope should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestPolicy_RequireScopes_RejectsMissingScope(t *testing.T) {
	h := auth.RequireScopes("sites:write").Wrap(okHandler())

	ctx := auth.WithOrgKeyScopes(context.Background(), []string{"sites:read"})
	req := httptest.NewRequest(http.MethodDelete, "/x", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("RequireScopes: missing scope should get 403, got %d (body=%s)", w.Code, w.Body.String())
	}
	if e := decodePolicyErr(t, w.Body.Bytes()); e["code"] != "insufficient_scopes" {
		t.Fatalf("RequireScopes: want code=insufficient_scopes, got %q", e["code"])
	}
}

// Unscoped keys (no scopes in context, e.g. JWT auth) pass through for backward
// compatibility — mirrors the legacy RequireScopes middleware contract.
func TestPolicy_RequireScopes_UnscopedKeyPassesThrough(t *testing.T) {
	h := auth.RequireScopes("sites:write").Wrap(okHandler())

	// No scopes set in context (JWT auth path).
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RequireScopes: unscoped/JWT caller should pass through, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// --- Composite policies (With + RequireScopes) -------------------------------

// RequireOrgScoped + scopes is the realistic org write route: needs org
// isolation AND a permitted scope. This proves the declarative model composes.
func TestPolicy_OrgScopedWithScopes_RequiresBoth(t *testing.T) {
	h := auth.RequireOrgScoped().
		WithScopes("orgs:write").
		Wrap(okHandler())

	t.Run("org_and_scope_ok", func(t *testing.T) {
		ctx := context.Background()
		ctx = auth.WithOrgID(ctx, 3)
		ctx = auth.WithOrgKeyScopes(ctx, []string{"orgs:write"})
		req := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("org+scope: should pass, got %d (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("org_missing_scope_denied", func(t *testing.T) {
		ctx := context.Background()
		ctx = auth.WithOrgID(ctx, 3)
		ctx = auth.WithOrgKeyScopes(ctx, []string{"sites:read"})
		req := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("org+scope: missing scope should get 403, got %d", w.Code)
		}
	})

	t.Run("scope_but_no_org_denied", func(t *testing.T) {
		ctx := context.Background()
		ctx = auth.WithOrgKeyScopes(ctx, []string{"orgs:write"})
		req := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("org+scope: no org_id should get 403 (org-scoping checked), got %d", w.Code)
		}
	})
}

// --- Bare zero-value Policy (none required) ---------------------------------

// A zero-value Policy (RequireNone) is a pass-through. This lets a registration
// helper require a Policy for every route while still allowing public routes.
func TestPolicy_None_AllowsAll(t *testing.T) {
	h := auth.RequireNone().Wrap(okHandler())

	// No auth context at all.
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RequireNone: unauthenticated should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
}
