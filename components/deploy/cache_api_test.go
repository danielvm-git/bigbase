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

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// newCacheTestDeploy builds a Deploy backed by an in-memory DB and an isolated
// cache directory, with Start() run so the config table exists.
func newCacheTestDeploy(t *testing.T) (*Deploy, func()) {
	t.Helper()
	database := db.New(db.Options{Path: ":memory:", Logger: noopLogger{}})
	if err := database.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	dep := New(Options{DB: database, Logger: noopLogger{}, CacheDir: t.TempDir()})
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
	d2 := New(Options{DB: d.db, Logger: noopLogger{}, CacheDir: t.TempDir()})
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
