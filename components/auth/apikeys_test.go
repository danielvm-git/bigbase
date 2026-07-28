package auth_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func setupAPIKeys(t *testing.T) (*auth.Auth, http.Handler, http.Handler, string, float64) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(a)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	pub := a.Handler()
	prot := a.ProtectedHandler()

	// Register user
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"keyholder@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	pub.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", regW.Code, regW.Body.String())
	}
	token := parseResponse(t, regW.Body.Bytes())["token"].(string)

	// Create org
	createOrgReq := httptest.NewRequest("POST", "/api/orgs",
		strings.NewReader(`{"name":"KeyOrg","slug":"key-org"}`))
	createOrgReq.Header.Set("Content-Type", "application/json")
	createOrgReq.Header.Set("Authorization", "Bearer "+token)
	createOrgW := httptest.NewRecorder()
	prot.ServeHTTP(createOrgW, createOrgReq)
	if createOrgW.Code != http.StatusCreated {
		t.Fatalf("create org: %d %s", createOrgW.Code, createOrgW.Body.String())
	}
	var orgResp map[string]any
	_ = json.NewDecoder(createOrgW.Body).Decode(&orgResp)
	orgID := orgResp["data"].(map[string]any)["id"].(float64)

	return a, pub, prot, token, orgID
}

func TestAPIKeys(t *testing.T) {
	t.Run("create_api_key", func(t *testing.T) {
		_, _, prot, token, orgID := setupAPIKeys(t)

		body := `{"name":"test-key","scopes":["read","write"]}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("create key: expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		data, ok := resp["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data, got: %v", resp)
		}
		// The raw key is returned only on creation
		if data["key"] == "" || data["key"] == nil {
			t.Error("expected non-empty API key in creation response")
		}
		if data["name"] != "test-key" {
			t.Errorf("expected name 'test-key', got %v", data["name"])
		}
	})

	t.Run("list_api_keys", func(t *testing.T) {
		_, _, prot, token, orgID := setupAPIKeys(t)

		// Create two keys
		for _, name := range []string{"key-a", "key-b"} {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
				strings.NewReader(fmt.Sprintf(`{"name":%q,"scopes":["read"]}`, name)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			prot.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("create %s: %d %s", name, w.Code, w.Body.String())
			}
		}

		listReq := httptest.NewRequest("GET", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID), nil)
		listReq.Header.Set("Authorization", "Bearer "+token)
		listW := httptest.NewRecorder()
		prot.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d: %s", listW.Code, listW.Body.String())
		}
		var listResp map[string]any
		if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		keys, ok := listResp["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got: %v", listResp)
		}
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(keys))
		}
		// Raw key must NOT be in list response (security)
		for _, k := range keys {
			km := k.(map[string]any)
			if _, hasKey := km["key"]; hasKey {
				t.Error("raw API key must not be returned in list response")
			}
		}
	})

	t.Run("delete_api_key", func(t *testing.T) {
		_, _, prot, token, orgID := setupAPIKeys(t)

		// Create a key
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
			strings.NewReader(`{"name":"del-key","scopes":["read"]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
		var createResp map[string]any
		_ = json.NewDecoder(w.Body).Decode(&createResp)
		keyID := createResp["data"].(map[string]any)["id"].(float64)

		// Delete it
		delReq := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/orgs/%.0f/api-keys/%.0f", orgID, keyID), nil)
		delReq.Header.Set("Authorization", "Bearer "+token)
		delW := httptest.NewRecorder()
		prot.ServeHTTP(delW, delReq)

		if delW.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d: %s", delW.Code, delW.Body.String())
		}
	})

	t.Run("resolve_api_key_to_org_context", func(t *testing.T) {
		a, _, prot, token, orgID := setupAPIKeys(t)

		// Create a key
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
			strings.NewReader(`{"name":"resolve-key","scopes":["read","write"]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
		var createResp map[string]any
		_ = json.NewDecoder(w.Body).Decode(&createResp)
		rawKey := createResp["data"].(map[string]any)["key"].(string)

		// Resolve the key to get org context
		resolvedOrgID, err := a.ResolveAPIKey(rawKey)
		if err != nil {
			t.Fatalf("resolve key: %v", err)
		}
		if resolvedOrgID != int64(orgID) {
			t.Fatalf("expected org_id=%.0f, got %d", orgID, resolvedOrgID)
		}
	})

	t.Run("resolve_invalid_key_returns_error", func(t *testing.T) {
		a, _, _, _, _ := setupAPIKeys(t)

		_, err := a.ResolveAPIKey("not-a-valid-key")
		if err == nil {
			t.Fatal("expected error for invalid key")
		}
	})

	t.Run("create_key_missing_name", func(t *testing.T) {
		_, _, prot, token, orgID := setupAPIKeys(t)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
			strings.NewReader(`{"scopes":["read"]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non_owner_cannot_create_key", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)
		d := db.New(db.Options{Path: ":memory:", Logger: logger})
		a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
		k.Register(a)
		k.Register(d)
		if err := k.Start(); err != nil {
			t.Fatalf("kernel start: %v", err)
		}
		t.Cleanup(func() { _ = k.Stop() })

		pub := a.Handler()
		prot := a.ProtectedHandler()

		// Owner registers and creates org
		regOwner := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"owner2@test.com","password":"secret123"}`))
		regOwner.Header.Set("Content-Type", "application/json")
		regOwnerW := httptest.NewRecorder()
		pub.ServeHTTP(regOwnerW, regOwner)
		ownerToken := parseResponse(t, regOwnerW.Body.Bytes())["token"].(string)

		createOrgReq := httptest.NewRequest("POST", "/api/orgs",
			strings.NewReader(`{"name":"Owned","slug":"owned-org"}`))
		createOrgReq.Header.Set("Content-Type", "application/json")
		createOrgReq.Header.Set("Authorization", "Bearer "+ownerToken)
		createOrgW := httptest.NewRecorder()
		prot.ServeHTTP(createOrgW, createOrgReq)
		if createOrgW.Code != http.StatusCreated {
			t.Fatalf("create org: %d", createOrgW.Code)
		}
		var orgResp map[string]any
		_ = json.NewDecoder(createOrgW.Body).Decode(&orgResp)
		orgID := orgResp["data"].(map[string]any)["id"].(float64)

		// Non-owner registers
		regOther := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"other2@test.com","password":"secret123"}`))
		regOther.Header.Set("Content-Type", "application/json")
		regOtherW := httptest.NewRecorder()
		pub.ServeHTTP(regOtherW, regOther)
		otherToken := parseResponse(t, regOtherW.Body.Bytes())["token"].(string)

		// Non-owner tries to create API key for owner's org
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
			strings.NewReader(`{"name":"hack","scopes":["read"]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+otherToken)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestAPIKeyOrgScoped(t *testing.T) {
	a, _, prot, token, orgID := setupAPIKeys(t)

	// Create API key
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
		strings.NewReader(`{"name":"scoped-key","scopes":["read"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	prot.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create api key: %d %s", w.Code, w.Body.String())
	}
	var createResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&createResp)
	rawKey := createResp["data"].(map[string]any)["key"].(string)

	// Use the API key through middleware to verify org_id is set
	var gotOrgID int64
	var gotOK bool
	mw := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrgID, gotOK = auth.OrgIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	authReq := httptest.NewRequest("GET", "/api/protected", nil)
	authReq.Header.Set("Authorization", "Bearer "+rawKey)
	authW := httptest.NewRecorder()
	mw.ServeHTTP(authW, authReq)

	if authW.Code != http.StatusOK {
		t.Fatalf("middleware: expected 200, got %d: %s", authW.Code, authW.Body.String())
	}
	if !gotOK {
		t.Fatal("expected OrgIDFromContext to return true with API key")
	}
	if gotOrgID != int64(orgID) {
		t.Fatalf("expected org_id=%.0f, got %d", orgID, gotOrgID)
	}
}

func TestRequireScopes_MatchingScopeAllowed(t *testing.T) {
	a, _, prot, token, orgID := setupAPIKeys(t)

	// Create API key with sites:write scope
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
		strings.NewReader(`{"name":"write-key","scopes":["sites:write"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	prot.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create api key: %d %s", w.Code, w.Body.String())
	}
	var createResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&createResp)
	rawKey := createResp["data"].(map[string]any)["key"].(string)

	// Use RequireScopes middleware directly
	handler := auth.RequireScopes("sites:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	mw := a.Middleware(handler)

	req2 := httptest.NewRequest("POST", "/api/sites/test-site/deploy-keys", nil)
	req2.Header.Set("Authorization", "Bearer "+rawKey)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 with matching scope, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRequireScopes_MissingScopeDenied(t *testing.T) {
	a, _, prot, token, orgID := setupAPIKeys(t)

	// Create API key with read-only scope
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
		strings.NewReader(`{"name":"read-key","scopes":["sites:read"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	prot.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create api key: %d %s", w.Code, w.Body.String())
	}
	var createResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&createResp)
	rawKey := createResp["data"].(map[string]any)["key"].(string)

	// Require sites:write — should be denied
	handler := auth.RequireScopes("sites:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	mw := a.Middleware(handler)

	req2 := httptest.NewRequest("DELETE", "/api/sites/test-site/deploy-keys/123", nil)
	req2.Header.Set("Authorization", "Bearer "+rawKey)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with missing scope, got %d: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "insufficient scopes") {
		t.Fatalf("expected 'insufficient scopes' error, got: %s", w2.Body.String())
	}
}

func TestRequireScopes_UnscopedKeyAllowed(t *testing.T) {
	a, _, prot, token, orgID := setupAPIKeys(t)

	// Create API key with no scopes
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/api-keys", orgID),
		strings.NewReader(`{"name":"unscoped-key"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	prot.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create api key: %d %s", w.Code, w.Body.String())
	}
	var createResp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&createResp)
	rawKey := createResp["data"].(map[string]any)["key"].(string)

	// Unscoped key should pass through RequireScopes (backward compat)
	handler := auth.RequireScopes("sites:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	mw := a.Middleware(handler)

	req2 := httptest.NewRequest("POST", "/api/sites/test-site/deploy-keys", nil)
	req2.Header.Set("Authorization", "Bearer "+rawKey)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for unscoped key (backward compat), got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRequireScopes_JWTAuthBypassesScopeCheck(t *testing.T) {
	_, _, prot, token, _ := setupAPIKeys(t)

	// JWT-authenticated request (no scopes in context) should pass through.
	// ProtectedHandler already wires RequireScopes on org write routes.
	req := httptest.NewRequest("POST", "/api/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	prot.ServeHTTP(w, req)

	// Note: This will hit the real handler which expects JSON body,
	// but the RequireScopes layer should let it through.
	// The 400 from the handler body parse is expected — what matters
	// is it didn't get 403 from RequireScopes.
	if w.Code == http.StatusForbidden {
		t.Fatalf("JWT auth should bypass RequireScopes, got 403")
	}
}
