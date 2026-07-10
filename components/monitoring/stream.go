package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const sseInterval = 5 * time.Second

// sseEvent is the payload emitted on the SSE stream.
type sseEvent struct {
	System SystemMetrics `json:"system"`
	Host   HostMetrics   `json:"host"`
	SentAt string        `json:"sent_at"`
}

// handleMetricsStream serves a Server-Sent Events stream of host metrics, emitting
// one event every sseInterval. Clients receive data immediately on connect, then
// every 5 seconds until they disconnect.
func (m *Monitoring) handleMetricsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	emit := func() bool {
		evt := sseEvent{
			System: m.SystemMetrics(),
			Host:   m.LatestHostMetrics(),
			SentAt: time.Now().UTC().Format(time.RFC3339),
		}
		data, err := json.Marshal(evt)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !emit() {
		return
	}

	ticker := time.NewTicker(sseInterval)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !emit() {
				return
			}
		}
	}
}
