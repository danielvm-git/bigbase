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

// TestListSites_AdminSeesLegacyOrglessSites verifies that sites created before
// org_id scoping existed (org_id=0, the column's DEFAULT) remain visible to an
// ADMIN caller — the admin owns the bootstrap/migration path and must be able
// to see and reassign legacy sites. (BUG-2026-07-28T000002: the org_id=0
// carve-out is now admin-only; it was previously visible to every org, which
// was a cross-tenant leak.)
func TestListSites_AdminSeesLegacyOrglessSites(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-legacy", "legacy-site", "repo-legacy", 0)
	insertRawSite(t, d, "site-mine", "my-site", "repo-mine", 1)

	req := authedRequestSiteAs(http.MethodGet, "/api/sites", nil, "admin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	names := siteNamesFromList(t, w)
	if !names["legacy-site"] {
		t.Errorf("expected admin to see legacy (org_id=0) site, got: %v", names)
	}
	if !names["my-site"] {
		t.Errorf("expected admin to see own site, got: %v", names)
	}
}

// TestListSites_NonAdminDoesNotSeeLegacyOrglessSites is the BUG-2026-07-28T000002
// regression guard: a regular (non-admin) org MUST NOT see org_id=0 legacy sites.
// Previously the `OR s.org_id = 0` fallback returned those rows to every org,
// leaking cross-tenant data. Now only an admin sees them.
func TestListSites_NonAdminDoesNotSeeLegacyOrglessSites(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-legacy", "legacy-site", "repo-legacy", 0)
	insertRawSite(t, d, "site-mine", "my-site", "repo-mine", 1)

	req := authedRequestSiteAs(http.MethodGet, "/api/sites", nil, "user")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	names := siteNamesFromList(t, w)
	if names["legacy-site"] {
		t.Errorf("non-admin must NOT see legacy (org_id=0) site — cross-tenant leak (BUG-000002), got: %v", names)
	}
	if !names["my-site"] {
		t.Errorf("expected non-admin to see own site, got: %v", names)
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

// TestRequireSiteOwnership_AdminAllowsLegacyOrglessSite verifies single-site
// handlers (get/delete/redeploy/manifest/etc, all gated by
// requireSiteOwnership) work for legacy org_id=0 sites WHEN the caller is an
// admin. Non-admin access to org_id=0 sites is covered by the
// TestRequireSiteOwnership_NonAdminDeniesLegacyOrglessSite guard below.
func TestRequireSiteOwnership_AdminAllowsLegacyOrglessSite(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-legacy", "legacy-site", "repo-legacy", 0)

	req := authedRequestSiteAs(http.MethodGet, "/api/sites/site-legacy", nil, "admin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin accessing legacy site, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRequireSiteOwnership_NonAdminDeniesLegacyOrglessSite is the single-site
// counterpart to TestListSites_NonAdminDoesNotSeeLegacyOrglessSites: a non-admin
// must get 404 on a legacy org_id=0 site (BUG-2026-07-28T000002).
func TestRequireSiteOwnership_NonAdminDeniesLegacyOrglessSite(t *testing.T) {
	_, h, d := setupSitesWithHandler(t)

	insertRawSite(t, d, "site-legacy", "legacy-site", "repo-legacy", 0)

	req := authedRequestSiteAs(http.MethodGet, "/api/sites/site-legacy", nil, "user")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-admin accessing legacy (org_id=0) site, got %d: %s", w.Code, w.Body.String())
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
