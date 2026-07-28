package sites_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

// authedRequestSite creates an HTTP request with org_id=1 in context.
// This is used for tests that need auth context for site ownership checks.
func authedRequestSite(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	return req.WithContext(auth.WithOrgID(req.Context(), 1))
}

// authedRequestSiteAs creates an HTTP request with org_id=1 and the given user
// role in context. Used to exercise role-gated behaviour such as the admin-only
// legacy org_id=0 site visibility carve-out (BUG-2026-07-28T000002).
func authedRequestSiteAs(method, path string, body io.Reader, role string) *http.Request {
	req := httptest.NewRequest(method, path, body)
	ctx := auth.WithOrgID(req.Context(), 1)
	return req.WithContext(auth.WithUserRole(ctx, role))
}

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
		TriggerDeploy: func(_ context.Context, repoID, branch, siteName, siteID string, _ []string, _ string) (*sites.Deployment, error) {
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
	req := authedRequestSite(http.MethodPost, "/api/sites", body)
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
		TriggerDeploy: func(_ context.Context, repoID, branch, siteName, siteID string, _ []string, _ string) (*sites.Deployment, error) {
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
	req := authedRequestSite(http.MethodPost, "/api/sites", body)
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
		`INSERT INTO sites (id, name, git_repo_id, org_id) VALUES (?, 'mysite', 'r1', 1)`, siteID)

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

	req := authedRequestSite(http.MethodGet, "/api/sites/"+siteID+"/logs", nil)
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
	req := authedRequestSite(http.MethodGet, "/api/sites", nil)
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

func seedDeploymentsTable(t *testing.T, d *db.DB) {
	t.Helper()
	_, err := d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		branch TEXT NOT NULL DEFAULT 'main',
		commit_sha TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		url TEXT DEFAULT '',
		port INTEGER DEFAULT 0,
		app_type TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("create deployments table: %v", err)
	}
}

func TestSitesListReturnsAllSitesWithDeployments(t *testing.T) {
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
	seedDeploymentsTable(t, d)

	const siteCount = 35
	for i := 0; i < siteCount; i++ {
		repoID := fmt.Sprintf("repo-%d", i)
		siteID := fmt.Sprintf("site-%d", i)
		_, _ = d.ExecContext(context.Background(),
			`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
			 VALUES (?, ?, 0, 1, 'main', '', datetime('now'))`, repoID, fmt.Sprintf("site-%d", i))
		_, _ = d.ExecContext(context.Background(),
			`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at, org_id)
			 VALUES (?, ?, ?, 'main', './', ?, datetime('now'), 1)`,
			siteID, fmt.Sprintf("site-%d", i), repoID, fmt.Sprintf("owner/site-%d", i))
		// Older deployment
		_, _ = d.ExecContext(context.Background(),
			`INSERT INTO deployments (id, repo_id, branch, commit_sha, status, url, port, app_type, created_at)
			 VALUES (?, ?, 'main', 'old0000', 'replaced', 'http://localhost:1', 1, 'static', '2026-01-01T00:00:00Z')`,
			fmt.Sprintf("dep-old-%d", i), repoID)
		// Latest deployment — must be attached to the site card
		_, _ = d.ExecContext(context.Background(),
			`INSERT INTO deployments (id, repo_id, branch, commit_sha, status, url, port, app_type, created_at)
			 VALUES (?, ?, 'main', 'new1234', 'running', 'http://localhost:8080', 8080, 'static', '2026-06-01T00:00:00Z')`,
			fmt.Sprintf("dep-new-%d", i), repoID)
	}

	req := authedRequestSite(http.MethodGet, "/api/sites", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []struct {
			ID               string `json:"id"`
			GitRepoID        string `json:"git_repo_id"`
			LatestDeployment *struct {
				CommitSHA string `json:"commit_sha"`
				Status    string `json:"status"`
			} `json:"latest_deployment"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != siteCount {
		t.Fatalf("expected %d sites, got %d", siteCount, len(resp.Data))
	}
	for _, site := range resp.Data {
		if site.LatestDeployment == nil {
			t.Fatalf("site %s missing latest_deployment", site.ID)
		}
		if site.LatestDeployment.CommitSHA != "new1234" {
			t.Fatalf("site %s: expected latest commit new1234, got %s", site.ID, site.LatestDeployment.CommitSHA)
		}
		if site.LatestDeployment.Status != "running" {
			t.Fatalf("site %s: expected status running, got %s", site.ID, site.LatestDeployment.Status)
		}
	}
}

// --- e36 delete-site tests ---

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
	_, _ = d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL DEFAULT '',
		site_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	_, _ = d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS site_request_logs (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
}

func seedSiteForDelete(t *testing.T, d *db.DB, siteID, repoID string) {
	t.Helper()
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES (?, 'delete-site-repo', 0, 1, 'main', '', datetime('now'))`, repoID)
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at, org_id)
		 VALUES (?, 'delete-site', ?, 'main', './', 'owner/delete-site', datetime('now'), 1)`, siteID, repoID)
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO site_domains (id, site_id, domain, verify_token, created_at)
		 VALUES ('dom-1', ?, 'delete-site.example.com', 'token', datetime('now'))`, siteID)
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO site_request_logs (id, site_id, method, path, status, duration_ms, created_at)
		 VALUES ('log-1', ?, 'GET', '/', 200, 4, datetime('now'))`, siteID)
}

func countRows(t *testing.T, d *db.DB, query string, args ...any) int {
	t.Helper()
	var count int
	_ = d.QueryRowContext(context.Background(), query, args...).Scan(&count)
	return count
}

func TestDeleteSiteNotFound(t *testing.T) {
	_, h := setupSitesDeleteTest(t)
	req := authedRequestSite(http.MethodDelete, "/api/sites/missing", nil)
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
			seedSiteForDelete(t, d, "site-"+status, "repo-"+status)
			_, _ = d.ExecContext(context.Background(),
				`INSERT INTO deployments (id, site_id, repo_id, status) VALUES (?, ?, ?, ?)`,
				"dep-"+status, "site-"+status, "repo-"+status, status)

			req := authedRequestSite(http.MethodDelete, "/api/sites/site-"+status, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
			}
			if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", "site-"+status); got != 1 {
				t.Fatalf("site should remain, got %d rows", got)
			}
		})
	}
}

func TestDeleteSiteCascade(t *testing.T) {
	t.Run("canonical site id", func(t *testing.T) {
		d, h := setupSitesDeleteTest(t)
		createSiteDeleteSupportTables(t, d)
		seedSiteForDelete(t, d, "site-del", "repo-del")
		_, _ = d.ExecContext(context.Background(),
			`INSERT INTO deployments (id, site_id, repo_id, status) VALUES
			 ('dep-site', 'site-del', 'repo-del', 'failed'),
			 ('dep-legacy', '', 'repo-del', 'failed'),
			 ('dep-other', 'other-site', 'repo-del', 'failed')`)

		req := authedRequestSite(http.MethodDelete, "/api/sites/site-del", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", "site-del"); got != 0 {
			t.Fatalf("site row should be deleted, got %d", got)
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM site_domains WHERE site_id = ?", "site-del"); got != 0 {
			t.Fatalf("site domains should be deleted, got %d", got)
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM site_request_logs WHERE site_id = ?", "site-del"); got != 0 {
			t.Fatalf("site request logs should be deleted, got %d", got)
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM deployments WHERE site_id = ? OR (site_id = '' AND repo_id = ?)", "site-del", "repo-del"); got != 0 {
			t.Fatalf("site deployments should be deleted, got %d", got)
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM deployments WHERE site_id <> '' AND repo_id = ?", "repo-del"); got != 1 {
			t.Fatalf("other site deployment for same repo should remain, got %d", got)
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM git_repos WHERE id = ?", "repo-del"); got != 1 {
			t.Fatalf("source git repo should remain, got %d", got)
		}
	})

	t.Run("legacy git repo id alias", func(t *testing.T) {
		d, h := setupSitesDeleteTest(t)
		createSiteDeleteSupportTables(t, d)
		seedSiteForDelete(t, d, "site-alias", "repo-alias")
		_, _ = d.ExecContext(context.Background(),
			`INSERT INTO deployments (id, site_id, repo_id, status) VALUES
			 ('dep-alias-site', 'site-alias', 'repo-alias', 'failed'),
			 ('dep-alias-legacy', '', 'repo-alias', 'failed')`)

		req := authedRequestSite(http.MethodDelete, "/api/sites/repo-alias", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", "site-alias"); got != 0 {
			t.Fatalf("site row should be deleted, got %d", got)
		}
		if got := countRows(t, d, "SELECT COUNT(*) FROM git_repos WHERE id = ?", "repo-alias"); got != 1 {
			t.Fatalf("source git repo should remain, got %d", got)
		}
	})
}

func TestDeleteSiteCallsCleanupCallback(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})

	var cleanupCalled bool
	var gotSiteID, gotRepoID string

	s := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		DeleteSiteCleanup: func(_ context.Context, siteID, repoID string) error {
			cleanupCalled = true
			gotSiteID = siteID
			gotRepoID = repoID
			return nil
		},
	})
	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	createSiteDeleteSupportTables(t, d)
	seedSiteForDelete(t, d, "site-cb", "repo-cb")
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO deployments (id, site_id, repo_id, status) VALUES ('dep-running', 'site-cb', 'repo-cb', 'running')`)

	req := authedRequestSite(http.MethodDelete, "/api/sites/site-cb", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 with cleanup, got %d: %s", w.Code, w.Body.String())
	}
	if !cleanupCalled {
		t.Fatal("expected DeleteSiteCleanup callback to be called")
	}
	if gotSiteID != "site-cb" {
		t.Fatalf("cleanup got site_id=%q, want site-cb", gotSiteID)
	}
	if gotRepoID != "repo-cb" {
		t.Fatalf("cleanup got repo_id=%q, want repo-cb", gotRepoID)
	}
	// Site should still be cascade-deleted after cleanup
	if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", "site-cb"); got != 0 {
		t.Fatalf("site should be deleted after cleanup, got %d", got)
	}
}

func TestDeleteSiteHandlesMissingColumn(t *testing.T) {
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

	// Simulate old production DB: deployments table WITHOUT site_id column
	_, _ = d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	_, _ = d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS site_request_logs (
		id TEXT PRIMARY KEY,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)

	seedSiteForDelete(t, d, "site-olddb", "repo-olddb")
	_, _ = d.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, status) VALUES ('dep-olddb', 'repo-olddb', 'failed')`)

	req := authedRequestSite(http.MethodDelete, "/api/sites/site-olddb", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 even with missing columns, got %d: %s", w.Code, w.Body.String())
	}
	if got := countRows(t, d, "SELECT COUNT(*) FROM sites WHERE id = ?", "site-olddb"); got != 0 {
		t.Fatalf("site should be deleted, got %d", got)
	}
}

// debugLoggerForTest logs to t.Log so we can see git errors in CI output.
type debugLoggerForTest struct{ *testing.T }

func (d debugLoggerForTest) Info(msg string, args ...any)  { d.Logf("[INFO] "+msg, args...) }
func (d debugLoggerForTest) Warn(msg string, args ...any)  { d.Logf("[WARN] "+msg, args...) }
func (d debugLoggerForTest) Error(msg string, args ...any) { d.Logf("[ERROR] "+msg, args...) }
func (d debugLoggerForTest) Debug(msg string, args ...any) {}

func TestSiteManifestGetAndSave(t *testing.T) {
	logger := debugLoggerForTest{T: t}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()

	// Create bare git repo manually for test
	repoID := "repo-1"
	bareRepoPath := filepath.Join(gitDir, repoID+".git")
	if err := os.MkdirAll(bareRepoPath, 0755); err != nil {
		t.Fatalf("failed to create bare repo dir: %v", err)
	}
	cmd := exec.Command("git", "init", "--bare", bareRepoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}

	// Seed an initial commit with a main branch
	tempClone := t.TempDir()
	cmd = exec.Command("git", "clone", bareRepoPath, tempClone)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git clone bare repo: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempClone
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempClone
	_ = cmd.Run()

	err := os.WriteFile(filepath.Join(tempClone, "README.md"), []byte("# Hello"), 0644)
	if err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tempClone
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempClone
	_ = cmd.Run()
	// Rename branch to main regardless of init.defaultBranch setting
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = tempClone
	_ = cmd.Run()
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = tempClone
	_ = cmd.Run()

	g := git.New(git.Options{DB: d, Logger: logger, Dir: gitDir})
	s := sites.New(sites.Options{
		DB:               d,
		Logger:           logger,
		GitDir:           gitDir,
		ValidateManifest: deploy.ValidateManifest,
	})
	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	_, err = d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES ('repo-1', 'test-site', 0, 1, 'main', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert git_repo: %v", err)
	}
	_, err = d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, created_at, org_id)
		 VALUES ('site-1', 'test-site', 'repo-1', 'main', './', datetime('now'), 1)`)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	// 1. GET manifest -> should return exists = false
	req := authedRequestSite(http.MethodGet, "/api/sites/site-1/manifest", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var getResp struct {
		Exists  bool   `json:"exists"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to parse get response: %v", err)
	}
	if getResp.Exists {
		t.Fatalf("expected exists to be false, got true")
	}

	// 2. POST manifest -> with invalid content (fails validation)
	invalidYAML := `
version: 1
framework: unknown-framework
`
	body := bytes.NewBufferString(fmt.Sprintf(`{"content": %q}`, invalidYAML))
	req = authedRequestSite(http.MethodPost, "/api/sites/site-1/manifest", body)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid manifest, got %d: %s", w.Code, w.Body.String())
	}

	// 3. POST manifest -> with valid content
	validYAML := `
version: 1
framework: static
build:
  command: "echo"
start:
  command: "serve"
  port: 8080
`
	body = bytes.NewBufferString(fmt.Sprintf(`{"content": %q}`, validYAML))
	req = authedRequestSite(http.MethodPost, "/api/sites/site-1/manifest", body)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for valid manifest, got %d: %s", w.Code, w.Body.String())
	}

	// 4. GET manifest -> should return exists = true, content = validYAML
	req = authedRequestSite(http.MethodGet, "/api/sites/site-1/manifest", nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to parse get response: %v", err)
	}
	if !getResp.Exists {
		t.Fatalf("expected exists to be true, got false")
	}
	if !strings.Contains(getResp.Content, "framework: static") {
		t.Fatalf("expected content to contain static, got: %s", getResp.Content)
	}
}

func TestSiteAuthPolicyStruct(t *testing.T) {
	s := setupSites(t)
	ctx := context.Background()

	// Insert a site directly with auth policy
	siteID := "site-auth-struct-test"
	now := "2026-07-08T20:00:00Z"
	policyJSON := `{"default":"protected","protected_paths":["/secret/*"],"public_paths":["/public"],"accept":["jwt"]}`

	_, err := s.DB().ExecContext(ctx,
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at, auth_policy, org_id)
		 VALUES (?, 'test-site', 'repo-1', 'main', './', '', ?, ?, 1)`,
		siteID, now, policyJSON)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	// Retrieve site via GetSite
	site, err := s.GetSite(ctx, siteID)
	if err != nil {
		t.Fatalf("GetSite: %v", err)
	}
	if site.AuthPolicy == nil {
		t.Fatalf("expected AuthPolicy not to be nil")
	}
	if site.AuthPolicy.Default != "protected" {
		t.Errorf("expected default to be protected, got %s", site.AuthPolicy.Default)
	}
	if len(site.AuthPolicy.ProtectedPaths) != 1 || site.AuthPolicy.ProtectedPaths[0] != "/secret/*" {
		t.Errorf("unexpected protected paths: %v", site.AuthPolicy.ProtectedPaths)
	}

	// Retrieve sites via ListSites
	allSites, err := s.ListSites(ctx)
	if err != nil {
		t.Fatalf("ListSites: %v", err)
	}
	found := false
	for _, st := range allSites {
		if st.ID == siteID {
			found = true
			if st.AuthPolicy == nil {
				t.Fatalf("expected AuthPolicy not to be nil in list")
			}
			if st.AuthPolicy.Default != "protected" {
				t.Errorf("expected default in list to be protected, got %s", st.AuthPolicy.Default)
			}
		}
	}
	if !found {
		t.Fatalf("site not found in ListSites")
	}
}

func TestSiteAuthPolicyAPI(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})

	var callbackCalled bool
	var callbackSiteID string
	var callbackPolicyJSON string

	s := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		UpdateAuthPolicy: func(siteID string, policyJSON string) {
			callbackCalled = true
			callbackSiteID = siteID
			callbackPolicyJSON = policyJSON
		},
	})

	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	ctx := context.Background()
	siteID := "site-auth-api-test"
	_, err := d.ExecContext(ctx,
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at, org_id)
		 VALUES (?, 'test-site', 'repo-1', 'main', './', '', datetime('now'), 1)`,
		siteID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	// 1. GET initial auth policy (should default to public)
	req := authedRequestSite(http.MethodGet, fmt.Sprintf("/api/sites/%s/auth-policy", siteID), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var policy sites.AuthPolicy
	if err := json.Unmarshal(w.Body.Bytes(), &policy); err != nil {
		t.Fatalf("failed to unmarshal policy: %v", err)
	}
	if policy.Default != "public" {
		t.Errorf("expected default policy 'public', got '%s'", policy.Default)
	}

	// 2. POST updated auth policy
	updatedPolicyJSON := `{"default":"protected","protected_paths":["/books/*"],"public_paths":["/login"],"accept":["jwt"]}`
	req = authedRequestSite(http.MethodPost, fmt.Sprintf("/api/sites/%s/auth-policy", siteID), bytes.NewBufferString(updatedPolicyJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify callback was triggered
	if !callbackCalled {
		t.Errorf("expected UpdateAuthPolicy callback to be called")
	}
	if callbackSiteID != siteID {
		t.Errorf("expected callback siteID to be %s, got %s", siteID, callbackSiteID)
	}
	if !strings.Contains(callbackPolicyJSON, `"default":"protected"`) {
		t.Errorf("expected callback policy JSON to contain default:protected, got %s", callbackPolicyJSON)
	}

	// 3. GET again and verify updated values
	req = authedRequestSite(http.MethodGet, fmt.Sprintf("/api/sites/%s/auth-policy", siteID), nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", w.Code)
	}
	var policy2 sites.AuthPolicy
	if err := json.Unmarshal(w.Body.Bytes(), &policy2); err != nil {
		t.Fatalf("failed to unmarshal policy2: %v", err)
	}
	if policy2.Default != "protected" {
		t.Errorf("expected default policy 'protected', got '%s'", policy2.Default)
	}
	if len(policy2.ProtectedPaths) != 1 || policy2.ProtectedPaths[0] != "/books/*" {
		t.Errorf("unexpected protected paths: %v", policy2.ProtectedPaths)
	}
}
