package kernel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteJSON serializes data as JSON and writes it to the response with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// GenerateID produces a cryptographically random hex-encoded 32-character identifier.
func GenerateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// NoopLogger implements the Logger interface as a no-op. Useful as a default
// logger for tests or when a component doesn't need its own logger.
type NoopLogger struct{}

func (NoopLogger) Info(msg string, args ...any)  {}
func (NoopLogger) Warn(msg string, args ...any)  {}
func (NoopLogger) Error(msg string, args ...any) {}
func (NoopLogger) Debug(msg string, args ...any) {}
