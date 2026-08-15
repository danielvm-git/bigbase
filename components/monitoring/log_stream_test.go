package monitoring

// e86s02 coverage: creating a log broadcasts it to live subscribers, and the
// SSE handler only forwards entries for the caller's org.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func buildStreamMonitoring(t *testing.T) *Monitoring {
	t.Helper()
	logger := kernel.NoopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	m := New(Options{DB: d, Logger: logger})
	k.Register(d)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return m
}

// syncWriter is a race-safe http.ResponseWriter+Flusher that records the body
// and signals on every write so tests can wait for streamed data.
type syncWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	header http.Header
	writes chan struct{}
}

func newSyncWriter() *syncWriter {
	return &syncWriter{header: http.Header{}, writes: make(chan struct{}, 64)}
}
func (s *syncWriter) Header() http.Header { return s.header }
func (s *syncWriter) WriteHeader(int)     {}
func (s *syncWriter) Flush()              {}
func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	n, err := s.buf.Write(p)
	s.mu.Unlock()
	select {
	case s.writes <- struct{}{}:
	default:
	}
	return n, err
}
func (s *syncWriter) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestLogStream(t *testing.T) {
	m := buildStreamMonitoring(t)

	// Subscribe deterministically before creating a log.
	id, ch := m.logStream.subscribe()
	defer m.logStream.unsubscribe(id)

	req := httptest.NewRequest("POST", "/api/monitoring/logs",
		strings.NewReader(`{"level":"info","message":"streamed-hello"}`)).
		WithContext(kernel.WithOrgID(context.Background(), 3))
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create log: got %d, want 201: %s", w.Code, w.Body.String())
	}

	select {
	case l := <-ch:
		if l.Message != "streamed-hello" || l.OrgID != 3 {
			t.Errorf("broadcast log = %+v, want message=streamed-hello org=3", l)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no log broadcast received within 2s")
	}
}

func TestLogStreamOrgFilter(t *testing.T) {
	m := buildStreamMonitoring(t)

	sw := newSyncWriter()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest("GET", "/api/monitoring/logs/stream", nil).
		WithContext(kernel.WithOrgID(ctx, 3))

	done := make(chan struct{})
	go func() { m.handleLogStream(sw, req); close(done) }()

	// Broadcast a foreign-org and own-org entry each round until the own-org
	// message is written — guaranteeing the subscription is live and that the
	// foreign-org entry was delivered-and-skipped in the same round.
	deadline := time.Now().Add(3 * time.Second)
	for !strings.Contains(sw.String(), "my-org-msg") {
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("own-org message never streamed")
		}
		m.logStream.broadcast(LogEntry{ID: "x", OrgID: 9, Level: "info", Message: "other-org-msg"})
		m.logStream.broadcast(LogEntry{ID: "y", OrgID: 3, Level: "info", Message: "my-org-msg"})
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	body := sw.String()
	if strings.Contains(body, "other-org-msg") {
		t.Error("SSE stream leaked another org's log entry")
	}
	if !strings.Contains(body, "my-org-msg") {
		t.Error("SSE stream missing the caller's own log entry")
	}
}
