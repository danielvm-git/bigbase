package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
)

// setupAuthWithSite creates a full auth test environment with a pre-seeded site.
func setupAuthWithSite(t *testing.T) (*auth.Auth, http.Handler, http.Handler, string) {
	t.Helper()
	a, handler, protected, d := setupAuthWithDB(t)

	_, err := d.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("create sites table: %v", err)
	}

	siteID := "test-site"
	_, err = d.Exec(`INSERT INTO sites (id, name, git_repo_id) VALUES (?, 'app', 'repo-1')`, siteID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	return a, handler, protected, siteID
}

func TestSiteKeyHandlers(t *testing.T) {
	t.Run("create_site_key", func(t *testing.T) {
		_, handler, protected, siteID := setupAuthWithSite(t)

		// Register and login
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"sitekey@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: expected 201, got %d: %s", regW.Code, regW.Body.String())
		}

		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"sitekey@test.com","password":"secret123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		var loginResp map[string]any
		if err := json.NewDecoder(loginW.Body).Decode(&loginResp); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		token, _ := loginResp["token"].(string)
		if token == "" {
			t.Fatal("expected non-empty token")
		}

		// Create a deploy key
		createReq := httptest.NewRequest("POST", "/api/sites/"+siteID+"/deploy-keys",
			strings.NewReader(`{"name":"ci-bot"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		protected.ServeHTTP(createW, createReq)

		if createW.Code != http.StatusCreated {
			t.Fatalf("create site key: expected 201, got %d: %s", createW.Code, createW.Body.String())
		}

		var createResp map[string]any
		if err := json.NewDecoder(createW.Body).Decode(&createResp); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		data, ok := createResp["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data in response, got %v", createResp)
		}
		if data["key"] == "" {
			t.Fatal("expected non-empty raw key in create response")
		}
		if !strings.HasPrefix(data["key"].(string), "bb_dep_") {
			t.Fatalf("expected key to start with bb_dep_, got %q", data["key"])
		}
	})

	t.Run("list_site_keys_returns_metadata_only", func(t *testing.T) {
		_, handler, protected, siteID := setupAuthWithSite(t)

		// Register and login
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"list-keys@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)

		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"list-keys@test.com","password":"secret123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		var loginResp map[string]any
		if err := json.NewDecoder(loginW.Body).Decode(&loginResp); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		token, _ := loginResp["token"].(string)

		// Create two keys
		for _, name := range []string{"ci-bot", "deploy-bot"} {
			req := httptest.NewRequest("POST", "/api/sites/"+siteID+"/deploy-keys",
				strings.NewReader(`{"name":"`+name+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			protected.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("create %s: expected 201, got %d: %s", name, w.Code, w.Body.String())
			}
		}

		// List keys
		listReq := httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-keys", nil)
		listReq.Header.Set("Authorization", "Bearer "+token)
		listW := httptest.NewRecorder()
		protected.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("list site keys: expected 200, got %d: %s", listW.Code, listW.Body.String())
		}

		var listResp map[string]any
		if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		keys, ok := listResp["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got %v", listResp)
		}
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(keys))
		}

		for _, k := range keys {
			key := k.(map[string]any)
			if _, hasRaw := key["key"]; hasRaw {
				t.Fatal("list endpoint must NOT return raw key tokens")
			}
			if _, hasKeyID := key["key_id"]; !hasKeyID {
				t.Fatal("expected key_id in list response")
			}
			if _, hasName := key["name"]; !hasName {
				t.Fatal("expected name in list response")
			}
		}
	})

	t.Run("revoke_site_key", func(t *testing.T) {
		_, handler, protected, siteID := setupAuthWithSite(t)

		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"revoke@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)

		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"revoke@test.com","password":"secret123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		var loginResp map[string]any
		if err := json.NewDecoder(loginW.Body).Decode(&loginResp); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		token, _ := loginResp["token"].(string)

		// Create a key
		createReq := httptest.NewRequest("POST", "/api/sites/"+siteID+"/deploy-keys",
			strings.NewReader(`{"name":"to-revoke"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		protected.ServeHTTP(createW, createReq)

		// Get key ID from list
		listReq := httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-keys", nil)
		listReq.Header.Set("Authorization", "Bearer "+token)
		listW := httptest.NewRecorder()
		protected.ServeHTTP(listW, listReq)
		var listResp map[string]any
		if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		keys := listResp["data"].([]any)
		if len(keys) < 1 {
			t.Fatal("expected at least 1 key in list")
		}
		keyID := keys[0].(map[string]any)["key_id"].(string)

		// Revoke
		revokeReq := httptest.NewRequest("DELETE", "/api/sites/"+siteID+"/deploy-keys/"+keyID, nil)
		revokeReq.Header.Set("Authorization", "Bearer "+token)
		revokeW := httptest.NewRecorder()
		protected.ServeHTTP(revokeW, revokeReq)

		if revokeW.Code != http.StatusOK {
			t.Fatalf("revoke: expected 200, got %d: %s", revokeW.Code, revokeW.Body.String())
		}
	})

	t.Run("requires_authentication", func(t *testing.T) {
		_, _, protected, siteID := setupAuthWithSite(t)

		// No auth token
		req := httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-keys", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("bb_dep_token_rejected", func(t *testing.T) {
		_, handler, protected, siteID := setupAuthWithSite(t)

		// Register and login
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"bbdep-test@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)

		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"bbdep-test@test.com","password":"secret123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		var loginResp map[string]any
		if err := json.NewDecoder(loginW.Body).Decode(&loginResp); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		token, _ := loginResp["token"].(string)

		// Create a deploy key to get a bb_dep_ token
		createReq := httptest.NewRequest("POST", "/api/sites/"+siteID+"/deploy-keys",
			strings.NewReader(`{"name":"test-key"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		protected.ServeHTTP(createW, createReq)
		var createResp map[string]any
		if err := json.NewDecoder(createW.Body).Decode(&createResp); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		depToken := createResp["data"].(map[string]any)["key"].(string)

		// Use the bb_dep_ token as auth — should be rejected by middleware
		// because the middleware resolves bb_dep_ tokens to site context,
		// and the handler expects a JWT (user context) to set org_id/user_id
		req := httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-keys", nil)
		req.Header.Set("Authorization", "Bearer "+depToken)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)
		// bb_dep_ tokens resolve in middleware to site context (not JWT user context)
		// The handler checks UserIDFromContext which will be missing → rejects with 401
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for bb_dep_ token, got %d: %s", w.Code, w.Body.String())
		}
	})
}

		func TestSiteKeyInputValidation(t *testing.T) {
	_, handler, protected, siteID := setupAuthWithSite(t)

	// Register and login
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"val@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)

	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"val@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	var loginResp map[string]any
	if err := json.NewDecoder(loginW.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	token, _ := loginResp["token"].(string)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"name too long", `{"name":"` + strings.Repeat("a", 101) + `"}`, http.StatusBadRequest},
		{"name with special chars", `{"name":"bad@name!"}`, http.StatusBadRequest},
		{"empty body", `{}`, http.StatusCreated}, // name defaults
		{"unknown scopes", `{"scopes":["admin"]}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/sites/"+siteID+"/deploy-keys",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			protected.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s: expected %d, got %d: %s", tc.name, tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestSiteKeyListNoTokens(t *testing.T) {
	_, handler, protected, siteID := setupAuthWithSite(t)

	// Register and login
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"no-raw@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)

	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"no-raw@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	var loginResp map[string]any
	if err := json.NewDecoder(loginW.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	token, _ := loginResp["token"].(string)

	// Create a key first
	createReq := httptest.NewRequest("POST", "/api/sites/"+siteID+"/deploy-keys",
		strings.NewReader(`{"name":"no-raw-check"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createW := httptest.NewRecorder()
	protected.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	// List — verify no raw token field
	listReq := httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-keys", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	protected.ServeHTTP(listW, listReq)

	var body map[string]any
	if err := json.NewDecoder(listW.Body).Decode(&body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	keys, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %v", body)
	}
	for _, k := range keys {
		key := k.(map[string]any)
		if _, hasRaw := key["key"]; hasRaw {
			t.Fatal("list response must not contain raw key field")
		}
	}
}
