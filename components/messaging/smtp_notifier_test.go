package messaging_test

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/messaging"
)

// smtpTestServer is a minimal SMTP server that captures one transaction's
// envelope + DATA payload so we can assert the notifier actually sent mail.
type smtpTestServer struct {
	ln      net.Listener
	to      []string
	from    string
	data    string
	readyCh chan struct{}
	mu      sync.Mutex
}

func newSMTPTestServer(t *testing.T) *smtpTestServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &smtpTestServer{ln: ln, readyCh: make(chan struct{})}
	go s.serve()
	<-s.readyCh
	return s
}

func (s *smtpTestServer) addr() string { return s.ln.Addr().String() }

func (s *smtpTestServer) serve() {
	close(s.readyCh)
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *smtpTestServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	write := func(line string) { _, _ = w.WriteString(line + "\r\n"); _ = w.Flush() }
	write("220 test.smtp ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		upper := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			write("250-test.smtp")
			write("250 OK")
		case strings.HasPrefix(upper, "MAIL FROM"):
			s.mu.Lock()
			s.from = strings.TrimSpace(line)
			s.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			s.mu.Lock()
			s.to = append(s.to, strings.TrimSpace(line))
			s.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(upper, "DATA"):
			write("354 End data with <CR><LF>.<CR><LF>")
			var sb strings.Builder
			for {
				dl, derr := r.ReadString('\n')
				if derr != nil {
					return
				}
				if strings.TrimSpace(dl) == "." {
					break
				}
				sb.WriteString(dl)
			}
			s.mu.Lock()
			s.data = sb.String()
			s.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(upper, "QUIT"):
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

// TestSMTPAlertNotifierSendsEmail verifies that the SMTP-backed alert notifier
// composes and sends an email when an alert.triggered event is delivered.
// (Issue #178: wire alert.triggered → components/messaging SMTP delivery.)
func TestSMTPAlertNotifierSendsEmail(t *testing.T) {
	srv := newSMTPTestServer(t)
	t.Cleanup(func() { _ = srv.ln.Close() })

	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	m := messaging.New(messaging.Options{DB: d, Logger: testLogger{}, SMTPHost: "127.0.0.1"})

	// Split host:port from the listener address.
	host, port, _ := net.SplitHostPort(srv.addr())
	notifier := messaging.NewSMTPAlertNotifier(messaging.SMTPAlertNotifierOptions{
		Host:     host,
		Port:     port,
		From:     "alerts@bigbase.local",
		To:       []string{"oncall@bigbase.local"},
		AlertURL: "https://app.bigbase.local/monitoring/incidents",
	})

	err := notifier.NotifyAlert(context.Background(), messaging.AlertEvent{
		AlertID:    "rule-7",
		IncidentID: "inc-7",
		Name:       "site down",
		Metric:     "site_up",
		Value:      0,
		Threshold:  1,
		Operator:   "lt",
	})
	if err != nil {
		t.Fatalf("NotifyAlert: %v", err)
	}

	// The notifier may send asynchronously; poll for the DATA payload.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		srv.mu.Lock()
		if srv.data != "" {
			srv.mu.Unlock()
			break
		}
		srv.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}

	srv.mu.Lock()
	data := srv.data
	toAddrs := append([]string{}, srv.to...)
	srv.mu.Unlock()

	if data == "" {
		t.Fatal("expected SMTP server to receive a DATA payload, got none")
	}
	if !strings.Contains(data, "site down") {
		t.Errorf("email body should mention alert name; got:\n%s", data)
	}
	if !strings.Contains(data, "rule-7") || !strings.Contains(data, "inc-7") {
		t.Errorf("email body should reference alert/incident ids; got:\n%s", data)
	}
	var matchedRecipient bool
	for _, a := range toAddrs {
		if strings.Contains(a, "oncall@bigbase.local") {
			matchedRecipient = true
		}
	}
	if !matchedRecipient {
		t.Errorf("expected RCPT TO oncall@bigbase.local; got %v", toAddrs)
	}
}
