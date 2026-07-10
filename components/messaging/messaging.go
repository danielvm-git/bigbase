package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Message struct {
	ID        string `json:"id"`
	Channel   string `json:"channel"`
	ToAddr    string `json:"to_addr"`
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type sendRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body"`
	Title   string `json:"title,omitempty"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

type Provider interface {
	Send(ctx context.Context, msg Message) error
}

type Messaging struct {
	db        DBer
	logger    kernel.Logger
	smtpHost  string
	smtpPort  int
	smtpUser  string
	smtpPass  string
	smtpFrom  string
	providers map[string]Provider
}

var _ kernel.Component = (*Messaging)(nil)

type Options struct {
	DB       DBer
	Logger   kernel.Logger
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func New(opts Options) *Messaging {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	if opts.SMTPHost == "" {
		opts.SMTPHost = "localhost"
	}
	if opts.SMTPPort == 0 {
		opts.SMTPPort = 1025
	}
	if opts.SMTPFrom == "" {
		opts.SMTPFrom = "no-reply@bigbase.local"
	}
	return &Messaging{
		db:        opts.DB,
		logger:    logger,
		smtpHost:  opts.SMTPHost,
		smtpPort:  opts.SMTPPort,
		smtpUser:  opts.SMTPUser,
		smtpPass:  opts.SMTPPass,
		smtpFrom:  opts.SMTPFrom,
		providers: make(map[string]Provider),
	}
}

func (m *Messaging) RegisterProvider(channel string, p Provider) {
	m.providers[channel] = p
}

func (m *Messaging) Provider(channel string) Provider {
	return m.providers[channel]
}

func (m *Messaging) Name() string                  { return "messaging" }
func (m *Messaging) Version() string               { return version }
func (m *Messaging) Dependencies() []string        { return []string{"db"} }

func (m *Messaging) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (m *Messaging) Start(ctx *kernel.Context) error {
	if err := m.db.Migrate(`CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		channel TEXT NOT NULL,
		to_addr TEXT NOT NULL,
		subject TEXT,
		body TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'sent',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate messages table: %w", err)
	}
	m.logger.Info("messaging component ready")
	return nil
}

func (m *Messaging) Stop(ctx *kernel.Context) error {
	return nil
}

func (m *Messaging) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/messaging/email", m.handleEmail)
	mux.HandleFunc("/api/messaging/sms", m.handleSMS)
	mux.HandleFunc("/api/messaging/push", m.handlePush)
	mux.HandleFunc("/api/messaging/telegram", m.handleTelegram)
	mux.HandleFunc("/api/messaging/messages", m.handleList)
	return mux
}

func (m *Messaging) send(ctx context.Context, channel, toAddr, subject, body string) (*Message, error) {
	id, err := kernel.GenerateID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	msg := &Message{
		ID:        id,
		Channel:   channel,
		ToAddr:    toAddr,
		Subject:   subject,
		Body:      body,
		Status:    "sent",
		CreatedAt: now,
	}

	if p, ok := m.providers[channel]; ok {
		if err := p.Send(ctx, *msg); err != nil {
			msg.Status = "failed"
			m.logger.Error("send message", "channel", channel, "id", id, "error", err)
		}
	}

	_, err = m.db.ExecContext(ctx,
		"INSERT INTO messages (id, channel, to_addr, subject, body, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, channel, toAddr, subject, body, msg.Status, now)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	return msg, nil
}

// handleTelegram handles POST /api/messaging/telegram — accepts {chat_id, text, parse_mode, token}
// and routes through the registered telegram channel provider.
func (m *Messaging) handleTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode,omitempty"`
		Token     string `json:"token,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.ChatID == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "chat_id is required"})
		return
	}
	if req.Text == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	msg, err := m.send(r.Context(), "telegram", req.ChatID, "", req.Text)
	if err != nil {
		m.logger.Error("telegram send", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "send failed"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, msg)
}

func (m *Messaging) handleMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return false
	}
	return true
}

func validateRequired(value, field string) string {
	if strings.TrimSpace(value) == "" {
		return field + " is required"
	}
	return ""
}

func collectErrors(errs ...string) string {
	var msgs []string
	for _, e := range errs {
		if e != "" {
			msgs = append(msgs, e)
		}
	}
	if len(msgs) > 0 {
		return strings.Join(msgs, "; ")
	}
	return ""
}
