package deploy_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/kernel"
)

func TestSiteKeyCrossSite(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-x", gitDir)

	_, err := database.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("sites table: %v", err)
	}
	_, err = database.Exec(`INSERT INTO sites (id, name, git_repo_id) VALUES ('site-a', 'a', ?), ('site-b', 'b', ?)`, repoID, repoID)
	if err != nil {
		t.Fatalf("insert sites: %v", err)
	}

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{
		"repo_id": repoID,
		"site_id": "site-b",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(kernel.WithSiteID(req.Context(), "site-a"))
	w := httptest.NewRecorder()
	dep.HandleCreate(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !bytes.Contains([]byte(body), []byte("site key not authorized")) {
		t.Fatalf("expected authorization error, got %s", body)
	}
}

func TestSiteKeyMatchingSite(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-y", gitDir)

	_, err := database.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("sites table: %v", err)
	}
	_, err = database.Exec(`INSERT INTO sites (id, name, git_repo_id) VALUES ('site-a', 'a', ?)`, repoID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{
		"repo_id": repoID,
		"site_id": "site-a",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(kernel.WithSiteID(req.Context(), "site-a"))
	w := httptest.NewRecorder()
	dep.HandleCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}
