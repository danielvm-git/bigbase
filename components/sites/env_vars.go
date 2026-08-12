package sites

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danielvm/bigbase/components/internal/envcrypto"
	"github.com/danielvm/bigbase/kernel"
)

var validEnvKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

const (
	maxEnvKeyLen   = 128
	maxEnvValueLen = 64 * 1024
)

// EnvVar represents a site-scoped environment variable.
// Value is the plaintext in write/single-read paths.
// ValuePreview is a masked preview (last 4 chars) returned by the list endpoint.
type EnvVar struct {
	ID           string `json:"id"`
	SiteID       string `json:"site_id"`
	Key          string `json:"key"`
	Value        string `json:"value,omitempty"`
	ValuePreview string `json:"value_preview"`
	IsBuildTime  bool   `json:"is_build_time"`
	IsRuntime    bool   `json:"is_runtime"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (s *Sites) migrateEnvVars() error {
	return s.db.Migrate(`CREATE TABLE IF NOT EXISTS site_env_vars (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value_encrypted TEXT NOT NULL DEFAULT '',
		is_build_time INTEGER NOT NULL DEFAULT 0,
		is_runtime INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(site_id, key)
	)`)
}

func (s *Sites) handleEnvVarsRoute(w http.ResponseWriter, r *http.Request, siteID string, parts []string) {
	if !s.requireSiteOwnership(r.Context(), w, siteID) {
		return
	}

	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.listEnvVars(w, r, siteID)
		case http.MethodPost:
			s.createEnvVar(w, r, siteID)
		default:
			kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	if len(parts) == 3 {
		key := parts[2]
		switch r.Method {
		case http.MethodPut:
			s.updateEnvVar(w, r, siteID, key)
		case http.MethodDelete:
			s.deleteEnvVar(w, r, siteID, key)
		default:
			kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
}

// listEnvVars returns env vars with values masked server-side (F03).
// Full plaintext is never sent over the wire in list responses.
func (s *Sites) listEnvVars(w http.ResponseWriter, r *http.Request, siteID string) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT id, site_id, key, value_encrypted, is_build_time, is_runtime, created_at, updated_at
		 FROM site_env_vars WHERE site_id = ? ORDER BY key`, siteID)
	if err != nil {
		s.logger.Error("list env vars", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	items, scanErr := s.scanEnvVarRows(rows)
	// F04/F05: a scan failure or a row-iteration error both mean the result set is
	// incomplete — treat them the same way (fail the request) rather than silently
	// returning partial data with a 200.
	if scanErr != nil {
		s.logger.Error("list env vars scan", "error", scanErr)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("list env vars rows iteration", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

// previewDecryptError is the visible sentinel preview shown for a row whose stored
// value cannot be decrypted (e.g. key rotation or DB corruption). It is distinct
// from a normal short-value mask so operators can tell corruption from a 4-char
// secret rather than seeing an indistinguishable "••••".
const previewDecryptError = "<decrypt error>"

// scanEnvVarRows decrypts values and populates ValuePreview; Value is left empty
// so the list response never leaks full secrets (F03). Returns the first scan
// error encountered (F06) so the caller can fail the request.
func (s *Sites) scanEnvVarRows(rows *sql.Rows) ([]EnvVar, error) {
	out := make([]EnvVar, 0)
	for rows.Next() {
		var ev EnvVar
		var encrypted string
		var bt, rt int
		if err := rows.Scan(&ev.ID, &ev.SiteID, &ev.Key, &encrypted, &bt, &rt, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			s.logger.Error("scan env var row", "error", err) // F06: log instead of silently skip
			return nil, err
		}
		ev.IsBuildTime, ev.IsRuntime = bt == 1, rt == 1
		plaintext, err := envcrypto.Decrypt(s.encryptionKey, encrypted)
		if err != nil {
			s.logger.Warn("decrypt env var", "key", ev.Key, "error", err)
			ev.ValuePreview = previewDecryptError // F04: surface corruption, don't masquerade as a value
		} else {
			ev.ValuePreview = envcrypto.MaskValue(plaintext) // F03: mask server-side
			// ev.Value intentionally left empty in list responses
		}
		out = append(out, ev)
	}
	return out, nil
}

func (s *Sites) createEnvVar(w http.ResponseWriter, r *http.Request, siteID string) {
	key, value, buildTime, isRuntime, ok := s.decodeEnvVarBody(w, r, true)
	if !ok {
		return
	}
	encrypted, err := envcrypto.Encrypt(s.encryptionKey, value)
	if err != nil {
		s.logger.Error("encrypt env var", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	ev, err := s.insertEnvVar(r.Context(), siteID, key, value, encrypted, buildTime, isRuntime)
	if err != nil {
		s.handleEnvVarWriteErr(w, "insert env var", err)
		return
	}
	kernel.WriteJSON(w, http.StatusCreated, ev)
}

func (s *Sites) updateEnvVar(w http.ResponseWriter, r *http.Request, siteID, key string) {
	if len(key) > maxEnvKeyLen || !validEnvKey.MatchString(key) {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid env var key"})
		return
	}
	_, value, buildTime, isRuntime, ok := s.decodeEnvVarBody(w, r, false)
	if !ok {
		return
	}
	existingID, err := s.lookupEnvVarID(r.Context(), siteID, key)
	if err != nil {
		s.handleEnvVarLookupErr(w, "lookup env var", err)
		return
	}
	encrypted, err := envcrypto.Encrypt(s.encryptionKey, value)
	if err != nil {
		s.logger.Error("encrypt env var", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.execUpdateEnvVar(r.Context(), siteID, key, encrypted, buildTime, isRuntime, now); err != nil {
		s.logger.Error("update env var", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	ev := EnvVar{
		ID:           existingID,
		SiteID:       siteID,
		Key:          key,
		ValuePreview: envcrypto.MaskValue(value),
		IsBuildTime:  buildTime,
		IsRuntime:    isRuntime,
		UpdatedAt:    now,
	}
	kernel.WriteJSON(w, http.StatusOK, ev)
}

func (s *Sites) deleteEnvVar(w http.ResponseWriter, r *http.Request, siteID, key string) {
	if _, err := s.lookupEnvVarID(r.Context(), siteID, key); err != nil {
		s.handleEnvVarLookupErr(w, "lookup env var for delete", err)
		return
	}
	if _, err := s.db.ExecContext(r.Context(), "DELETE FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key); err != nil {
		s.logger.Error("delete env var", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Sites) decodeEnvVarBody(w http.ResponseWriter, r *http.Request, requireKey bool) (key, value string, buildTime, isRuntime bool, ok bool) {
	var body struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		IsBuildTime bool   `json:"is_build_time"`
		IsRuntime   bool   `json:"is_runtime"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxEnvValueLen+maxEnvKeyLen+1024)).Decode(&body); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if requireKey && body.Key == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}
	if body.Key != "" && (len(body.Key) > maxEnvKeyLen || !validEnvKey.MatchString(body.Key)) {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid env var key"})
		return
	}
	if len(body.Value) > maxEnvValueLen {
		kernel.WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "env var value exceeds size limit"})
		return
	}
	if !body.IsBuildTime && !body.IsRuntime {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one env var flag is required"})
		return
	}
	return body.Key, body.Value, body.IsBuildTime, body.IsRuntime, true
}

func (s *Sites) insertEnvVar(ctx context.Context, siteID, key, plainValue, encrypted string, buildTime, isRuntime bool) (EnvVar, error) {
	id, err := kernel.GenerateID()
	if err != nil {
		return EnvVar{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO site_env_vars (id, site_id, key, value_encrypted, is_build_time, is_runtime, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, siteID, key, encrypted, boolToInt(buildTime), boolToInt(isRuntime), now, now)
	if err != nil {
		return EnvVar{}, err
	}
	return EnvVar{
		ID:           id,
		SiteID:       siteID,
		Key:          key,
		ValuePreview: envcrypto.MaskValue(plainValue),
		IsBuildTime:  buildTime,
		IsRuntime:    isRuntime,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (s *Sites) execUpdateEnvVar(ctx context.Context, siteID, key, encrypted string, buildTime, isRuntime bool, now string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE site_env_vars SET value_encrypted = ?, is_build_time = ?, is_runtime = ?, updated_at = ?
		 WHERE site_id = ? AND key = ?`,
		encrypted, boolToInt(buildTime), boolToInt(isRuntime), now, siteID, key)
	return err
}

func (s *Sites) lookupEnvVarID(ctx context.Context, siteID, key string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key).Scan(&id)
	return id, err
}

func (s *Sites) handleEnvVarLookupErr(w http.ResponseWriter, msg string, err error) {
	if err == sql.ErrNoRows {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "env var not found"})
		return
	}
	s.logger.Error(msg, "error", err)
	kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func (s *Sites) handleEnvVarWriteErr(w http.ResponseWriter, msg string, err error) {
	if strings.Contains(err.Error(), "UNIQUE constraint") {
		kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "env var with this key already exists"})
		return
	}
	s.logger.Error(msg, "error", err)
	kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ListSiteEnvVars returns env vars for a site with masked values (for MCP tools).
func (s *Sites) ListSiteEnvVars(ctx context.Context, siteID string) ([]EnvVar, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, site_id, key, value_encrypted, is_build_time, is_runtime, created_at, updated_at
		 FROM site_env_vars WHERE site_id = ? ORDER BY key`, siteID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items, scanErr := s.scanEnvVarRows(rows)
	if scanErr != nil {
		return nil, scanErr
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// SetSiteEnvVars upserts multiple env vars for a site (for MCP tools).
func (s *Sites) SetSiteEnvVars(ctx context.Context, siteID string, vars map[string]string) ([]EnvVar, error) {
	if len(vars) > 1000 {
		return nil, fmt.Errorf("env var batch exceeds size limit")
	}
	out := make([]EnvVar, 0, len(vars))
	for key, value := range vars {
		if len(key) > maxEnvKeyLen || !validEnvKey.MatchString(key) || len(value) > maxEnvValueLen {
			return nil, fmt.Errorf("invalid env var payload")
		}
		encrypted, err := envcrypto.Encrypt(s.encryptionKey, value)
		if err != nil {
			return nil, fmt.Errorf("encrypt env var")
		}
		existingID, lookupErr := s.lookupEnvVarID(ctx, siteID, key)
		now := time.Now().UTC().Format(time.RFC3339)
		if lookupErr == sql.ErrNoRows {
			ev, insertErr := s.insertEnvVar(ctx, siteID, key, value, encrypted, false, true)
			if insertErr != nil {
				return nil, fmt.Errorf("persist env vars")
			}
			out = append(out, ev)
		} else if lookupErr != nil {
			return nil, fmt.Errorf("lookup env vars")
		} else {
			if err := s.execUpdateEnvVar(ctx, siteID, key, encrypted, false, true, now); err != nil {
				return nil, fmt.Errorf("persist env vars")
			}
			out = append(out, EnvVar{ID: existingID, SiteID: siteID, Key: key, ValuePreview: envcrypto.MaskValue(value), IsRuntime: true, UpdatedAt: now})
		}
	}
	return out, nil
}

// DeleteSiteEnvVar deletes a single env var for a site (for MCP tools).
func (s *Sites) DeleteSiteEnvVar(ctx context.Context, siteID, key string) error {
	if _, err := s.lookupEnvVarID(ctx, siteID, key); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("env var %q not found", key)
		}
		return err
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key)
	return err
}

// parseEncryptionKey validates and decodes the hex env encryption key.
// Returns nil, nil when empty (encryption disabled).
func parseEncryptionKey(keyHex string) ([]byte, error) {
	return envcrypto.ParseKey(keyHex)
}

// ParseRootEncryptionKey is the composition-root seam for canonical key
// validation; the crypto primitive remains owned by envcrypto.
func ParseRootEncryptionKey(raw string) ([]byte, error) { return envcrypto.ParseRootKey(raw) }

// ParseLegacyEncryptionKey is an explicit migration/test-only input.
func ParseLegacyEncryptionKey(raw string) ([]byte, error) { return envcrypto.ParseLegacyKey(raw) }
