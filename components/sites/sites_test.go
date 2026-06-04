package sites_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)   {}
func (testLogger) Warn(msg string, args ...any)   {}
func (testLogger) Error(msg string, args ...any)  {}
func (testLogger) Debug(msg string, args ...any)  {}

func setupSites(t *testing.T) *sites.Sites {
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
	return s
}

func TestSitesCreateTriggersDeploy(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	triggered := false
	s := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		TriggerDeploy: func(_ context.Context, repoID, branch string) (*sites.Deployment, error) {
			triggered = true
			if repoID != "repo-1" || branch != "main" {
				t.Fatalf("trigger args repoID=%s branch=%s", repoID, branch)
			}
			return &sites.Deployment{
				ID: "dep-1", RepoID: repoID, Branch: branch, Status: "pending",
				URL: "https://my-site.bigbase.click", Port: 10001, CreatedAt: "2026-06-04T00:00:00Z",
			}, nil
		},
	})
	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-1', 'my-site', 0, 1, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"my-site","git_repo_id":"repo-1","production_branch":"main","root_path":"./"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sites", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !triggered {
		t.Fatal("expected TriggerDeploy to be called")
	}
	var site sites.Site
	_ = json.NewDecoder(w.Body).Decode(&site)
	if site.LatestDeployment == nil || site.LatestDeployment.URL != "https://my-site.bigbase.click" {
		t.Fatalf("unexpected deployment: %+v", site.LatestDeployment)
	}
}

func TestSitesListEmpty(t *testing.T) {
	s := setupSites(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data, _ := resp["data"].([]any)
	if data == nil {
		t.Fatal("expected data array")
	}
}
