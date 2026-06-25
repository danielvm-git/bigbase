// Package envcrypto provides shared AES-256-GCM encryption helpers for
// site environment variables. It is internal to the components/ tree and must
// not be imported by external packages.
package envcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// ParseKey decodes a 64-char hex string into a 32-byte AES-256 key.
// Returns nil, nil when keyHex is empty (encryption disabled).
//
// Encryption is deliberately optional: a nil key puts Encrypt/Decrypt into a
// no-op pass-through mode so the feature works out-of-the-box in development
// without key management. The trade-off is that values are then stored as
// plaintext at rest — callers are expected to log a warning at startup when the
// key is missing/invalid so operators notice the misconfiguration in production.
func ParseKey(keyHex string) ([]byte, error) {
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

// Encrypt encrypts plaintext with AES-256-GCM and returns a base64-encoded
// nonce+ciphertext blob. When key is nil the plaintext is returned unchanged.
func Encrypt(key []byte, plaintext string) (string, error) {
	if key == nil {
		return plaintext, nil
	}
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

// Decrypt reverses Encrypt. When key is nil the stored value is returned
// unchanged (no-op mode).
func Decrypt(key []byte, stored string) (string, error) {
	if key == nil {
		return stored, nil
	}
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
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// MaskValue returns a masked representation of a secret value that is safe to
// expose in API list responses. The last 4 characters are revealed; values of 4
// characters or fewer are fully masked. Counting is done in runes (not bytes) so
// multibyte UTF-8 values are never sliced mid-character.
func MaskValue(v string) string {
	r := []rune(v)
	if len(r) <= 4 {
		return "••••"
	}
	return "••••" + string(r[len(r)-4:])
}
