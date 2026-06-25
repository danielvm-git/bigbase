package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// cacheConfigMigration stores the operator-set max cache size as a single row.
// The CLI flag --build-cache-max-size provides the default; a row here overrides it.
const cacheConfigMigration = `CREATE TABLE IF NOT EXISTS deploy_cache_config (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	max_bytes INTEGER NOT NULL
)`

const defaultPruneDays = 7

// ensureCacheConfigTable creates the cache config table.
func (d *Deploy) ensureCacheConfigTable() error {
	return d.db.Migrate(cacheConfigMigration)
}

// loadCacheConfig applies a persisted max-size override to the live cache, if set.
// Called from Start after the table exists. Absence of a row leaves the default.
func (d *Deploy) loadCacheConfig() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var maxBytes int64
	err := d.db.QueryRowContext(ctx, "SELECT max_bytes FROM deploy_cache_config WHERE id = 1").Scan(&maxBytes)
	if err != nil {
		return // no override persisted
	}
	d.cache.SetMaxBytes(maxBytes)
}

// registerCacheRoutes wires the cache-management endpoints onto mux.
func (d *Deploy) registerCacheRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/deploy/cache", d.handleCache)
	mux.HandleFunc("/api/deploy/cache/prune", d.handleCachePrune)
	mux.HandleFunc("/api/deploy/cache/config", d.handleCacheConfig)
	mux.HandleFunc("/api/deploy/cache/site/", d.handleSiteCache)
}

// handleCache serves global cache stats (GET) and clear-all (DELETE).
func (d *Deploy) handleCache(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		stats, err := d.cache.Stats()
		if err != nil {
			d.logger.Error("cache stats", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"total_entries":    stats.TotalEntries,
			"total_size_bytes": stats.TotalSizeB,
			"max_size_bytes":   d.cache.MaxBytes(),
		})
	case http.MethodDelete:
		if err := d.cache.Clear(); err != nil {
			d.logger.Error("cache clear", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleCachePrune removes entries older than max_age_days (default 7).
func (d *Deploy) handleCachePrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	// MaxAgeDays is a pointer so an absent field (nil → default 7) is
	// distinguishable from an explicit 0 (invalid → 400).
	var req struct {
		MaxAgeDays *int `json:"max_age_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}
	days := defaultPruneDays
	if req.MaxAgeDays != nil {
		days = *req.MaxAgeDays
	}
	if days < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "max_age_days must be at least 1"})
		return
	}

	pruned, err := d.cache.Prune(time.Duration(days) * 24 * time.Hour)
	if err != nil {
		d.logger.Error("cache prune", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pruned": pruned})
}

// handleCacheConfig persists and applies the max cache size (PUT).
func (d *Deploy) handleCacheConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		MaxSizeBytes int64 `json:"max_size_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}
	if req.MaxSizeBytes <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "max_size_bytes must be positive"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := d.ensureCacheConfigTable(); err != nil {
		d.logger.Error("ensure cache config table", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO deploy_cache_config (id, max_bytes) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET max_bytes = excluded.max_bytes`, req.MaxSizeBytes)
	if err != nil {
		d.logger.Error("persist cache config", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	d.cache.SetMaxBytes(req.MaxSizeBytes)
	writeJSON(w, http.StatusOK, map[string]any{"max_size_bytes": req.MaxSizeBytes})
}

// handleSiteCache serves per-site cache status (GET) and purge (DELETE).
func (d *Deploy) handleSiteCache(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimPrefix(r.URL.Path, "/api/deploy/cache/site/")
	siteID = strings.Trim(siteID, "/")
	if siteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "site id is required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		entries, err := d.cache.SiteEntries(siteID)
		if err != nil {
			d.logger.Error("site cache entries", "site_id", siteID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		var totalSize int64
		var totalHits int
		for _, e := range entries {
			totalSize += e.Size
			totalHits += e.HitCount
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"entries":          entries,
			"total_size_bytes": totalSize,
			"total_hits":       totalHits,
		})
	case http.MethodDelete:
		if err := d.cache.PurgeSite(siteID); err != nil {
			d.logger.Error("purge site cache", "site_id", siteID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "purged"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
