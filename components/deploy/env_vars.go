package deploy

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// FetchSiteEnvVars queries site_env_vars for the given site and returns
// decrypted KEY=value strings. When buildTime is true, only is_build_time=1
// vars are returned; when false, only is_runtime=1 vars are returned.
// Returns nil (no error) when the table does not exist yet.
func (d *Deploy) FetchSiteEnvVars(ctx context.Context, siteID string, buildTime bool) ([]string, error) {
	if siteID == "" {
		return nil, nil
	}

	col := "is_runtime"
	if buildTime {
		col = "is_build_time"
	}

	rows, err := d.db.QueryContext(ctx,
		"SELECT key, value_encrypted FROM site_env_vars WHERE site_id = ? AND "+col+" = 1",
		siteID)
	if err != nil {
		// Table may not exist in older deployments — treat as empty.
		if isNoSuchTable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch site env vars: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var key, encrypted string
		if err := rows.Scan(&key, &encrypted); err != nil {
			continue
		}
		value, err := d.decryptEnvValue(encrypted)
		if err != nil {
			d.logger.Warn("decrypt env var", "key", key, "error", err)
			continue
		}
		out = append(out, key+"="+value)
	}
	return out, nil
}

// EncryptEnvValue encrypts a plaintext value using the deploy component's
// configured encryption key. Exported for use in tests. Returns plaintext
// unchanged when no key is configured.
func (d *Deploy) EncryptEnvValue(plaintext string) (string, error) {
	if d.envKey == nil {
		return plaintext, nil
	}
	return encryptAESGCM(d.envKey, plaintext)
}

func (d *Deploy) decryptEnvValue(stored string) (string, error) {
	if d.envKey == nil {
		return stored, nil
	}
	return decryptAESGCM(d.envKey, stored)
}

func isNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "does not exist")
}

func parseEnvEncryptionKey(keyHex string) ([]byte, error) {
	if keyHex == "" {
		return nil, nil
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("env encryption key must be 32 bytes (64 hex chars), got %d", len(key))
	}
	return key, nil
}

func encryptAESGCM(key []byte, plaintext string) (string, error) {
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

func decryptAESGCM(key []byte, stored string) (string, error) {
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
