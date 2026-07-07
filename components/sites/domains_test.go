package sites_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

func setupSitesWithHandler(t *testing.T) (*sites.Sites, http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	s := sites.New(sites.Options{DB: d, Logger: logger})

	k.Register(d)
	k.Register(g)
	k.Register(s)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return s, s.Handler(), d
}

func createTestSite(t *testing.T, h http.Handler, d *db.DB) string {
	t.Helper()
	_, err := d.Exec(`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		VALUES ('repo-001', 'testrepo', 0, 1, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed git repo: %v", err)
	}
	body := `{"name":"testsite","git_repo_id":"repo-001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sites", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create site: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["id"].(string)
}

func TestCustomDomain(t *testing.T) {
	t.Run("register domain", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		body := `{"domain":"example.com"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/sites/"+siteID+"/domains", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		raw, _ := io.ReadAll(w.Body)
		var resp map[string]any
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if resp["domain"] != "example.com" {
			t.Errorf("expected domain=example.com, got %v", resp["domain"])
		}
		if resp["verify_token"] == "" || resp["verify_token"] == nil {
			t.Error("expected non-empty verify_token")
		}
	})

	t.Run("list domains for site", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		// Register two domains
		for _, domain := range []string{"alpha.com", "beta.com"} {
			body := `{"domain":"` + domain + `"}`
			req := httptest.NewRequest(http.MethodPost,
				"/api/sites/"+siteID+"/domains", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("register %s: %d", domain, w.Code)
			}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/domains", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", w.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data, ok := resp["data"].([]any)
		if !ok || len(data) != 2 {
			t.Errorf("expected 2 domains, got %v", resp)
		}
	})

	t.Run("verify endpoint returns unverified status for unknown dns", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		// Register
		body := `{"domain":"notexist.invalid"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/sites/"+siteID+"/domains", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("register: %d", w.Code)
		}

		// Verify (POST — it mutates verified_at and triggers a DNS lookup)
		req = httptest.NewRequest(http.MethodPost,
			"/api/sites/"+siteID+"/domains/notexist.invalid/verify", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)

		// Should return 200 with verified=false (DNS doesn't have our TXT record)
		if w.Code != http.StatusOK {
			t.Fatalf("verify: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["verified"] != false {
			t.Errorf("expected verified=false for unknown domain, got %v", resp["verified"])
		}
	})

	t.Run("register duplicate domain returns 409", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		body := `{"domain":"dup.com"}`
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost,
				"/api/sites/"+siteID+"/domains", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if i == 0 && w.Code != http.StatusCreated {
				t.Fatalf("first create: expected 201, got %d", w.Code)
			}
			if i == 1 && w.Code != http.StatusConflict {
				t.Fatalf("second create: expected 409, got %d", w.Code)
			}
		}
	})

	t.Run("register domain with missing domain field returns 400", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		body := `{}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/sites/"+siteID+"/domains", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("register malformed domain returns 400", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		for _, bad := range []string{"not a domain", "no-tld", "http://example.com", "exa_mple.com"} {
			body := `{"domain":"` + bad + `"}`
			req := httptest.NewRequest(http.MethodPost,
				"/api/sites/"+siteID+"/domains", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("domain %q: expected 400, got %d", bad, w.Code)
			}
		}
	})

	t.Run("register domain for unknown site returns 404", func(t *testing.T) {
		_, h, _ := setupSitesWithHandler(t)

		body := `{"domain":"orphan.com"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/sites/does-not-exist/domains", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete domain removes it", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		body := `{"domain":"gone.com"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/sites/"+siteID+"/domains", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("register: expected 201, got %d", w.Code)
		}

		// Delete
		req = httptest.NewRequest(http.MethodDelete, "/api/sites/"+siteID+"/domains/gone.com", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("delete: expected 204, got %d: %s", w.Code, w.Body.String())
		}

		// List is now empty
		req = httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/domains", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if data, _ := resp["data"].([]any); len(data) != 0 {
			t.Errorf("expected 0 domains after delete, got %v", resp)
		}

		// Deleting again returns 404
		req = httptest.NewRequest(http.MethodDelete, "/api/sites/"+siteID+"/domains/gone.com", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("second delete: expected 404, got %d", w.Code)
		}
	})

	t.Run("GET on verify is rejected (POST only)", func(t *testing.T) {
		_, h, d := setupSitesWithHandler(t)
		siteID := createTestSite(t, h, d)

		body := `{"domain":"getverify.com"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/sites/"+siteID+"/domains", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("register: %d", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet,
			"/api/sites/"+siteID+"/domains/getverify.com/verify", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("GET verify: expected 405, got %d", w.Code)
		}
	})
}
