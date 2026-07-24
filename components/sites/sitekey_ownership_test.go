package sites_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

func siteKeyRequest(method, path string, body []byte, siteID string) *http.Request {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(kernel.WithSiteID(req.Context(), siteID))
}

func setupSitesWithDeploy(t *testing.T) (http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	s := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		TriggerDeploy: func(_ context.Context, repoID, branch, siteName, _ string, _ []string, appType string) (*sites.Deployment, error) {
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

// TestSiteKeyOwnership_OwnSiteAllowed: bb_dep_ context for site-a can POST deploy.
func TestSiteKeyOwnership_OwnSiteAllowed(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-a", "alpha", "repo-a", 1)

	body := []byte(`{"branch":"main","app_type":"static"}`)
	req := siteKeyRequest(http.MethodPost, "/api/sites/site-a/deploy", body, "site-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for own-site deploy key, got %d: %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("organization required")) {
		t.Fatalf("site key must not require organization: %s", w.Body.String())
	}
}

// TestSiteKeyOwnership_OtherSite404: key for site-a cannot access site-b.
func TestSiteKeyOwnership_OtherSite404(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-a", "alpha", "repo-a", 1)
	insertRawSite(t, d, "site-b", "beta", "repo-b", 1)

	req := siteKeyRequest(http.MethodGet, "/api/sites/site-b", nil, "site-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-site key, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSiteKeyOwnership_LegacyOrgIDZero: site key works for legacy org_id=0 sites.
func TestSiteKeyOwnership_LegacyOrgIDZero(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-legacy", "legacy", "repo-legacy", 0)

	body := []byte(`{"branch":"main","app_type":"python"}`)
	req := siteKeyRequest(http.MethodPost, "/api/sites/site-legacy/deploy", body, "site-legacy")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for legacy site key deploy, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSiteKeyOwnership_JWTCrossOrgStill404: JWT org path unchanged.
func TestSiteKeyOwnership_JWTCrossOrgStill404(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-other-org", "other", "repo-other", 2)

	req := authedRequestSite(http.MethodGet, "/api/sites/site-other-org", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for JWT cross-org, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSiteKeyOwnership_NoOrgEscalation: site key must not list sites or unlock peers via org_id.
func TestSiteKeyOwnership_NoOrgEscalation(t *testing.T) {
	h, d := setupSitesWithDeploy(t)
	insertRawSite(t, d, "site-a", "alpha", "repo-a", 1)
	insertRawSite(t, d, "site-b", "beta", "repo-b", 1)

	req := siteKeyRequest(http.MethodGet, "/api/sites", nil, "site-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 organization required for site-key list, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "organization required" {
		t.Fatalf("expected organization required, got %v", body)
	}

	// Site-key branch must win when both SiteID and OrgID are present — no peer access.
	req2 := httptest.NewRequest(http.MethodGet, "/api/sites/site-b", nil)
	ctx := kernel.WithSiteID(req2.Context(), "site-a")
	ctx = auth.WithOrgID(ctx, 1)
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("site-key branch must win over org_id (no escalation); got %d: %s", w2.Code, w2.Body.String())
	}
}
