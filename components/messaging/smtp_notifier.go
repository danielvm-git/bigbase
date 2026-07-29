package messaging

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// AlertEvent mirrors monitoring.AlertEvent so components/messaging can deliver
// alert notifications without an import cycle on components/monitoring.
// (Issue #178: wire alert.triggered → components/messaging SMTP delivery.)
type AlertEvent struct {
	AlertID    string  `json:"alert_id"`
	IncidentID string  `json:"incident_id"`
	Name       string  `json:"name"`
	Metric     string  `json:"metric"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Operator   string  `json:"operator"`
}

// SMTPAlertNotifierOptions configures the SMTP-backed alert notifier.
type SMTPAlertNotifierOptions struct {
	// Host is the SMTP server hostname.
	Host string
	// Port is the SMTP server port (e.g. 587, 2525, 25).
	Port string
	// Username and Password enable SMTP AUTH (PLAIN). Leave empty for no auth.
	Username string
	Password string
	// From is the envelope + header sender address.
	From string
	// To is the list of recipient addresses. If empty, NotifyAlert is a no-op
	// (and returns nil) so a misconfigured notifier never blocks alerts.
	To []string
	// AlertURL is an optional base URL used to deep-link to the incident in the
	// email body (e.g. "https://app.bigbase.local/monitoring/incidents").
	AlertURL string
}

// SMTPAlertNotifier delivers alert.triggered events as email via net/smtp.
// It implements the monitoring.AlertNotifier contract (NotifyAlert).
type SMTPAlertNotifier struct {
	opts    SMTPAlertNotifierOptions
	auth    smtp.Auth
	address string
	send    func(ctx context.Context, addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// NewSMTPAlertNotifier builds an SMTP-backed alert notifier.
func NewSMTPAlertNotifier(opts SMTPAlertNotifierOptions) *SMTPAlertNotifier {
	n := &SMTPAlertNotifier{
		opts:    opts,
		address: opts.Host + ":" + opts.Port,
		// send defaults to net/smtp.SendMail but is overridable for tests.
		send: func(_ context.Context, addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			return smtp.SendMail(addr, a, from, to, msg)
		},
	}
	if opts.Username != "" && opts.Password != "" {
		n.auth = smtp.PlainAuth("", opts.Username, opts.Password, opts.Host)
	}
	return n
}

// NotifyAlert formats the alert as an email and sends it via SMTP. It is a
// no-op (returns nil) when no recipients are configured.
func (n *SMTPAlertNotifier) NotifyAlert(ctx context.Context, ev AlertEvent) error {
	if len(n.opts.To) == 0 {
		return nil
	}
	msg := buildAlertEmail(n.opts.From, n.opts.To, ev, n.opts.AlertURL)

	// Honor context cancellation/deadline on a best-effort basis. smtp.SendMail
	// does not accept a context, so we race it against ctx.Done() with a guard
	// goroutine. A short hard cap guards against a hung SMTP server.
	done := make(chan error, 1)
	go func() {
		done <- n.send(ctx, n.address, n.auth, n.opts.From, n.opts.To, msg)
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("smtp alert notify: %w", ctx.Err())
	case <-time.After(30 * time.Second):
		return fmt.Errorf("smtp alert notify: timed out after 30s")
	}
}

// buildAlertEmail renders a plain-text email body for an alert event.
func buildAlertEmail(from string, to []string, ev AlertEvent, alertURL string) []byte {
	subject := fmt.Sprintf("[BigBase Alert] %s", fallback(ev.Name, "untitled alert"))

	var body strings.Builder
	fmt.Fprintf(&body, "Alert:      %s\n", ev.Name)
	fmt.Fprintf(&body, "Metric:     %s\n", ev.Metric)
	fmt.Fprintf(&body, "Condition:  %s %.0f (current value: %.0f)\n", ev.Operator, ev.Threshold, ev.Value)
	if ev.AlertID != "" {
		fmt.Fprintf(&body, "Alert ID:   %s\n", ev.AlertID)
	}
	if ev.IncidentID != "" {
		fmt.Fprintf(&body, "Incident:   %s\n", ev.IncidentID)
	}
	if alertURL != "" && ev.IncidentID != "" {
		fmt.Fprintf(&body, "URL:        %s/%s\n", strings.TrimRight(alertURL, "/"), ev.IncidentID)
	}
	body.WriteString("\n-- BigBase monitoring\n")

	var hdr strings.Builder
	fmt.Fprintf(&hdr, "From: %s\r\n", from)
	fmt.Fprintf(&hdr, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&hdr, "Subject: %s\r\n", subject)
	hdr.WriteString("MIME-Version: 1.0\r\n")
	hdr.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	hdr.WriteString("\r\n")
	hdr.WriteString(body.String())
	return []byte(hdr.String())
}

func fallback(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
