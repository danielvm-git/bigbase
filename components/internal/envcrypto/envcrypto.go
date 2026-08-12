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

// ParseRootKey decodes the canonical base64-encoded 32-byte root key.
// An empty or malformed value is always an error; callers must not silently
// enter plaintext mode when production configuration is missing.
func ParseRootKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, fmt.Errorf("root encryption key is required")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("root encryption key must be base64-encoded 32 bytes")
	}
	return key, nil
}

// ParseLegacyKey decodes the pre-e89 hexadecimal key format. It is retained
// only for explicit migration and test inputs; production composition must use
// ParseRootKey instead.
func ParseLegacyKey(keyHex string) ([]byte, error) {
	if keyHex == "" {
		return nil, fmt.Errorf("legacy encryption key is required")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("legacy encryption key must be 32 bytes")
	}
	return key, nil
}

// ParseKey is the legacy/test parser. New composition code must call
// ParseRootKey and pass legacy values only through an explicit migration path.
func ParseKey(keyHex string) ([]byte, error) {
	if keyHex == "" {
		return nil, nil
	}
	return ParseLegacyKey(keyHex)
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
