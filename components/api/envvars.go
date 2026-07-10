package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const envVarMigration = `CREATE TABLE IF NOT EXISTS env_vars (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	value_encrypted TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

// EnvVarHandler returns an http.Handler for /api/env endpoints.
// The encryption key is loaded from BIGBASE_ENCRYPTION_KEY (32 raw bytes, base64-encoded).
func (a *API) EnvVarHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/env", a.listEnvVars)
	mux.HandleFunc("POST /api/env", a.createEnvVar)
	mux.HandleFunc("DELETE /api/env/", a.deleteEnvVar)
	return mux
}

func (a *API) ensureEnvVarsTable() error {
	return a.db.Migrate(envVarMigration)
}

type envVarRow struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func (a *API) listEnvVars(w http.ResponseWriter, r *http.Request) {
	if err := a.ensureEnvVarsTable(); err != nil {
		a.logger.Error("ensure env_vars table", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := a.db.QueryContext(ctx, "SELECT id, name, created_at FROM env_vars ORDER BY name")
	if err != nil {
		a.logger.Error("list env_vars", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	result := make([]envVarRow, 0)
	for rows.Next() {
		var ev envVarRow
		if err := rows.Scan(&ev.ID, &ev.Name, &ev.CreatedAt); err != nil {
			a.logger.Error("scan env_var", "error", err)
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		result = append(result, ev)
	}
	if err := rows.Err(); err != nil {
		a.logger.Error("iterate env_vars", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (a *API) createEnvVar(w http.ResponseWriter, r *http.Request) {
	if err := a.ensureEnvVarsTable(); err != nil {
		a.logger.Error("ensure env_vars table", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}
	if req.Name == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if !isValidEnvVarName(req.Name) {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name must be alphanumeric and underscores only"})
		return
	}

	key, err := loadEncryptionKey()
	if err != nil {
		a.logger.Error("load encryption key", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption key not configured"})
		return
	}

	encrypted, err := aesGCMEncrypt(key, req.Value)
	if err != nil {
		a.logger.Error("encrypt value", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := a.db.ExecContext(ctx,
		`INSERT INTO env_vars (name, value_encrypted) VALUES (?, ?)
		 ON CONFLICT(name) DO UPDATE SET value_encrypted=excluded.value_encrypted`,
		req.Name, encrypted)
	if err != nil {
		a.logger.Error("upsert env_var", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	id, _ := res.LastInsertId()
	kernel.WriteJSON(w, http.StatusCreated, map[string]any{"id": id, "name": req.Name})
}

func (a *API) deleteEnvVar(w http.ResponseWriter, r *http.Request) {
	if err := a.ensureEnvVarsTable(); err != nil {
		a.logger.Error("ensure env_vars table", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/env/")
	if name == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := a.db.ExecContext(ctx, "DELETE FROM env_vars WHERE name = ?", name)
	if err != nil {
		a.logger.Error("delete env_var", "name", name, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// loadEncryptionKey reads the 32-byte AES-256 key from BIGBASE_ENCRYPTION_KEY (base64-encoded).
func loadEncryptionKey() ([]byte, error) {
	raw := os.Getenv("BIGBASE_ENCRYPTION_KEY")
	if raw == "" {
		return nil, fmt.Errorf("BIGBASE_ENCRYPTION_KEY is not set")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	return key, nil
}

// aesGCMEncrypt encrypts plaintext with AES-256-GCM and returns base64(nonce||ciphertext).
func aesGCMEncrypt(key []byte, plaintext string) (string, error) {
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
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESGCMEncryptExported is the exported wrapper of aesGCMEncrypt for testing.
func AESGCMEncryptExported(key []byte, plaintext string) (string, error) {
	return aesGCMEncrypt(key, plaintext)
}

// AESGCMDecrypt decrypts a value encrypted by aesGCMEncrypt. Exported for testing.
func AESGCMDecrypt(key []byte, encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
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
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

func isValidEnvVarName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}
