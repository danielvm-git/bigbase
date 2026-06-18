package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookProvider implements the Provider interface by POST-ing JSON messages
// to a configured URL with a Bearer token.
type WebhookProvider struct {
	url     string
	token   string
	client  *http.Client
}

// NewWebhookProvider creates a WebhookProvider that sends messages to the given URL
// with the given Bearer token. Uses a 10-second timeout.
func NewWebhookProvider(url, token string) *WebhookProvider {
	return &WebhookProvider{
		url:   url,
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send POSTs a JSON-encoded message to the configured webhook URL.
func (w *WebhookProvider) Send(ctx context.Context, msg Message) error {
	payload := map[string]string{
		"channel":  msg.Channel,
		"to_addr":  msg.ToAddr,
		"body":     msg.Body,
	}
	if msg.Subject != "" {
		payload["subject"] = msg.Subject
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.token != "" {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook: server returned %d", resp.StatusCode)
	}

	return nil
}
