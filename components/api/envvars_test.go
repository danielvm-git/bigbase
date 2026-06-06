package api_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/api"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func setupAPIWithKey(t *testing.T) (*api.API, http.Handler) {
	t.Helper()
	// 32-byte key, base64-encoded
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	t.Setenv("BIGBASE_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))

	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := api.New(api.Options{DB: d, Logger: logger})

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return a, a.EnvVarHandler()
}

func TestEnvVars(t *testing.T) {
	t.Run("list returns empty on fresh db", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		req := httptest.NewRequest(http.MethodGet, "/api/env", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body, _ := io.ReadAll(w.Body)
		var resp map[string]any
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		data, ok := resp["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got %T", resp["data"])
		}
		if len(data) != 0 {
			t.Errorf("expected empty list, got %d items", len(data))
		}
	})

	t.Run("create stores encrypted value", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		body := `{"name":"DB_PASSWORD","value":"supersecret"}`
		req := httptest.NewRequest(http.MethodPost, "/api/env", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if resp["name"] != "DB_PASSWORD" {
			t.Errorf("expected name DB_PASSWORD, got %v", resp["name"])
		}
	})

	t.Run("create returns 400 for missing name", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		body := `{"value":"something"}`
		req := httptest.NewRequest(http.MethodPost, "/api/env", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("create returns 400 for invalid name chars", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		body := `{"name":"my-var","value":"val"}`
		req := httptest.NewRequest(http.MethodPost, "/api/env", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("list after create shows variable name without decrypted value", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		// Create
		body := `{"name":"API_KEY","value":"secret123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/env", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %d", w.Code)
		}

		// List
		req = httptest.NewRequest(http.MethodGet, "/api/env", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("list failed: %d", w.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]any)
		if len(data) != 1 {
			t.Fatalf("expected 1 item, got %d", len(data))
		}
		item := data[0].(map[string]any)
		if item["name"] != "API_KEY" {
			t.Errorf("expected name API_KEY, got %v", item["name"])
		}
		// value_encrypted must NOT appear in the list response (only name+id+created_at)
		if _, hasVal := item["value_encrypted"]; hasVal {
			t.Error("list must not expose value_encrypted")
		}
	})

	t.Run("delete removes variable", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		// Create
		body := `{"name":"TO_DELETE","value":"val"}`
		req := httptest.NewRequest(http.MethodPost, "/api/env", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		// Delete
		req = httptest.NewRequest(http.MethodDelete, "/api/env/TO_DELETE", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", w.Code)
		}

		// List — should be empty
		req = httptest.NewRequest(http.MethodGet, "/api/env", nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]any)
		if len(data) != 0 {
			t.Errorf("expected 0 items after delete, got %d", len(data))
		}
	})

	t.Run("delete non-existent returns 404", func(t *testing.T) {
		_, h := setupAPIWithKey(t)

		req := httptest.NewRequest(http.MethodDelete, "/api/env/NONEXISTENT", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("encryption round-trip", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i + 7)
		}
		plaintext := "my-secret-value-123"
		encrypted, err := api.AESGCMEncryptExported(key, plaintext)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		decrypted, err := api.AESGCMDecrypt(key, encrypted)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if decrypted != plaintext {
			t.Errorf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
		}
	})
}
