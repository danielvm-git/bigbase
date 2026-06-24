package sites

import (
	"context"
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
	if len(parts) == 2 {
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
	writeJSON(w, http.StatusOK, map[string]any{"data": s.scanEnvVarRows(rows)})
}

func (s *Sites) scanEnvVarRows(rows interface{ Next() bool; Scan(...any) error }) []EnvVar {
	out := make([]EnvVar, 0)
	for rows.Next() {
		var ev EnvVar
		var encrypted string
		var bt, rt int
		if err := rows.Scan(&ev.ID, &ev.SiteID, &ev.Key, &encrypted, &bt, &rt, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			continue
		}
		ev.IsBuildTime, ev.IsRuntime = bt == 1, rt == 1
		ev.Value, _ = s.decryptValue(encrypted)
		out = append(out, ev)
	}
	return out
}

func (s *Sites) createEnvVar(w http.ResponseWriter, r *http.Request, siteID string) {
	key, value, buildTime, isRuntime, ok := s.decodeEnvVarBody(w, r, true)
	if !ok {
		return
	}
	encrypted, err := s.encryptValue(value)
	if err != nil {
		s.logger.Error("encrypt env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	ev, err := s.insertEnvVar(r.Context(), siteID, key, value, encrypted, buildTime, isRuntime)
	if err != nil {
		s.handleEnvVarWriteErr(w, "insert env var", err)
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

func (s *Sites) updateEnvVar(w http.ResponseWriter, r *http.Request, siteID, key string) {
	_, value, buildTime, isRuntime, ok := s.decodeEnvVarBody(w, r, false)
	if !ok {
		return
	}
	existingID, err := s.lookupEnvVarID(r.Context(), siteID, key)
	if err != nil {
		s.handleEnvVarLookupErr(w, "lookup env var", err)
		return
	}
	encrypted, err := s.encryptValue(value)
	if err != nil {
		s.logger.Error("encrypt env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.execUpdateEnvVar(r.Context(), siteID, key, encrypted, buildTime, isRuntime, now); err != nil {
		s.logger.Error("update env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, EnvVar{ID: existingID, SiteID: siteID, Key: key, Value: value, IsBuildTime: buildTime, IsRuntime: isRuntime, UpdatedAt: now})
}

func (s *Sites) deleteEnvVar(w http.ResponseWriter, r *http.Request, siteID, key string) {
	if _, err := s.lookupEnvVarID(r.Context(), siteID, key); err != nil {
		s.handleEnvVarLookupErr(w, "lookup env var for delete", err)
		return
	}
	if _, err := s.db.ExecContext(r.Context(), "DELETE FROM site_env_vars WHERE site_id = ? AND key = ?", siteID, key); err != nil {
		s.logger.Error("delete env var", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if requireKey && body.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}
	if requireKey && !validEnvKey.MatchString(body.Key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key must be uppercase alphanumeric + underscore, starting with a letter"})
		return
	}
	return body.Key, body.Value, body.IsBuildTime, body.IsRuntime, true
}

func (s *Sites) insertEnvVar(ctx context.Context, siteID, key, plainValue, encrypted string, buildTime, isRuntime bool) (EnvVar, error) {
	id, err := generateID()
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
	return EnvVar{ID: id, SiteID: siteID, Key: key, Value: plainValue, IsBuildTime: buildTime, IsRuntime: isRuntime, CreatedAt: now, UpdatedAt: now}, nil
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
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "env var not found"})
		return
	}
	s.logger.Error(msg, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
func (s *Sites) handleEnvVarWriteErr(w http.ResponseWriter, msg string, err error) {
	if strings.Contains(err.Error(), "UNIQUE constraint") {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "env var with this key already exists"})
		return
	}
	s.logger.Error(msg, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// encryptValue uses AES-256-GCM when a key is configured; stores as-is otherwise.
func (s *Sites) encryptValue(plaintext string) (string, error) {
	if s.encryptionKey == nil {
		return plaintext, nil
	}
	return siteEncryptAESGCM(s.encryptionKey, plaintext)
}
func (s *Sites) decryptValue(stored string) (string, error) {
	if s.encryptionKey == nil {
		return stored, nil
	}
	return siteDecryptAESGCM(s.encryptionKey, stored)
}
func siteEncryptAESGCM(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
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
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}
func siteDecryptAESGCM(key []byte, stored string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
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
