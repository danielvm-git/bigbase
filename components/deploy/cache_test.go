package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// --- CacheKey tests ---

func TestCacheKey_NodeLockfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"lockfileVersion":2}`), 0644); err != nil {
		t.Fatal(err)
	}

	key, err := CacheKey(dir, "repo1", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-char key, got %d: %q", len(key), key)
	}

	// Deterministic: same inputs → same key
	key2, _ := CacheKey(dir, "repo1", "main")
	if key != key2 {
		t.Errorf("key not deterministic: %q vs %q", key, key2)
	}
}

func TestCacheKey_YarnLockfileFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte("# yarn lockfile v1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	key, err := CacheKey(dir, "repo1", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty key")
	}
}

func TestCacheKey_GoSumFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("module example.com/foo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	key, err := CacheKey(dir, "repo1", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty key")
	}
}

func TestCacheKey_PackageJSONFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
		t.Fatal(err)
	}

	key, err := CacheKey(dir, "repo1", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty key")
	}
}

func TestCacheKey_NoLockfile(t *testing.T) {
	dir := t.TempDir()
	key, err := CacheKey(dir, "repo1", "main")
	if err == nil {
		t.Errorf("expected error for missing lockfile, got key %q", key)
	}
}

func TestCacheKey_DiffersByRepoAndBranch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	k1, _ := CacheKey(dir, "repo1", "main")
	k2, _ := CacheKey(dir, "repo2", "main")
	k3, _ := CacheKey(dir, "repo1", "feat/x")

	if k1 == k2 {
		t.Error("expected different key for different repoID")
	}
	if k1 == k3 {
		t.Error("expected different key for different branch")
	}
}

func TestCacheKey_PackageLockPreferredOverYarn(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"lock":1}`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(`{"lock":2}`), 0644)

	keyPL, _ := CacheKey(dir, "r", "b")

	dirYarn := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirYarn, "yarn.lock"), []byte(`{"lock":2}`), 0644)
	keyYarn, _ := CacheKey(dirYarn, "r", "b")

	if keyPL == keyYarn {
		t.Error("package-lock.json should be preferred over yarn.lock; keys should differ")
	}
}

func TestCacheKey_ChangesWhenLockfileChanges(t *testing.T) {
	dir := t.TempDir()
	lockfile := filepath.Join(dir, "package-lock.json")
	_ = os.WriteFile(lockfile, []byte(`{"version":1}`), 0644)
	k1, _ := CacheKey(dir, "r", "b")

	_ = os.WriteFile(lockfile, []byte(`{"version":2}`), 0644)
	k2, _ := CacheKey(dir, "r", "b")

	if k1 == k2 {
		t.Error("cache key should change when lockfile changes")
	}
}

func TestCacheKey_KnownValue(t *testing.T) {
	dir := t.TempDir()
	lockfileContent := []byte(`{"lockfileVersion":2}`)
	_ = os.WriteFile(filepath.Join(dir, "package-lock.json"), lockfileContent, 0644)

	h := sha256.Sum256(lockfileContent)
	lockfileHash := hex.EncodeToString(h[:])
	raw := fmt.Sprintf("%s:%s:%s", "myrepo", "main", lockfileHash)
	h2 := sha256.Sum256([]byte(raw))
	wantKey := hex.EncodeToString(h2[:16])

	got, err := CacheKey(dir, "myrepo", "main")
	if err != nil {
		t.Fatal(err)
	}
	if got != wantKey {
		t.Errorf("key mismatch: got %q want %q", got, wantKey)
	}
}

// --- CacheSaveRestore tests ---

func TestCacheSaveRestore_HitAndMiss(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	// Source: a directory with node_modules
	srcDir := t.TempDir()
	nmDir := filepath.Join(srcDir, "node_modules", "pkg")
	_ = os.MkdirAll(nmDir, 0755)
	_ = os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("module.exports={}"), 0644)

	const key = "abc123"

	// Miss before save
	hit, err := cache.Restore(key, t.TempDir())
	if err != nil {
		t.Fatalf("Restore error on miss: %v", err)
	}
	if hit {
		t.Error("expected cache miss before save")
	}

	// Save
	if err := cache.Save(key, srcDir, "siteA", "repoA", "main", []string{"node_modules"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Restore into fresh directory
	dstDir := t.TempDir()
	hit, err = cache.Restore(key, dstDir)
	if err != nil {
		t.Fatalf("Restore error on hit: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit after save")
	}

	// Verify restored content
	restored := filepath.Join(dstDir, "node_modules", "pkg", "index.js")
	data, err := os.ReadFile(restored)
	if err != nil {
		t.Fatalf("restored file missing: %v", err)
	}
	if string(data) != "module.exports={}" {
		t.Errorf("restored content mismatch: %q", data)
	}
}

func TestCacheSaveRestore_MetadataWritten(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "dep.txt"), []byte("data"), 0644)

	if err := cache.Save("key1", srcDir, "siteA", "repoA", "feat", []string{"dep.txt"}); err != nil {
		t.Fatal(err)
	}

	entries, err := cache.ListEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Key != "key1" {
		t.Errorf("wrong key: %q", e.Key)
	}
	if e.SiteID != "siteA" {
		t.Errorf("wrong site_id: %q", e.SiteID)
	}
	if e.RepoID != "repoA" {
		t.Errorf("wrong repo_id: %q", e.RepoID)
	}
	if e.Branch != "feat" {
		t.Errorf("wrong branch: %q", e.Branch)
	}
	if e.Size <= 0 {
		t.Error("size should be > 0")
	}
	if e.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}
}

func TestCacheSaveRestore_HitCountIncremented(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0644)
	_ = cache.Save("k", srcDir, "", "", "", []string{"f.txt"})

	cache.Restore("k", t.TempDir()) //nolint:errcheck
	cache.Restore("k", t.TempDir()) //nolint:errcheck

	entries, _ := cache.ListEntries()
	if len(entries) == 0 {
		t.Fatal("no entries")
	}
	if entries[0].HitCount != 2 {
		t.Errorf("expected HitCount=2, got %d", entries[0].HitCount)
	}
}

func TestCacheSaveRestore_MissingPathSkipped(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)
	srcDir := t.TempDir()

	// Save with a path that doesn't exist — should not error, just produce empty archive
	err := cache.Save("k2", srcDir, "", "", "", []string{"nonexistent_dir"})
	if err != nil {
		t.Fatalf("Save should succeed even for missing paths: %v", err)
	}
}

func TestCacheEvict_RemovesOldestFirst(t *testing.T) {
	cacheDir := t.TempDir()
	// maxBytes=1 ensures any two entries trigger eviction regardless of compression ratio
	cache := NewCache(cacheDir, 1)

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0644)

	_ = cache.Save("old-entry", srcDir, "", "", "", []string{"f.txt"})
	time.Sleep(10 * time.Millisecond) // ensure different created_at
	_ = cache.Save("new-entry", srcDir, "", "", "", []string{"f.txt"})

	if err := cache.Evict(); err != nil {
		t.Fatalf("Evict error: %v", err)
	}

	entries, _ := cache.ListEntries()
	for _, e := range entries {
		if e.Key == "old-entry" {
			t.Error("old-entry should have been evicted")
		}
	}
}

func TestCachePurgeSite(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0644)

	_ = cache.Save("k-siteA-1", srcDir, "siteA", "r1", "main", []string{"f.txt"})
	_ = cache.Save("k-siteA-2", srcDir, "siteA", "r1", "feat", []string{"f.txt"})
	_ = cache.Save("k-siteB-1", srcDir, "siteB", "r2", "main", []string{"f.txt"})

	if err := cache.PurgeSite("siteA"); err != nil {
		t.Fatalf("PurgeSite error: %v", err)
	}

	entries, _ := cache.ListEntries()
	for _, e := range entries {
		if e.SiteID == "siteA" {
			t.Errorf("entry for siteA should have been purged: %q", e.Key)
		}
	}
	found := false
	for _, e := range entries {
		if e.SiteID == "siteB" {
			found = true
		}
	}
	if !found {
		t.Error("siteB entry should still exist")
	}
}

func TestCacheStats(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("hello"), 0644)
	_ = cache.Save("s1", srcDir, "site1", "r1", "main", []string{"f.txt"})
	_ = cache.Save("s2", srcDir, "site1", "r1", "feat", []string{"f.txt"})

	stats, err := cache.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalEntries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.TotalEntries)
	}
	if stats.TotalSizeB <= 0 {
		t.Error("total size should be > 0")
	}
}

// TestCacheRemoveEntry_PropagatesError verifies eviction surfaces real I/O
// failures instead of silently believing a file was removed while it remains
// on disk (which would let the cache grow past maxBytes forever).
func TestCacheRemoveEntry_PropagatesError(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	// Make "bad.tar.gz" a non-empty directory so os.Remove fails with ENOTEMPTY
	// rather than a missing-file (IsNotExist) case, which is legitimately ignored.
	tarAsDir := filepath.Join(cacheDir, "bad.tar.gz")
	if err := os.Mkdir(tarAsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tarAsDir, "child"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := cache.removeEntry("bad"); err == nil {
		t.Error("removeEntry should propagate the os.Remove failure, got nil")
	}
}

// TestCacheConcurrentRestore_NoRace exercises the mutex: many goroutines restore
// the same key at once. Run with -race to catch the missing-lock regression.
// The serialized read-modify-write must also leave HitCount exactly == N.
func TestCacheConcurrentRestore_NoRace(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir, 0)

	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("payload"), 0644)
	if err := cache.Save("shared", srcDir, "site1", "r1", "main", []string{"f.txt"}); err != nil {
		t.Fatal(err)
	}

	const n = 16
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			dst := t.TempDir()
			if hit, err := cache.Restore("shared", dst); err != nil || !hit {
				t.Errorf("Restore: hit=%v err=%v", hit, err)
			}
		}()
	}
	wg.Wait()

	entries, err := cache.ListEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].HitCount != n {
		t.Errorf("expected HitCount %d, got %d (lost increments)", n, entries[0].HitCount)
	}
}
