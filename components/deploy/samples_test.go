package deploy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/db"
)

// initSampleRepo seeds git_repos table and creates a bare git repo for
// testing the sample deploy flow without hitting any remote.
func initSampleRepo(t *testing.T, database *db.DB, gitDir, sampleName string) string {
	t.Helper()

	_, err := database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL DEFAULT 0,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	if err != nil {
		t.Fatalf("ensure git_repos table: %v", err)
	}

	id := "sample-" + sampleName
	_, err = database.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description) VALUES (?, ?, 0, 0, 'main', 'sample')",
		id, sampleName)
	if err != nil {
		t.Fatalf("insert sample repo row: %v", err)
	}

	// Bare repo on disk named <id>.git
	repoPath := filepath.Join(gitDir, id+".git")
	mustRun(t, "git", "init", "--bare", "-b", "main", repoPath)
	mustRun(t, "git", "config", "--global", "--add", "safe.directory", repoPath)

	sourceDir := t.TempDir()
	mustRun(t, "git", "init", "-b", "main", sourceDir)
	mustRun(t, "git", "config", "--global", "--add", "safe.directory", sourceDir)
	mustRun(t, "git", "-C", sourceDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", sourceDir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("<h1>sample</h1>"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "init")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")

	return id
}

func TestSampleList(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	t.Run("returns three sample apps", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/samples", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data []struct {
				Name        string `json:"name"`
				RepoURL     string `json:"repo_url"`
				Description string `json:"description"`
			} `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data) != 3 {
			t.Fatalf("expected 3 samples, got %d", len(resp.Data))
		}
		for _, s := range resp.Data {
			if s.Name == "" || s.RepoURL == "" || s.Description == "" {
				t.Errorf("incomplete sample descriptor: %+v", s)
			}
		}
	})

	t.Run("rejects POST on list endpoint", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/samples", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 got %d", w.Code)
		}
	})
}

func TestSampleDeploy(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)

	// Pre-seed the git_repos table and bare repo for the "todo-app" sample.
	initSampleRepo(t, database, gitDir, "todo-app")

	t.Run("deploy known sample returns 201", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/samples/todo-app/deploy", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["id"] == nil || resp["id"] == "" {
			t.Error("expected deployment id in response")
		}
		if resp["status"] != "pending" {
			t.Errorf("expected status=pending got %v", resp["status"])
		}
	})

	t.Run("returns 404 for unknown sample", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/samples/nonexistent/deploy", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 got %d", w.Code)
		}
	})

	t.Run("rejects GET on deploy sub-path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/samples/todo-app/deploy", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 got %d", w.Code)
		}
	})
}
