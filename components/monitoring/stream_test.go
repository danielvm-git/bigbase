package monitoring_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestMetricsStream verifies the SSE endpoint emits a well-formed data: {json} event.
func TestMetricsStream(t *testing.T) {
	_, handler := setupMonitoring(t)

	// Use a context with a 3-second timeout so the stream request terminates.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/monitoring/metrics/stream", nil).WithContext(ctx)

	// Use a real httptest server so we can read from the response body in a streaming fashion.
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := &http.Client{}
	resp, err := client.Get(srv.URL + "/api/monitoring/metrics/stream")
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close()

	t.Run("content-type is text/event-stream", func(t *testing.T) {
		ct := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "text/event-stream") {
			t.Fatalf("expected text/event-stream, got %q", ct)
		}
	})

	// Read first SSE event line within the timeout.
	done := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				done <- line
				return
			}
		}
		errCh <- scanner.Err()
	}()

	select {
	case line := <-done:
		t.Run("data line has json payload", func(t *testing.T) {
			payload := strings.TrimPrefix(line, "data: ")
			var m map[string]json.RawMessage
			if err := json.Unmarshal([]byte(payload), &m); err != nil {
				t.Fatalf("unmarshal SSE payload: %v (line: %s)", err, line)
			}
			if _, ok := m["host"]; !ok {
				t.Fatalf("expected 'host' in SSE payload, got keys: %v", sseKeys(m))
			}
			if _, ok := m["system"]; !ok {
				t.Fatalf("expected 'system' in SSE payload, got keys: %v", sseKeys(m))
			}
		})
	case err := <-errCh:
		t.Fatalf("scanner error: %v", err)
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for first SSE event")
	}

	// unused — suppress test-only reference
	_ = req
}

func sseKeys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
