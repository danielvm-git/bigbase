package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// CacheEntry holds metadata for a single cached dependency archive.
type CacheEntry struct {
	Key       string    `json:"key"`
	SiteID    string    `json:"site_id"`
	RepoID    string    `json:"repo_id"`
	Branch    string    `json:"branch"`
	Size      int64     `json:"size"`
	HitCount  int       `json:"hit_count"`
	CreatedAt time.Time `json:"created_at"`
}

// CacheStats summarises the cache directory.
type CacheStats struct {
	TotalEntries int   `json:"total_entries"`
	TotalSizeB   int64 `json:"total_size_bytes"`
}

// Cache manages on-disk build dependency archives.
// All methods are safe to use even when the cache directory does not yet exist.
// A mutex serialises every read-modify-write so concurrent deploys (each
// launched as its own goroutine) never race on the archive or metadata files.
type Cache struct {
	mu       sync.RWMutex
	dir      string
	maxBytes int64
}

// NewCache returns a Cache rooted at dir.
// maxBytes is the eviction threshold (0 → default 2 GiB).
func NewCache(dir string, maxBytes int64) *Cache {
	if maxBytes <= 0 {
		maxBytes = 2 << 30 // 2 GiB
	}
	return &Cache{dir: dir, maxBytes: maxBytes}
}

// CacheKey computes a deterministic cache key from the repository context and
// the lockfile found in buildDir.
//
// Lockfile priority: package-lock.json → yarn.lock → go.sum → requirements.txt
// Fallback manifests: package.json → go.mod
//
// Returns an error when no usable file is found.
func CacheKey(buildDir, repoID, branch string) (string, error) {
	lockfileHash, err := findLockfileHash(buildDir)
	if err != nil {
		return "", err
	}
	raw := fmt.Sprintf("%s:%s:%s", repoID, branch, lockfileHash)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:16]), nil
}

// findLockfileHash returns the sha256 hex of the first lockfile or manifest
// found in dir, probing in priority order.
func findLockfileHash(dir string) (string, error) {
	candidates := []string{
		"package-lock.json",
		"yarn.lock",
		"go.sum",
		"requirements.txt",
		"package.json",
		"go.mod",
	}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		h := sha256.Sum256(data)
		return hex.EncodeToString(h[:]), nil
	}
	return "", fmt.Errorf("no lockfile or dependency manifest found in %s", dir)
}

// Restore extracts a cached archive into dstDir.
// Returns (true, nil) on cache hit, (false, nil) on miss, and (false, err) on I/O error.
func (c *Cache) Restore(key, dstDir string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tarPath := filepath.Join(c.dir, key+".tar.gz")
	f, err := os.Open(tarPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open cache archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := extractTarGz(f, dstDir); err != nil {
		return false, fmt.Errorf("extract cache: %w", err)
	}
	// Increment hit count; ignore errors (metadata is advisory).
	_ = c.updateMeta(key, func(e *CacheEntry) { e.HitCount++ })
	return true, nil
}

// Save creates a gzipped tar of paths (relative to srcDir) and stores metadata.
// Paths that do not exist in srcDir are silently skipped.
func (c *Cache) Save(key, srcDir, siteID, repoID, branch string, paths []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	size, err := c.writeArchive(key, srcDir, paths)
	if err != nil {
		return err
	}
	return c.writeMeta(key, CacheEntry{
		Key:       key,
		SiteID:    siteID,
		RepoID:    repoID,
		Branch:    branch,
		Size:      size,
		CreatedAt: time.Now().UTC(),
	})
}

// writeArchive tars srcDir/paths into a gzipped file in the cache directory.
func (c *Cache) writeArchive(key, srcDir string, paths []string) (int64, error) {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return 0, fmt.Errorf("create cache dir: %w", err)
	}
	tarPath := filepath.Join(c.dir, key+".tar.gz")
	f, err := os.Create(tarPath)
	if err != nil {
		return 0, fmt.Errorf("create cache archive: %w", err)
	}
	size, archiveErr := createTarGz(f, srcDir, paths)
	_ = f.Close()
	if archiveErr != nil {
		_ = os.Remove(tarPath)
		return 0, fmt.Errorf("create tar: %w", archiveErr)
	}
	return size, nil
}

// Evict removes the oldest cache entries until total size falls below maxBytes.
func (c *Cache) Evict() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := c.listEntries()
	if err != nil {
		return err
	}
	var total int64
	for _, e := range entries {
		total += e.Size
	}
	if total <= c.maxBytes {
		return nil
	}
	// Entries are already sorted oldest-first by ListEntries.
	for _, e := range entries {
		if total <= c.maxBytes {
			break
		}
		if err := c.removeEntry(e.Key); err != nil {
			return err
		}
		total -= e.Size
	}
	return nil
}

// SiteEntries returns all cache entries belonging to siteID, oldest first.
func (c *Cache) SiteEntries(siteID string) ([]CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	all, err := c.listEntries()
	if err != nil {
		return nil, err
	}
	out := make([]CacheEntry, 0, len(all))
	for _, e := range all {
		if e.SiteID == siteID {
			out = append(out, e)
		}
	}
	return out, nil
}

// Clear removes every cache entry. Returns the first removal error, if any.
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := c.listEntries()
	if err != nil {
		return err
	}
	var firstErr error
	for _, e := range entries {
		if rmErr := c.removeEntry(e.Key); rmErr != nil && firstErr == nil {
			firstErr = rmErr
		}
	}
	return firstErr
}

// Prune removes entries whose CreatedAt is older than maxAge and returns the
// number removed. A non-positive maxAge prunes nothing.
func (c *Cache) Prune(maxAge time.Duration) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if maxAge <= 0 {
		return 0, nil
	}
	entries, err := c.listEntries()
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().UTC().Add(-maxAge)
	pruned := 0
	for _, e := range entries {
		if e.CreatedAt.Before(cutoff) {
			if rmErr := c.removeEntry(e.Key); rmErr != nil {
				return pruned, rmErr
			}
			pruned++
		}
	}
	return pruned, nil
}

// MaxBytes returns the current eviction threshold in bytes.
func (c *Cache) MaxBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxBytes
}

// SetMaxBytes updates the live eviction threshold. Non-positive values are
// ignored so the limit can never be zeroed out by a bad request.
func (c *Cache) SetMaxBytes(n int64) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxBytes = n
}

// PurgeSite removes all cache entries associated with siteID.
func (c *Cache) PurgeSite(siteID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := c.listEntries()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.SiteID == siteID {
			if err := c.removeEntry(e.Key); err != nil {
				return err
			}
		}
	}
	return nil
}

// ListEntries returns all cache entries sorted by CreatedAt ascending.
func (c *Cache) ListEntries() ([]CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.listEntries()
}

// listEntries is the lock-free core of ListEntries. Callers must hold c.mu.
func (c *Cache) listEntries() ([]CacheEntry, error) {
	files, err := filepath.Glob(filepath.Join(c.dir, "*.meta.json"))
	if err != nil {
		return nil, err
	}
	entries := make([]CacheEntry, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var e CacheEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

// Stats returns aggregate cache statistics.
func (c *Cache) Stats() (CacheStats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries, err := c.listEntries()
	if err != nil {
		return CacheStats{}, err
	}
	var s CacheStats
	s.TotalEntries = len(entries)
	for _, e := range entries {
		s.TotalSizeB += e.Size
	}
	return s, nil
}

func (c *Cache) writeMeta(key string, entry CacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, key+".meta.json"), data, 0644)
}

func (c *Cache) updateMeta(key string, fn func(*CacheEntry)) error {
	metaPath := filepath.Join(c.dir, key+".meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}
	var e CacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	fn(&e)
	return c.writeMeta(key, e)
}

// removeEntry deletes the archive and metadata for key. A missing file is not
// an error (the entry is already gone), but any other I/O failure is returned
// so Evict and PurgeSite never report a phantom deletion that left bytes on disk.
func (c *Cache) removeEntry(key string) error {
	var firstErr error
	for _, suffix := range []string{".tar.gz", ".meta.json"} {
		if err := os.Remove(filepath.Join(c.dir, key+suffix)); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
