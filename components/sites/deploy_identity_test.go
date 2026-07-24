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

func decodeErrBody(t *testing.T, w *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v (%s)", err, w.Body.String())
	}
	return body
}

func setupSitesWithCaptureDeploy(t *testing.T, capture *struct {
	RepoID, Branch, SiteName, SiteID, AppType string
}) (http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	s := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		TriggerDeploy: func(_ context.Context, repoID, branch, siteName, siteID string, _ []string, appType string) (*sites.Deployment, error) {
			if capture != nil {
				capture.RepoID = repoID
				capture.Branch = branch
				capture.SiteName = siteName
				capture.SiteID = siteID
				capture.AppType = appType
			}
			return &sites.Deployment{
				ID: "dep-test-1", RepoID: repoID, Branch: branch, Status: "pending",
				URL: "https://" + siteName + ".bigbase.click", Port: 10001,
				AppType: appType, CreatedAt: "2026-07-24T00:00:00Z",
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
	return s.Handler(), d
}

func TestSiteKeyOwnership_MissingSite_StructuredCode(t *testing.T) {
	h, _ := setupSitesWithDeploy(t)

	req := siteKeyRequest(http.MethodGet, "/api/sites/does-not-exist", nil, "site-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	body := decodeErrBody(t, w)
	if body["code"] != "site_not_found" {
		t.Fatalf("expected code=site_not_found, got %v", body)
	}
	if body["hint"] == "" {
		t.Fatalf("expected non-empty hint, got %v", body)
	}
}

func TestSiteKeyOwnership_Mismatch_StructuredCode(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-a", "alpha", "repo-a", 1)
	insertRawSite(t, d, "site-b", "beta", "repo-b", 1)

	req := siteKeyRequest(http.MethodGet, "/api/sites/site-b", nil, "site-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	body := decodeErrBody(t, w)
	if body["code"] != "key_site_mismatch" {
		t.Fatalf("expected code=key_site_mismatch, got %v", body)
	}
	if body["hint"] == "" {
		t.Fatalf("expected hint for mismatch, got %v", body)
	}
}

func TestSiteKeyOwnership_NoOrgIDInjection(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-a", "alpha", "repo-a", 1)
	insertRawSite(t, d, "site-b", "beta", "repo-b", 1)

	req := siteKeyRequest(http.MethodGet, "/api/sites", nil, "site-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without org escalation, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUniqueSiteName_RejectsDuplicateInOrg(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, default_branch) VALUES
		 ('repo-a', 'repo-a', 1, 'main'), ('repo-b', 'repo-b', 1, 'main')`)
	if err != nil {
		t.Fatalf("insert repos: %v", err)
	}
	insertRawSite(t, d, "site-a", "exames", "repo-a", 1)

	body := []byte(`{"name":"exames","git_repo_id":"repo-b","production_branch":"main"}`)
	req := authedRequestSite(http.MethodPost, "/api/sites", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate name, got %d: %s", w.Code, w.Body.String())
	}
	got := decodeErrBody(t, w)
	if got["code"] != "site_name_taken" {
		t.Fatalf("expected code=site_name_taken, got %v", got)
	}
}

func TestRedeploy_UsesSiteGitRepoID(t *testing.T) {
	var capture struct {
		RepoID, Branch, SiteName, SiteID, AppType string
	}
	h, d := setupSitesWithCaptureDeploy(t, &capture)
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, default_branch) VALUES ('repo-canonical', 'danielvm-git-big-exames', 1, 'main')`)
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	insertRawSite(t, d, "site-exames", "exames", "repo-canonical", 1)

	body := []byte(`{"branch":"main","app_type":"static"}`)
	req := siteKeyRequest(http.MethodPost, "/api/sites/site-exames/deploy", body, "site-exames")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if capture.RepoID != "repo-canonical" {
		t.Fatalf("expected git_repo_id repo-canonical, got %q", capture.RepoID)
	}
	if capture.SiteID != "site-exames" {
		t.Fatalf("expected canonical site id site-exames, got %q", capture.SiteID)
	}
}

func TestRedeploy_GitRepoIDAlias_StillBindsCanonicalRepo(t *testing.T) {
	var capture struct {
		RepoID, Branch, SiteName, SiteID, AppType string
	}
	h, d := setupSitesWithCaptureDeploy(t, &capture)
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name, owner_id, default_branch) VALUES ('repo-canonical', 'exames-repo', 1, 'main')`)
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	insertRawSite(t, d, "site-exames", "exames", "repo-canonical", 1)

	body := []byte(`{"branch":"main","app_type":"static"}`)
	req := siteKeyRequest(http.MethodPost, "/api/sites/repo-canonical/deploy", body, "site-exames")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if capture.RepoID != "repo-canonical" || capture.SiteID != "site-exames" {
		t.Fatalf("expected repo-canonical/site-exames, got repo=%q site=%q", capture.RepoID, capture.SiteID)
	}
}
