package monitoring_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func TestSSEEventsEndpoint(t *testing.T) {
	_, handler := setupMonitoring(t)

	t.Run("rejects non-GET", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/monitoring/events", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 got %d", w.Code)
		}
	})

	t.Run("streams SSE keepalive comment", func(t *testing.T) {
		// We use a cancellable request to avoid blocking forever.
		ctx := t.Context()
		req := httptest.NewRequest("GET", "/api/monitoring/events", nil).WithContext(ctx)

		// httptest.ResponseRecorder does not implement http.Flusher, so we
		// use a real httptest.Server instead.
		srv := httptest.NewServer(handler)
		defer srv.Close()

		respCh := make(chan string, 1)
		go func() {
			resp, err := srv.Client().Get(srv.URL + "/api/monitoring/events")
			if err != nil {
				respCh <- ""
				return
			}
			defer func() { _ = resp.Body.Close() }()
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					respCh <- line
					return
				}
			}
			respCh <- ""
		}()

		select {
		case line := <-respCh:
			if !strings.HasPrefix(line, ":") {
				t.Errorf("expected SSE comment line, got %q", line)
			}
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for SSE keepalive")
		}

		_ = req // suppress unused warning
	})
}

func TestSSEEventsBroadcast(t *testing.T) {
	m, handler := setupMonitoring(t)

	// Wire the event bus manually.
	bus := kernel.NewEventBus()
	m.WithEventBus(bus, []string{"test_hook"})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Connect an SSE client.
	lineCh := make(chan string, 4)
	go func() {
		resp, err := srv.Client().Get(srv.URL + "/api/monitoring/events")
		if err != nil {
			return
		}
		defer func() { _ = resp.Body.Close() }()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				lineCh <- line
				return
			}
		}
	}()

	// Give the client time to connect.
	time.Sleep(50 * time.Millisecond)

	// Emit an event on the bus.
	if err := bus.Emit(kernel.Event{Name: "test_hook", Data: map[string]any{"key": "val"}}, nil); err != nil {
		t.Fatalf("emit: %v", err)
	}

	select {
	case line := <-lineCh:
		if !strings.Contains(line, "test_hook") {
			t.Errorf("expected hook name in SSE data, got %q", line)
		}
	case <-time.After(3 * time.Second):
		t.Error("timeout waiting for SSE event")
	}
}
