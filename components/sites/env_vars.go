package sites

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var validEnvKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// EnvVar represents a site-scoped environment variable.
type EnvVar struct {
	ID          string `json:"id"`
	SiteID      string `json:"site_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsBuildTime bool   `json:"is_build_time"`
	IsRuntime   bool   `json:"is_runtime"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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
	// parts[1] == "env-vars"
	if len(parts) == 2 {
		// /api/sites/:id/env-vars
		switch r.Method {
		case http.MethodGet:
			s.listEnvVars(w, r, siteID)
		case http.MethodPost:
			s.createEnvVar(w, r, siteID)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	if len(parts) == 3 {
		// /api/sites/:id/env-vars/:key
		key := parts[2]
		switch r.Method {
		case http.MethodPut:
			s.updateEnvVar(w, r, siteID, key)
		case http.MethodDelete:
			s.deleteEnvVar(w, r, siteID, key)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
}

func (s *Sites) listEnvVars(w http.ResponseWriter, r *http.Request, siteID string) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT id, site_id, key, value_encrypted, is_build_time, is_runtime, created_at, updated_at
		 FROM site_env_vars WHERE site_id = ? ORDER BY key`, siteID)
	if err != nil {
		s.logger.Error("list env vars", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	out := make([]EnvVar, 0)
	for rows.Next() {
		var ev EnvVar
		var encrypted string
		var buildTime, runtime int
		if err := rows.Scan(&ev.ID, &ev.SiteID, &ev.Key, &encrypted, &buildTime, &runtime, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			continue
		}
		ev.IsBuildTime = buildTime == 1
		ev.IsRuntime = runtime == 1
		ev.Value, _ = s.decryptValue(encrypted)
		out = append(out, ev)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Sites) createEnvVar(w http.ResponseWriter, r *http.Request, siteID string) {
	var req struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		IsBuildTime bool   `json:"is_build_time"`
		IsRuntime   bool   `json:"is_runtime"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}
	if !validEnvKey.MatchString(req.Key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key must be uppercase alphanumeric + underscore, starting with a letter"})
		return
	}

	encrypted, err := s.encryptValue(req.Value)
	if err != nil {
		s.logger.Error("encrypt env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	id, err := generateID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	buildTime := 0
	if req.IsBuildTime {
		buildTime = 1
	}
	runtime := 1
	if !req.IsRuntime {
		runtime = 0
	}

	_, err = s.db.ExecContext(r.Context(),
		`INSERT INTO site_env_vars (id, site_id, key, value_encrypted, is_build_time, is_runtime, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, siteID, req.Key, encrypted, buildTime, runtime, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "env var with this key already exists"})
			return
		}
		s.logger.Error("insert env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, EnvVar{
		ID:          id,
		SiteID:      siteID,
		Key:         req.Key,
		Value:       req.Value,
		IsBuildTime: req.IsBuildTime,
		IsRuntime:   req.IsRuntime,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (s *Sites) updateEnvVar(w http.ResponseWriter, r *http.Request, siteID, key string) {
	var req struct {
		Value       string `json:"value"`
		IsBuildTime bool   `json:"is_build_time"`
		IsRuntime   bool   `json:"is_runtime"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Check exists
	var existingID string
	err := s.db.QueryRowContext(r.Context(),
		"SELECT id FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key).Scan(&existingID)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "env var not found"})
		return
	}
	if err != nil {
		s.logger.Error("lookup env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	encrypted, err := s.encryptValue(req.Value)
	if err != nil {
		s.logger.Error("encrypt env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	buildTime := 0
	if req.IsBuildTime {
		buildTime = 1
	}
	runtime := 1
	if !req.IsRuntime {
		runtime = 0
	}

	_, err = s.db.ExecContext(r.Context(),
		`UPDATE site_env_vars SET value_encrypted = ?, is_build_time = ?, is_runtime = ?, updated_at = ?
		 WHERE site_id = ? AND key = ?`,
		encrypted, buildTime, runtime, now, siteID, key)
	if err != nil {
		s.logger.Error("update env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, EnvVar{
		ID:          existingID,
		SiteID:      siteID,
		Key:         key,
		Value:       req.Value,
		IsBuildTime: req.IsBuildTime,
		IsRuntime:   req.IsRuntime,
		UpdatedAt:   now,
	})
}

func (s *Sites) deleteEnvVar(w http.ResponseWriter, r *http.Request, siteID, key string) {
	var existingID string
	err := s.db.QueryRowContext(r.Context(),
		"SELECT id FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key).Scan(&existingID)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "env var not found"})
		return
	}
	if err != nil {
		s.logger.Error("lookup env var for delete", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	_, err = s.db.ExecContext(r.Context(),
		"DELETE FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key)
	if err != nil {
		s.logger.Error("delete env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// encryptValue encrypts plaintext using AES-256-GCM if an encryption key is configured.
// When no key is set, the value is stored as-is (development mode).
func (s *Sites) encryptValue(plaintext string) (string, error) {
	if s.encryptionKey == nil {
		return plaintext, nil
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptValue decrypts a stored value. Returns the value as-is when no encryption key is set.
func (s *Sites) decryptValue(stored string) (string, error) {
	if s.encryptionKey == nil {
		return stored, nil
	}
	data, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// parseEncryptionKey parses a hex-encoded 32-byte key string.
func parseEncryptionKey(keyHex string) ([]byte, error) {
	if keyHex == "" {
		return nil, nil
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("env encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return key, nil
}
