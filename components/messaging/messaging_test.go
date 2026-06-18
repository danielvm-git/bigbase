package messaging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/messaging"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupMessaging(t *testing.T) (*messaging.Messaging, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	m := messaging.New(messaging.Options{DB: d, Logger: logger})

	recorder := new(recorderProvider)
	m.RegisterProvider("email", recorder)
	m.RegisterProvider("sms", recorder)
	m.RegisterProvider("push", recorder)

	k.Register(m)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return m, m.Handler()
}

type recorderProvider struct {
	messages []messaging.Message
}

func (r *recorderProvider) Send(ctx context.Context, msg messaging.Message) error {
	r.messages = append(r.messages, msg)
	return nil
}

func (r *recorderProvider) Messages() []messaging.Message {
	return r.messages
}

func postJSON(t *testing.T, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func getRequest(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestMessagingImplementsComponent(t *testing.T) {
	var _ kernel.Component = (*messaging.Messaging)(nil)
}

func TestMessagingName(t *testing.T) {
	m := &messaging.Messaging{}
	if got := m.Name(); got != "messaging" {
		t.Fatalf("expected Name()='messaging', got '%s'", got)
	}
}

func TestMessagingVersion(t *testing.T) {
	m := &messaging.Messaging{}
	if got := m.Version(); got == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestMessagingDependencies(t *testing.T) {
	m := &messaging.Messaging{}
	deps := m.Dependencies()
	if len(deps) != 1 || deps[0] != "db" {
		t.Fatalf("expected dependency on 'db', got %v", deps)
	}
}

func TestMessagingHooks(t *testing.T) {
	m := &messaging.Messaging{}
	if got := m.Hooks(); len(got) != 0 {
		t.Fatalf("expected no hooks, got %v", got)
	}
}

func TestEmailSendSuccess(t *testing.T) {
	_, handler := setupMessaging(t)

	resp := postJSON(t, handler, "/api/messaging/email", map[string]string{
		"to":      "user@example.com",
		"subject": "Hello",
		"body":    "World",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var msg messaging.Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if msg.Channel != "email" {
		t.Fatalf("expected channel 'email', got '%s'", msg.Channel)
	}
	if msg.ToAddr != "user@example.com" {
		t.Fatalf("expected to_addr 'user@example.com', got '%s'", msg.ToAddr)
	}
	if msg.Subject != "Hello" {
		t.Fatalf("expected subject 'Hello', got '%s'", msg.Subject)
	}
	if msg.Status != "sent" {
		t.Fatalf("expected status 'sent', got '%s'", msg.Status)
	}
}

func TestEmailMissingFields(t *testing.T) {
	_, handler := setupMessaging(t)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing to", map[string]string{"subject": "Hi", "body": "test"}},
		{"missing subject", map[string]string{"to": "a@b.com", "body": "test"}},
		{"missing body", map[string]string{"to": "a@b.com", "subject": "Hi"}},
		{"empty body", map[string]string{"to": "a@b.com", "subject": "Hi", "body": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, handler, "/api/messaging/email", tt.body)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestEmailWrongMethod(t *testing.T) {
	_, handler := setupMessaging(t)

	req := httptest.NewRequest("GET", "/api/messaging/email", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestEmailInvalidJSON(t *testing.T) {
	_, handler := setupMessaging(t)

	req := httptest.NewRequest("POST", "/api/messaging/email", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSMSSendSuccess(t *testing.T) {
	_, handler := setupMessaging(t)

	resp := postJSON(t, handler, "/api/messaging/sms", map[string]string{
		"to":      "+1234567890",
		"message": "Hello world",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var msg messaging.Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if msg.Channel != "sms" {
		t.Fatalf("expected channel 'sms', got '%s'", msg.Channel)
	}
	if msg.ToAddr != "+1234567890" {
		t.Fatalf("expected to_addr '+1234567890', got '%s'", msg.ToAddr)
	}
	if msg.Body != "Hello world" {
		t.Fatalf("expected body 'Hello world', got '%s'", msg.Body)
	}
	if msg.Status != "sent" {
		t.Fatalf("expected status 'sent', got '%s'", msg.Status)
	}
}

func TestSMSMissingFields(t *testing.T) {
	_, handler := setupMessaging(t)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing to", map[string]string{"message": "Hi"}},
		{"missing message", map[string]string{"to": "+123"}},
		{"empty message", map[string]string{"to": "+123", "message": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, handler, "/api/messaging/sms", tt.body)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestSMSWrongMethod(t *testing.T) {
	_, handler := setupMessaging(t)

	req := httptest.NewRequest("GET", "/api/messaging/sms", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestPushSendSuccess(t *testing.T) {
	_, handler := setupMessaging(t)

	resp := postJSON(t, handler, "/api/messaging/push", map[string]string{
		"token": "device-token-123",
		"title": "Alert",
		"body":  "Something happened",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var msg messaging.Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if msg.Channel != "push" {
		t.Fatalf("expected channel 'push', got '%s'", msg.Channel)
	}
	if msg.ToAddr != "device-token-123" {
		t.Fatalf("expected to_addr 'device-token-123', got '%s'", msg.ToAddr)
	}
	if msg.Subject != "Alert" {
		t.Fatalf("expected subject 'Alert', got '%s'", msg.Subject)
	}
	if msg.Body != "Something happened" {
		t.Fatalf("expected body 'Something happened', got '%s'", msg.Body)
	}
	if msg.Status != "sent" {
		t.Fatalf("expected status 'sent', got '%s'", msg.Status)
	}
}

func TestPushMissingFields(t *testing.T) {
	_, handler := setupMessaging(t)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing token", map[string]string{"title": "Hi", "body": "test"}},
		{"missing title", map[string]string{"token": "abc", "body": "test"}},
		{"missing body", map[string]string{"token": "abc", "title": "Hi"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, handler, "/api/messaging/push", tt.body)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestPushWrongMethod(t *testing.T) {
	_, handler := setupMessaging(t)

	req := httptest.NewRequest("GET", "/api/messaging/push", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestListMessagesEmpty(t *testing.T) {
	_, handler := setupMessaging(t)

	resp := getRequest(t, handler, "/api/messaging/messages")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := body["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("expected empty list, got %d items", len(data))
	}
}

func TestListMessagesAfterSends(t *testing.T) {
	_, handler := setupMessaging(t)

	postJSON(t, handler, "/api/messaging/email", map[string]string{
		"to": "a@b.com", "subject": "S1", "body": "B1",
	})
	postJSON(t, handler, "/api/messaging/sms", map[string]string{
		"to": "+1", "message": "B2",
	})

	resp := getRequest(t, handler, "/api/messaging/messages")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := body["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(data))
	}
}

func TestListMessagesWrongMethod(t *testing.T) {
	_, handler := setupMessaging(t)

	req := httptest.NewRequest("POST", "/api/messaging/messages", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestProviderCalled(t *testing.T) {
	m, handler := setupMessaging(t)

	postJSON(t, handler, "/api/messaging/email", map[string]string{
		"to": "a@b.com", "subject": "S1", "body": "B1",
	})

	p := m.Provider("email")
	if p == nil {
		t.Fatal("expected email provider to be registered")
	}
	rp, ok := p.(*recorderProvider)
	if !ok {
		t.Fatal("expected recorderProvider")
	}
	if len(rp.Messages()) != 1 {
		t.Fatalf("expected 1 message sent to provider, got %d", len(rp.Messages()))
	}
	if rp.Messages()[0].ToAddr != "a@b.com" {
		t.Fatalf("expected to_addr 'a@b.com', got '%s'", rp.Messages()[0].ToAddr)
	}
}

func TestWebhookProvider(t *testing.T) {
	// Track received requests
	var receivedBody []byte
	var receivedAuth string
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	wp := messaging.NewWebhookProvider(srv.URL+"/webhook", "test-token-abc")

	msg := messaging.Message{
		ID:      "msg-1",
		Channel: "telegram",
		ToAddr:  "123456",
		Body:    "Hello from test",
	}

	err := wp.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Verify the webhook received the correct data
	if receivedPath != "/webhook" {
		t.Fatalf("expected path '/webhook', got '%s'", receivedPath)
	}
	if receivedAuth != "Bearer test-token-abc" {
		t.Fatalf("expected Authorization 'Bearer test-token-abc', got '%s'", receivedAuth)
	}

	var payload map[string]any
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("failed to parse webhook body: %v", err)
	}
	if payload["body"] != "Hello from test" {
		t.Fatalf("expected body 'Hello from test', got '%v'", payload["body"])
	}
	if payload["channel"] != "telegram" {
		t.Fatalf("expected channel 'telegram', got '%v'", payload["channel"])
	}
}
