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
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, created_at)
		 VALUES ('site-ev-1', 'my-app', 'repo-ev-1', 'main', './', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
	return "site-ev-1"
}

func TestEnvVarsAPI(t *testing.T) {
	_, h, d := setupSitesWithKey(t, "")
	siteID := createTestSiteForEnvVars(t, d)

	t.Run("list empty initially", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
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
		req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
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
		if ev["value"] != "postgres://localhost/mydb" {
			t.Fatalf("expected value=postgres://localhost/mydb, got %v", ev["value"])
		}
		if ev["is_runtime"] != true {
			t.Fatalf("expected is_runtime=true, got %v", ev["is_runtime"])
		}
		if ev["is_build_time"] != false {
			t.Fatalf("expected is_build_time=false, got %v", ev["is_build_time"])
		}
	})

	t.Run("list returns created var", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
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
		req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("reject invalid key format lowercase", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"database_url","value":"val","is_build_time":true,"is_runtime":false}`)
		req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("reject key with spaces", func(t *testing.T) {
		body := bytes.NewBufferString(`{"key":"MY KEY","value":"val","is_build_time":true,"is_runtime":false}`)
		req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update env var value", func(t *testing.T) {
		body := bytes.NewBufferString(`{"value":"postgres://prod/db","is_build_time":false,"is_runtime":true}`)
		req := httptest.NewRequest(http.MethodPut, "/api/sites/"+siteID+"/env-vars/DATABASE_URL", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var ev map[string]any
		_ = json.NewDecoder(w.Body).Decode(&ev)
		if ev["value"] != "postgres://prod/db" {
			t.Fatalf("expected updated value, got %v", ev["value"])
		}
	})

	t.Run("list reflects updated value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var res map[string]any
		_ = json.NewDecoder(w.Body).Decode(&res)
		data := res["data"].([]any)
		ev := data[0].(map[string]any)
		if ev["value"] != "postgres://prod/db" {
			t.Fatalf("expected updated value in list, got %v", ev["value"])
		}
	})

	t.Run("delete env var", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/sites/"+siteID+"/env-vars/DATABASE_URL", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("list empty after delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
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
		req := httptest.NewRequest(http.MethodPut, "/api/sites/"+siteID+"/env-vars/MISSING_KEY", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete nonexistent key returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/sites/"+siteID+"/env-vars/MISSING_KEY", nil)
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
		req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/env-vars", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		// List and verify plaintext is returned
		req2 := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/env-vars", nil)
		w2 := httptest.NewRecorder()
		h.ServeHTTP(w2, req2)
		var res map[string]any
		_ = json.NewDecoder(w2.Body).Decode(&res)
		data := res["data"].([]any)
		if len(data) != 1 {
			t.Fatalf("expected 1 env var, got %d", len(data))
		}
		ev := data[0].(map[string]any)
		if ev["value"] != "secret-token" {
			t.Fatalf("expected decrypted value, got %v", ev["value"])
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

func TestEnvVarsSiteIsolation(t *testing.T) {
	_, h, d := setupSitesWithKey(t, "")

	// Site A
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-a', 'app-a', 0, 0, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert repo a: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, created_at)
		 VALUES ('site-a', 'app-a', 'repo-a', 'main', './', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert site a: %v", err)
	}

	// Site B
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-b', 'app-b', 0, 0, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert repo b: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, created_at)
		 VALUES ('site-b', 'app-b', 'repo-b', 'main', './', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert site b: %v", err)
	}

	// Add env var to site A
	body := bytes.NewBufferString(`{"key":"SECRET_A","value":"only-for-a","is_build_time":true,"is_runtime":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sites/site-a/env-vars", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create env var for site-a: %d %s", w.Code, w.Body.String())
	}

	// List site B — must be empty
	req2 := httptest.NewRequest(http.MethodGet, "/api/sites/site-b/env-vars", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var res map[string]any
	_ = json.NewDecoder(w2.Body).Decode(&res)
	data := res["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("site-b should have no env vars, got %v", data)
	}
}
