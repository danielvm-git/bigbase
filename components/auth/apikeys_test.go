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

		body := fmt.Sprintf(`{"name":"test-key","scopes":["read","write"]}`)
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
