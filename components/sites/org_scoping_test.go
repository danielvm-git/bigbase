package sites_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
)

// insertRawSite inserts a minimal site row with an explicit org_id, bypassing
// the HTTP create path so tests can simulate both legacy (org_id=0) and
// real cross-org (org_id=2+) data directly.
func insertRawSite(t *testing.T, d *db.DB, id, name, repoID string, orgID int64) {
	t.Helper()
	_, err := d.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at, org_id)
		 VALUES (?, ?, ?, 'main', './', '', datetime('now'), ?)`,
		id, name, repoID, orgID)
	if err != nil {
		t.Fatalf("insert site %s: %v", id, err)
	}
}

// TestListSites_IncludesLegacyOrglessSites verifies that sites created before
// org_id scoping existed (org_id=0, the column's DEFAULT) remain visible to
// any authenticated org — they must not become permanently invisible just
// because the caller's real org_id can never equal 0.
func TestListSites_IncludesLegacyOrglessSites(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-legacy", "legacy-site", "repo-legacy", 0)
	insertRawSite(t, d, "site-mine", "my-site", "repo-mine", 1)

	req := authedRequestSite(http.MethodGet, "/api/sites", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	names := siteNamesFromList(t, w)
	if !names["legacy-site"] {
		t.Errorf("expected legacy (org_id=0) site to be visible, got: %v", names)
	}
	if !names["my-site"] {
		t.Errorf("expected caller's own site to be visible, got: %v", names)
	}
}

// TestListSites_StillIsolatesRealCrossOrgSites locks in that the legacy-data
// carve-out does NOT reopen BUG-136 — a site genuinely owned by a different,
// nonzero org must still be excluded.
func TestListSites_StillIsolatesRealCrossOrgSites(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-other-org", "other-org-site", "repo-other", 2)
	insertRawSite(t, d, "site-mine", "my-site", "repo-mine", 1)

	req := authedRequestSite(http.MethodGet, "/api/sites", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	names := siteNamesFromList(t, w)
	if names["other-org-site"] {
		t.Errorf("expected cross-org site NOT to be visible, got: %v", names)
	}
	if !names["my-site"] {
		t.Errorf("expected caller's own site to be visible, got: %v", names)
	}
}

// TestRequireSiteOwnership_AllowsLegacyOrglessSite verifies single-site
// handlers (get/delete/redeploy/manifest/etc, all gated by
// requireSiteOwnership) work for legacy org_id=0 sites.
func TestRequireSiteOwnership_AllowsLegacyOrglessSite(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-legacy", "legacy-site", "repo-legacy", 0)

	req := authedRequestSite(http.MethodGet, "/api/sites/site-legacy", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for legacy site, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRequireSiteOwnership_StillDeniesRealCrossOrgSite locks in that a site
// genuinely owned by a different, nonzero org still 404s.
func TestRequireSiteOwnership_StillDeniesRealCrossOrgSite(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-other-org", "other-org-site", "repo-other", 2)

	req := authedRequestSite(http.MethodGet, "/api/sites/site-other-org", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-org site, got %d: %s", w.Code, w.Body.String())
	}
}

func siteNamesFromList(t *testing.T, w *httptest.ResponseRecorder) map[string]bool {
	t.Helper()
	var body struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	names := make(map[string]bool, len(body.Data))
	for _, site := range body.Data {
		names[site.Name] = true
	}
	return names
}
