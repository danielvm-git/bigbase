package sites_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

func setupSitesWithKey(t *testing.T, encKey string) (*sites.Sites, http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	s := sites.New(sites.Options{DB: d, Logger: logger, EnvEncryptionKey: encKey})
	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return s, s.Handler(), d
}

func createTestSiteForEnvVars(t *testing.T, d *db.DB) string {
	t.Helper()
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-ev-1', 'my-app', 0, 0, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, org_id, created_at)
		 VALUES ('site-ev-1', 'my-app', 'repo-ev-1', 'main', './', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
	return "site-ev-1"
}

func TestEnvVarsAPI(t *testing.T) {
	_, h, d := setupSitesWithKey(t, "")
	siteID := createTestSiteForEnvVars(t, d)

	t.Run("list empty initially", func(t *testing.T) {
		req := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var res map[string]any
		_ = json.NewDecoder(w.Body).Decode(&res)
		data, ok := res["data"].([]any)
		if !ok || len(data) != 0 {
			t.Fatalf("expected empty data array, got %v", res["data"])
		}
	})

	t.Run("create env var", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"DATABASE_URL","value":"postgres://localhost/mydb","is_build_time":false,"is_runtime":true}`)
		req := authedRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var ev map[string]any
		_ = json.NewDecoder(w.Body).Decode(&ev)
		if ev["key"] != "DATABASE_URL" {
			t.Fatalf("expected key=DATABASE_URL, got %v", ev["key"])
		}
		if _, exists := ev["value"]; exists {
			t.Fatal("create response returned plaintext value")
		}
		if ev["is_runtime"] != true {
			t.Fatalf("expected is_runtime=true, got %v", ev["is_runtime"])
		}
		if ev["is_build_time"] != false {
			t.Fatalf("expected is_build_time=false, got %v", ev["is_build_time"])
		}
	})

	t.Run("list returns created var", func(t *testing.T) {
		req := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var res map[string]any
		_ = json.NewDecoder(w.Body).Decode(&res)
		data, ok := res["data"].([]any)
		if !ok || len(data) != 1 {
			t.Fatalf("expected 1 env var, got %v", res["data"])
		}
		ev := data[0].(map[string]any)
		if ev["key"] != "DATABASE_URL" {
			t.Fatalf("expected key=DATABASE_URL, got %v", ev["key"])
		}
	})

	t.Run("reject duplicate key", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"DATABASE_URL","value":"other","is_build_time":false,"is_runtime":true}`)
		req := authedRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("reject invalid key format lowercase", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"database_url","value":"val","is_build_time":true,"is_runtime":false}`)
		req := authedRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("reject key with spaces", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"MY KEY","value":"val","is_build_time":true,"is_runtime":false}`)
		req := authedRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update env var value", func(t *testing.T) {
		body := bytes.NewBufferString(`{"value":"postgres://prod/db","is_build_time":false,"is_runtime":true}`)
		req := authedRequest(http.MethodPut, "/api/sites/"+siteID+"/env-vars/DATABASE_URL", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var ev map[string]any
		_ = json.NewDecoder(w.Body).Decode(&ev)
		if _, exists := ev["value"]; exists {
			t.Fatal("update response returned plaintext value")
		}
	})

	t.Run("list reflects updated value", func(t *testing.T) {
		req := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var res map[string]any
		_ = json.NewDecoder(w.Body).Decode(&res)
		data := res["data"].([]any)
		ev := data[0].(map[string]any)
		// List returns value_preview (server-masked), not full plaintext (F03).
		// "postgres://prod/db" is > 4 chars so preview is "••••/db"
		preview, _ := ev["value_preview"].(string)
		if preview == "" {
			t.Fatalf("expected value_preview in list response, got none; full record: %v", ev)
		}
	})

	t.Run("delete env var", func(t *testing.T) {
		req := authedRequest(http.MethodDelete, "/api/sites/"+siteID+"/env-vars/DATABASE_URL", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("list empty after delete", func(t *testing.T) {
		req := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var res map[string]any
		_ = json.NewDecoder(w.Body).Decode(&res)
		data := res["data"].([]any)
		if len(data) != 0 {
			t.Fatalf("expected empty after delete, got %v", data)
		}
	})

	t.Run("update nonexistent key returns 404", func(t *testing.T) {
		body := bytes.NewBufferString(`{"value":"v","is_build_time":true,"is_runtime":false}`)
		req := authedRequest(http.MethodPut, "/api/sites/"+siteID+"/env-vars/MISSING_KEY", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete nonexistent key returns 404", func(t *testing.T) {
		req := authedRequest(http.MethodDelete, "/api/sites/"+siteID+"/env-vars/MISSING_KEY", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestEnvVarsAPIWithEncryption(t *testing.T) {
	// 32-byte key = 64 hex chars
	key := strings.Repeat("a1", 32)
	_, h, d := setupSitesWithKey(t, key)
	siteID := createTestSiteForEnvVars(t, d)

	t.Run("stores and retrieves encrypted value", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"API_KEY","value":"secret-token","is_build_time":true,"is_runtime":false}`)
		req := authedRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		// List returns value_preview (server-masked), not full plaintext (F03).
		// "secret-token" → preview "••••oken"
		req2 := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w2 := httptest.NewRecorder()
		h.ServeHTTP(w2, req2)
		var res map[string]any
		_ = json.NewDecoder(w2.Body).Decode(&res)
		data := res["data"].([]any)
		if len(data) != 1 {
			t.Fatalf("expected 1 env var, got %d", len(data))
		}
		ev := data[0].(map[string]any)
		preview, _ := ev["value_preview"].(string)
		if preview != "\u2022\u2022\u2022\u2022oken" {
			t.Fatalf("expected masked preview '••••oken', got %q", preview)
		}
		// Full value must NOT be present in list response.
		if ev["value"] != nil {
			t.Fatalf("list must not return full plaintext value, got %v", ev["value"])
		}
	})

	t.Run("raw DB value is not plaintext", func(t *testing.T) {
		// Verify the DB column stores cipher text, not plaintext
		var rawValue string
		err := d.QueryRowContext(context.Background(),
			"SELECT value_encrypted FROM site_env_vars WHERE key = 'API_KEY'").Scan(&rawValue)
		if err != nil {
			t.Fatalf("query raw value: %v", err)
		}
		if rawValue == "secret-token" {
			t.Fatal("expected encrypted value in DB, got plaintext")
		}
	})
}

func TestEnvVarsListDecryptFailure(t *testing.T) {
	// Sites runs with a real key, but we seed a row whose stored value cannot be
	// decrypted (garbage ciphertext, e.g. after a key rotation or DB corruption).
	key := strings.Repeat("a1", 32)
	_, h, d := setupSitesWithKey(t, key)
	siteID := createTestSiteForEnvVars(t, d)

	_, err := d.ExecContext(context.Background(),
		`INSERT INTO site_env_vars (id, site_id, key, value_encrypted, is_build_time, is_runtime, created_at, updated_at)
		 VALUES ('ev-bad', ?, 'BROKEN', 'not-valid-ciphertext!!!', 0, 1, datetime('now'), datetime('now'))`,
		siteID)
	if err != nil {
		t.Fatalf("seed broken row: %v", err)
	}

	req := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]any
	_ = json.NewDecoder(w.Body).Decode(&res)
	data := res["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(data))
	}
	ev := data[0].(map[string]any)
	// F04: an undecryptable value must surface a visible sentinel, not masquerade
	// as a normal "••••" short value.
	if ev["value_preview"] != "<decrypt error>" {
		t.Fatalf("expected '<decrypt error>' preview, got %v", ev["value_preview"])
	}
	if ev["value"] != nil {
		t.Fatalf("list must not return a plaintext value, got %v", ev["value"])
	}
}

func TestEnvVarsSiteIsolation(t *testing.T) {
	_, h, d := setupSitesWithKey(t, "")

	// Site A (org_id=1)
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-a', 'app-a', 0, 0, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert repo a: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, org_id, created_at)
		 VALUES ('site-a', 'app-a', 'repo-a', 'main', './', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert site a: %v", err)
	}

	// Site B (org_id=1, same org for env var isolation test)
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-b', 'app-b', 0, 0, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert repo b: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, org_id, created_at)
		 VALUES ('site-b', 'app-b', 'repo-b', 'main', './', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert site b: %v", err)
	}

	// Add env var to site A
	body := bytes.NewBufferString(`{"key":"SECRET_A","value":"only-for-a","is_build_time":true,"is_runtime":true}`)
	req := authedRequest(http.MethodPost, "/api/sites/site-a/env-vars", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create env var for site-a: %d %s", w.Code, w.Body.String())
	}

	// List site B — must be empty
	req2 := authedRequest(http.MethodGet, "/api/sites/site-b/env-vars", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var res map[string]any
	_ = json.NewDecoder(w2.Body).Decode(&res)
	data := res["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("site-b should have no env vars, got %v", data)
	}
}
