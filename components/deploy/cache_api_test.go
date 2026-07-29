package deploy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// newCacheTestDeploy builds a Deploy backed by an in-memory DB and an isolated
// cache directory, with Start() run so the config table exists.
func newCacheTestDeploy(t *testing.T) (*Deploy, func()) {
	t.Helper()
	database := db.New(db.Options{Path: ":memory:", Logger: kernel.NoopLogger{}})
	if err := database.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	dep := New(Options{DB: database, Logger: kernel.NoopLogger{}, CacheDir: t.TempDir()})
	if err := dep.Start(&kernel.Context{}); err != nil {
		t.Fatalf("deploy start: %v", err)
	}
	cleanup := func() {
		_ = dep.Stop(&kernel.Context{})
		_ = database.Stop(&kernel.Context{})
	}
	return dep, cleanup
}

// seedCacheEntry stores a cache entry for a site using a throwaway source dir.
func seedCacheEntry(t *testing.T, d *Deploy, key, siteID string) {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := d.cache.Save(key, src, siteID, "repo", "main", []string{"f.txt"}); err != nil {
		t.Fatalf("seed cache entry %s: %v", key, err)
	}
}

func TestCacheAPI_GlobalStats(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()
	seedCacheEntry(t, d, "k1", "siteA")
	seedCacheEntry(t, d, "k2", "siteB")

	req := httptest.NewRequest(http.MethodGet, "/api/deploy/cache", nil)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		TotalEntries int   `json:"total_entries"`
		TotalSizeB   int64 `json:"total_size_bytes"`
		MaxSizeB     int64 `json:"max_size_bytes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TotalEntries != 2 {
		t.Errorf("expected 2 entries, got %d", body.TotalEntries)
	}
	if body.MaxSizeB != 2<<30 {
		t.Errorf("expected default max 2 GiB, got %d", body.MaxSizeB)
	}
}

func TestCacheAPI_ClearAll(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()
	seedCacheEntry(t, d, "k1", "siteA")

	req := httptest.NewRequest(http.MethodDelete, "/api/deploy/cache", nil)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	entries, _ := d.cache.ListEntries()
	if len(entries) != 0 {
		t.Errorf("expected cache cleared, got %d entries", len(entries))
	}
}

func TestCacheAPI_Prune(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()
	seedCacheEntry(t, d, "fresh", "siteA")
	seedCacheEntry(t, d, "stale", "siteB")
	old := time.Now().UTC().Add(-10 * 24 * time.Hour)
	_ = d.cache.updateMeta("stale", func(e *CacheEntry) { e.CreatedAt = old })

	req := httptest.NewRequest(http.MethodPost, "/api/deploy/cache/prune",
		strings.NewReader(`{"max_age_days":7}`))
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Pruned int `json:"pruned"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", body.Pruned)
	}
}

func TestCacheAPI_Prune_RejectsZeroDays(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()

	// Explicit 0 must be rejected, not silently defaulted to 7.
	req := httptest.NewRequest(http.MethodPost, "/api/deploy/cache/prune",
		strings.NewReader(`{"max_age_days":0}`))
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for max_age_days=0, got %d", rec.Code)
	}
}

func TestCacheAPI_Prune_DefaultsWhenAbsent(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()
	seedCacheEntry(t, d, "stale", "siteA")
	old := time.Now().UTC().Add(-10 * 24 * time.Hour)
	_ = d.cache.updateMeta("stale", func(e *CacheEntry) { e.CreatedAt = old })

	// Empty body → default 7-day cutoff applies, pruning the 10-day-old entry.
	req := httptest.NewRequest(http.MethodPost, "/api/deploy/cache/prune", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Pruned int `json:"pruned"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Pruned != 1 {
		t.Errorf("expected default prune to remove 1, got %d", body.Pruned)
	}
}

func TestCacheAPI_SetConfig_Persists(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPut, "/api/deploy/cache/config",
		strings.NewReader(`{"max_size_bytes":524288000}`)) // 500 MiB
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if d.cache.MaxBytes() != 524288000 {
		t.Errorf("live limit not updated: %d", d.cache.MaxBytes())
	}
	// Persisted: a fresh Deploy on the same DB should load the override.
	d2 := New(Options{DB: d.db, Logger: kernel.NoopLogger{}, CacheDir: t.TempDir()})
	if err := d2.Start(&kernel.Context{}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d2.Stop(&kernel.Context{}) }()
	if d2.cache.MaxBytes() != 524288000 {
		t.Errorf("persisted limit not restored on restart: %d", d2.cache.MaxBytes())
	}
}

func TestCacheAPI_SetConfig_RejectsInvalid(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPut, "/api/deploy/cache/config",
		strings.NewReader(`{"max_size_bytes":0}`))
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-positive size, got %d", rec.Code)
	}
}

func TestCacheAPI_SiteStatusAndPurge(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()
	seedCacheEntry(t, d, "a1", "siteA")
	seedCacheEntry(t, d, "a2", "siteA")
	seedCacheEntry(t, d, "b1", "siteB")

	// GET site status
	req := httptest.NewRequest(http.MethodGet, "/api/deploy/cache/site/siteA", nil)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries    []CacheEntry `json:"entries"`
		TotalSizeB int64        `json:"total_size_bytes"`
		TotalHits  int          `json:"total_hits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 2 {
		t.Errorf("expected 2 entries for siteA, got %d", len(body.Entries))
	}

	// DELETE site cache
	delReq := httptest.NewRequest(http.MethodDelete, "/api/deploy/cache/site/siteA", nil)
	delRec := httptest.NewRecorder()
	d.Handler().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on purge, got %d", delRec.Code)
	}
	remaining, _ := d.cache.SiteEntries("siteA")
	if len(remaining) != 0 {
		t.Errorf("siteA should be purged, got %d", len(remaining))
	}
	// siteB untouched.
	if other, _ := d.cache.SiteEntries("siteB"); len(other) != 1 {
		t.Errorf("siteB should be intact, got %d", len(other))
	}
}

func TestCacheAPI_SiteStatus_RequiresID(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/deploy/cache/site/", nil)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing site id, got %d", rec.Code)
	}
}

// TestCacheAPI_SiteCache_PreventsCrossOrgIDOR is the regression guard for the
// fix in handleSiteCache (issue #180): a caller authenticated as org A must be
// denied (403) when reading or purging org B's site cache. Before the fix,
// handleSiteCache took the site_id path param straight to the cache store with
// no ownership check, leaking build-cache contents across tenants.
func TestCacheAPI_SiteCache_PreventsCrossOrgIDOR(t *testing.T) {
	d, cleanup := newCacheTestDeploy(t)
	defer cleanup()

	// Seed a sites table with org_id so verifySiteOwnership has a row to check.
	if _, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL DEFAULT '',
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		org_id INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("create sites table: %v", err)
	}
	if _, err := d.db.Exec(`INSERT INTO sites (id, name, git_repo_id, org_id) VALUES
		('siteA', 'A', 'r', 100),
		('siteB', 'B', 'r', 200)`); err != nil {
		t.Fatalf("seed sites: %v", err)
	}
	seedCacheEntry(t, d, "a1", "siteA")
	seedCacheEntry(t, d, "b1", "siteB")

	// Org A (id=100) tries to read org B's (id=200) site cache -> 403.
	getReq := httptest.NewRequest(http.MethodGet, "/api/deploy/cache/site/siteB", nil)
	getReq = getReq.WithContext(auth.WithOrgID(getReq.Context(), 100))
	getRec := httptest.NewRecorder()
	d.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusForbidden {
		t.Fatalf("cross-org GET site cache: expected 403, got %d (%s)", getRec.Code, getRec.Body.String())
	}

	// Org A tries to purge org B's site cache -> 403, and the cache is intact.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/deploy/cache/site/siteB", nil)
	delReq = delReq.WithContext(auth.WithOrgID(delReq.Context(), 100))
	delRec := httptest.NewRecorder()
	d.Handler().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusForbidden {
		t.Fatalf("cross-org DELETE site cache: expected 403, got %d (%s)", delRec.Code, delRec.Body.String())
	}
	if remaining, _ := d.cache.SiteEntries("siteB"); len(remaining) != 1 {
		t.Errorf("org B cache should be intact after denied purge, got %d entries", len(remaining))
	}

	// Sanity: org A reading its OWN site cache still works (the denial is a real
	// ownership check, not a blanket 403 — the deploy-key 403 regression class).
	ownReq := httptest.NewRequest(http.MethodGet, "/api/deploy/cache/site/siteA", nil)
	ownReq = ownReq.WithContext(auth.WithOrgID(ownReq.Context(), 100))
	ownRec := httptest.NewRecorder()
	d.Handler().ServeHTTP(ownRec, ownReq)
	if ownRec.Code != http.StatusOK {
		t.Fatalf("own-org GET site cache: expected 200, got %d (%s)", ownRec.Code, ownRec.Body.String())
	}
}
