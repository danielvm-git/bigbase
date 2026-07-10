// Package webhooks implements the outbound webhook system for BigBase.
// It stores webhook registrations, delivers HTTP POST events with HMAC-SHA256
// signatures, and retries on failure with exponential backoff (max 5 attempts).
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const (
	componentVersion = "0.1.0"
	maxAttempts      = 5
	deliveryTimeout  = 10 * time.Second
)

// DBer is the database interface required by this component.
type DBer = kernel.DBer

// Options configure the webhooks component.
type Options struct {
	DB     DBer
	Logger kernel.Logger
}

// Component is the ECC component for outbound webhooks.
type Component struct {
	db     DBer
	logger kernel.Logger
	client *http.Client
}

var _ kernel.Component = (*Component)(nil)

// New creates a new Component.
func New(opts Options) *Component {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	return &Component{
		db:     opts.DB,
		logger: logger,
		client: &http.Client{Timeout: deliveryTimeout},
	}
}

func (c *Component) Name() string                  { return "webhooks" }
func (c *Component) Version() string               { return componentVersion }
func (c *Component) Dependencies() []string        { return []string{"db"} }

func (c *Component) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (c *Component) Start(ctx *kernel.Context) error {
	if err := c.db.Migrate(`CREATE TABLE IF NOT EXISTS webhooks (
		id TEXT PRIMARY KEY,
		org_id TEXT NOT NULL,
		url TEXT NOT NULL,
		events TEXT NOT NULL,
		secret TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate webhooks: %w", err)
	}
	c.logger.Info("webhooks component ready")
	return nil
}

func (c *Component) Stop(ctx *kernel.Context) error {
	return nil
}

// Handler returns an http.Handler for /api/webhooks endpoints.
func (c *Component) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/webhooks", c.listWebhooks)
	mux.HandleFunc("POST /api/webhooks", c.createWebhook)
	mux.HandleFunc("DELETE /api/webhooks/", c.deleteWebhook)
	return mux
}

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID        string   `json:"id"`
	OrgID     string   `json:"org_id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Secret    string   `json:"secret,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// EventPayload is the data sent to a webhook endpoint.
type EventPayload struct {
	Event   string         `json:"event"`
	OrgID   string         `json:"org_id"`
	Payload map[string]any `json:"payload"`
}

func (c *Component) listWebhooks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		`SELECT id, org_id, url, events, created_at FROM webhooks ORDER BY created_at`)
	if err != nil {
		c.logger.Error("list webhooks", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	result := make([]Webhook, 0)
	for rows.Next() {
		var wh Webhook
		var eventsJSON string
		if err := rows.Scan(&wh.ID, &wh.OrgID, &wh.URL, &eventsJSON, &wh.CreatedAt); err != nil {
			c.logger.Error("scan webhook", "error", err)
			continue
		}
		_ = json.Unmarshal([]byte(eventsJSON), &wh.Events)
		result = append(result, wh)
	}
	if err := rows.Err(); err != nil {
		c.logger.Error("iterate webhooks", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (c *Component) createWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		OrgID  string   `json:"org_id"`
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Secret string   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}
	if req.URL == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}
	if len(req.Events) == 0 {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "events is required"})
		return
	}

	id, err := kernel.GenerateID()
	if err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	eventsJSON, _ := json.Marshal(req.Events)
	now := time.Now().UTC().Format(time.RFC3339)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := c.db.ExecContext(ctx,
		`INSERT INTO webhooks (id, org_id, url, events, secret, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, req.OrgID, req.URL, string(eventsJSON), req.Secret, now); err != nil {
		c.logger.Error("insert webhook", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, Webhook{
		ID:        id,
		OrgID:     req.OrgID,
		URL:       req.URL,
		Events:    req.Events,
		CreatedAt: now,
	})
}

func (c *Component) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/webhooks/")
	if id == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := c.db.ExecContext(ctx, "DELETE FROM webhooks WHERE id = ?", id)
	if err != nil {
		c.logger.Error("delete webhook", "id", id, "error", err)
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

// Deliver sends the event payload to the webhook's URL once (no retry).
// It signs the body with HMAC-SHA256 using the webhook's secret.
func (c *Component) Deliver(ctx context.Context, hook Webhook, payload EventPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	sig := computeHMAC(hook.Secret, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bigbase-Signature-256", "sha256="+sig)
	req.Header.Set("X-Bigbase-Event", payload.Event)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook delivery failed: status %d", resp.StatusCode)
	}
	return nil
}

// DeliverWithBackoff attempts delivery up to maxAttempts times with exponential
// backoff (100ms, 200ms, 400ms, 800ms, ...). Returns the last error if all fail.
func (c *Component) DeliverWithBackoff(ctx context.Context, hook Webhook, payload EventPayload) error {
	var lastErr error
	backoff := 100 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := c.Deliver(ctx, hook, payload); err != nil {
			lastErr = err
			c.logger.Warn("webhook delivery failed, will retry",
				"attempt", attempt, "max", maxAttempts, "url", hook.URL, "error", err)
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				backoff *= 2
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("webhook delivery failed after %d attempts: %w", maxAttempts, lastErr)
}

// computeHMAC returns the hex-encoded HMAC-SHA256 of body using secret.
func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
