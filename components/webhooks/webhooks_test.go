package webhooks_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/webhooks"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupWebhooks(t *testing.T) (*webhooks.Component, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	wh := webhooks.New(webhooks.Options{DB: d, Logger: logger})

	k.Register(d)
	k.Register(wh)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return wh, wh.Handler()
}

func TestWebhookDelivery(t *testing.T) {
	t.Run("register webhook via API", func(t *testing.T) {
		_, h := setupWebhooks(t)

		body := `{"org_id":"org1","url":"https://example.com/hook","events":["user.created"],"secret":"abc123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["id"] == nil {
			t.Error("expected id in response")
		}
	})

	t.Run("list webhooks", func(t *testing.T) {
		_, h := setupWebhooks(t)

		for i := 0; i < 2; i++ {
			body := `{"org_id":"org1","url":"https://example.com/hook","events":["push"],"secret":"s"}`
			req := httptest.NewRequest(http.MethodPost, "/api/webhooks", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("create %d: %d", i, w.Code)
			}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/webhooks", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]any)
		if len(data) != 2 {
			t.Errorf("expected 2 webhooks, got %d", len(data))
		}
	})

	t.Run("delete webhook", func(t *testing.T) {
		_, h := setupWebhooks(t)

		body := `{"org_id":"org1","url":"https://example.com/hook","events":["push"],"secret":"s"}`
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		var createResp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &createResp)
		id := createResp["id"].(string)

		req = httptest.NewRequest(http.MethodDelete, "/api/webhooks/"+id, nil)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", w.Code)
		}
	})

	t.Run("deliver sends POST with HMAC-SHA256 signature", func(t *testing.T) {
		wh, _ := setupWebhooks(t)

		var received atomic.Bool
		var sigHeader string
		var bodyReceived []byte
		var mu sync.Mutex

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			sigHeader = r.Header.Get("X-Bigbase-Signature-256")
			bodyReceived, _ = io.ReadAll(r.Body)
			received.Store(true)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		secret := "test-secret"
		payload := webhooks.EventPayload{
			Event:   "user.created",
			OrgID:   "org1",
			Payload: map[string]any{"user_id": "u123"},
		}

		hook := webhooks.Webhook{
			ID:     "wh-test",
			OrgID:  "org1",
			URL:    srv.URL,
			Events: []string{"user.created"},
			Secret: secret,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := wh.Deliver(ctx, hook, payload); err != nil {
			t.Fatalf("Deliver: %v", err)
		}

		// Wait for server to receive
		deadline := time.Now().Add(2 * time.Second)
		for !received.Load() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if !received.Load() {
			t.Fatal("server did not receive webhook delivery")
		}

		// Validate HMAC-SHA256 signature
		mu.Lock()
		defer mu.Unlock()
		if sigHeader == "" {
			t.Fatal("X-Bigbase-Signature-256 header missing")
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(bodyReceived)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if sigHeader != expected {
			t.Errorf("signature mismatch: got %s, want %s", sigHeader, expected)
		}
	})

	t.Run("deliver retries on failure with exponential backoff (max 5 attempts)", func(t *testing.T) {
		wh, _ := setupWebhooks(t)

		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusInternalServerError) // always fail
		}))
		defer srv.Close()

		hook := webhooks.Webhook{
			ID:     "wh-retry",
			OrgID:  "org1",
			URL:    srv.URL,
			Events: []string{"push"},
			Secret: "s",
		}

		payload := webhooks.EventPayload{Event: "push", OrgID: "org1", Payload: map[string]any{}}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// DeliverWithBackoff should attempt up to maxAttempts and return error
		err := wh.DeliverWithBackoff(ctx, hook, payload)
		if err == nil {
			t.Error("expected error after all retry attempts failed")
		}

		got := int(attempts.Load())
		if got != 5 {
			t.Errorf("expected exactly 5 attempts, got %d", got)
		}
	})

	t.Run("deliver to invalid URL returns error", func(t *testing.T) {
		wh, _ := setupWebhooks(t)

		hook := webhooks.Webhook{
			ID:     "wh-bad",
			OrgID:  "org1",
			URL:    "http://127.0.0.1:1", // nothing listening
			Events: []string{"push"},
			Secret: "s",
		}
		payload := webhooks.EventPayload{Event: "push", OrgID: "org1", Payload: map[string]any{}}
		err := wh.Deliver(context.Background(), hook, payload)
		if err == nil {
			t.Error("expected error for unreachable URL")
		}
	})
}
