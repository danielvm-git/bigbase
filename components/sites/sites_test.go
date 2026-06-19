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

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

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
		TriggerDeploy: func(_ context.Context, repoID, branch, siteName, siteID string) (*sites.Deployment, error) {
			triggered = true
			if repoID != "repo-1" || branch != "main" || siteName != "my-site" {
				t.Fatalf("trigger args repoID=%s branch=%s siteName=%s", repoID, branch, siteName)
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

func TestSitesCreateTriggersDeployWithCustomName(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	var receivedSiteName string
	s := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		TriggerDeploy: func(_ context.Context, repoID, branch, siteName, siteID string) (*sites.Deployment, error) {
			receivedSiteName = siteName
			return &sites.Deployment{
				ID: "dep-1", RepoID: repoID, Branch: branch, Status: "pending",
				URL: "https://" + siteName + ".bigbase.click", Port: 10001, CreatedAt: "2026-06-04T00:00:00Z",
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

	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-1', 'original-repo-name', 0, 1, 'main', '', datetime('now'))`)

	body := bytes.NewBufferString(`{"name":"custom-site-name","git_repo_id":"repo-1","production_branch":"main","root_path":"./"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sites", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if receivedSiteName != "custom-site-name" {
		t.Fatalf("expected siteName 'custom-site-name', got '%s'", receivedSiteName)
	}
}

func TestSitesListRequestLogs(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	s := sites.New(sites.Options{DB: d, Logger: logger})
	k.Register(d)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	siteID := "s1"
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id) VALUES (?, 'mysite', 'r1')`, siteID)

	// Create table and insert dummy logs (since deploy component owns the table but we test site component)
	_, _ = d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS site_request_logs (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)

	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO site_request_logs (id, site_id, method, path, status, duration_ms, created_at)
		 VALUES ('l1', ?, 'GET', '/test', 200, 5, '2026-06-12T15:00:00Z')`, siteID)

	req := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/logs", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data, _ := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 log, got %d", len(data))
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

func setupSitesDeleteTest(t *testing.T) (*db.DB, http.Handler) {
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
	return d, s.Handler()
}

func createSiteDeleteSupportTables(t *testing.T, d *db.DB) {
	t.Helper()
	if _, err := d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL DEFAULT '',
		site_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("create deployments table: %v", err)
	}
	if _, err := d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS site_request_logs (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("create site_request_logs table: %v", err)
	}
}

func seedSiteForDelete(t *testing.T, d *db.DB, siteID, repoID string) {
	t.Helper()
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES (?, 'delete-site-repo', 0, 1, 'main', '', datetime('now'))`, repoID)
	if err != nil {
		t.Fatalf("insert git repo: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at)
		 VALUES (?, 'delete-site', ?, 'main', './', 'owner/delete-site', datetime('now'))`, siteID, repoID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO site_domains (id, site_id, domain, verify_token, created_at)
		 VALUES (?, ?, 'delete-site.example.com', 'token', datetime('now'))`, siteID+"-domain", siteID)
	if err != nil {
		t.Fatalf("insert site domain: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO site_request_logs (id, site_id, method, path, status, duration_ms, created_at)
		 VALUES (?, ?, 'GET', '/', 200, 4, datetime('now'))`, siteID+"-log", siteID)
	if err != nil {
		t.Fatalf("insert site request log: %v", err)
	}
}

func countRows(t *testing.T, d *db.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := d.QueryRowContext(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func TestDeleteSiteNotFound(t *testing.T) {
	_, h := setupSitesDeleteTest(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/sites/missing", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSiteActiveDeploymentConflict(t *testing.T) {
	for _, status := range []string{"pending", "building", "running"} {
		t.Run(status, func(t *testing.T) {
			d, h := setupSitesDeleteTest(t)
			createSiteDeleteSupportTables(t, d)
			seedSiteForDelete(t, d, "site-active-"+status, "repo-active-"+status)
			_, err := d.ExecContext(context.Background(),
				`INSERT INTO deployments (id, site_id, repo_id, status) VALUES (?, ?, ?, ?)`,
				"dep-"+status, "site-active-"+status, "repo-active-"+status, status)
			if err != nil {
				t.Fatalf("insert deployment: %v", err)
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/sites/site-active-"+status, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
			}
			if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", "site-active-"+status); got != 1 {
				t.Fatalf("site should remain after conflict, got %d rows", got)
			}
			if got := countRows(t, d, "SELECT COUNT(*) FROM deployments WHERE id = ?", "dep-"+status); got != 1 {
				t.Fatalf("deployment should remain after conflict, got %d rows", got)
			}
			if got := countRows(t, d, "SELECT COUNT(*) FROM git_repos WHERE id = ?", "repo-active-"+status); got != 1 {
				t.Fatalf("git repo should remain after conflict, got %d rows", got)
			}
		})
	}
}

func TestDeleteSiteCascade(t *testing.T) {
	t.Run("canonical site id", func(t *testing.T) {
		d, h := setupSitesDeleteTest(t)
		createSiteDeleteSupportTables(t, d)
		seedSiteForDelete(t, d, "site-delete", "repo-delete")
		_, err := d.ExecContext(context.Background(),
			`INSERT INTO deployments (id, site_id, repo_id, status) VALUES
			 ('dep-site-delete', 'site-delete', 'repo-delete', 'failed'),
			 ('dep-legacy-delete', '', 'repo-delete', 'failed'),
			 ('dep-other-site-same-repo', 'other-site', 'repo-delete', 'failed')`)
		if err != nil {
			t.Fatalf("insert deployments: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/sites/site-delete", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
		assertSiteDeleted(t, d, "site-delete", "repo-delete")
	})

	t.Run("legacy git repo id alias", func(t *testing.T) {
		d, h := setupSitesDeleteTest(t)
		createSiteDeleteSupportTables(t, d)
		seedSiteForDelete(t, d, "site-alias-delete", "repo-alias-delete")
		_, err := d.ExecContext(context.Background(),
			`INSERT INTO deployments (id, site_id, repo_id, status) VALUES
			 ('dep-alias-site', 'site-alias-delete', 'repo-alias-delete', 'failed'),
			 ('dep-alias-legacy', '', 'repo-alias-delete', 'failed')`)
		if err != nil {
			t.Fatalf("insert deployments: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/sites/repo-alias-delete", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
		assertSiteDeleted(t, d, "site-alias-delete", "repo-alias-delete")
	})
}

func assertSiteDeleted(t *testing.T, d *db.DB, siteID, repoID string) {
	t.Helper()
	if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", siteID); got != 0 {
		t.Fatalf("site row should be deleted, got %d", got)
	}
	if got := countRows(t, d, "SELECT COUNT(*) FROM site_domains WHERE site_id = ?", siteID); got != 0 {
		t.Fatalf("site domains should be deleted, got %d", got)
	}
	if got := countRows(t, d, "SELECT COUNT(*) FROM site_request_logs WHERE site_id = ?", siteID); got != 0 {
		t.Fatalf("site request logs should be deleted, got %d", got)
	}
	if got := countRows(t, d, "SELECT COUNT(*) FROM deployments WHERE site_id = ? OR (site_id = '' AND repo_id = ?)", siteID, repoID); got != 0 {
		t.Fatalf("site deployments should be deleted, got %d", got)
	}
	if got := countRows(t, d, "SELECT COUNT(*) FROM deployments WHERE site_id <> ? AND site_id <> '' AND repo_id = ?", siteID, repoID); siteID == "site-delete" && got != 1 {
		t.Fatalf("other site deployments for same repo should remain, got %d", got)
	}
	if got := countRows(t, d, "SELECT COUNT(*) FROM git_repos WHERE id = ?", repoID); got != 1 {
		t.Fatalf("source git repo should remain, got %d", got)
	}
}
