package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// BusEvent holds a single captured event bus emission for SSE streaming.
type BusEvent struct {
	Hook      string         `json:"hook"`
	Data      map[string]any `json:"data"`
	Timestamp string         `json:"timestamp"`
}

// eventStream manages SSE clients for the event bus visualizer.
type eventStream struct {
	mu      sync.RWMutex
	clients map[uint64]chan BusEvent
	nextID  uint64
}

func newEventStream() *eventStream {
	return &eventStream{
		clients: make(map[uint64]chan BusEvent),
	}
}

func (es *eventStream) subscribe() (uint64, <-chan BusEvent) {
	ch := make(chan BusEvent, 64)
	es.mu.Lock()
	es.nextID++
	id := es.nextID
	es.clients[id] = ch
	es.mu.Unlock()
	return id, ch
}

func (es *eventStream) unsubscribe(id uint64) {
	es.mu.Lock()
	if ch, ok := es.clients[id]; ok {
		close(ch)
		delete(es.clients, id)
	}
	es.mu.Unlock()
}

func (es *eventStream) broadcast(ev BusEvent) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	for _, ch := range es.clients {
		select {
		case ch <- ev:
		default:
			// Drop events when client channel is full to avoid blocking.
		}
	}
}

// WithEventBus wires the monitoring component to the event bus so it can
// capture all emissions and relay them via SSE. This is called from Start.
func (m *Monitoring) WithEventBus(bus *kernel.EventBus, hookNames []string) {
	if bus == nil {
		return
	}
	if m.stream == nil {
		m.stream = newEventStream()
	}
	for _, name := range hookNames {
		hookName := name // capture loop var
		bus.Subscribe(kernel.HookDef{
			Name:     hookName,
			Priority: 1000,
			Handler: func(_ *kernel.Context, ev kernel.Event) error {
				siteID := eventSiteID(ev.Data)
				if m.recorder != nil {
					_ = m.recorder.Record(context.Background(), hookName, siteID, ev.Data)
				}
				m.stream.broadcast(BusEvent{
					Hook:      hookName,
					Data:      ev.Data,
					Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				})
				return nil
			},
		})
	}
}

// handleSSEEvents serves GET /api/monitoring/events as SSE.
func (m *Monitoring) handleSSEEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

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

	// Send a keepalive comment immediately so the client knows the stream is live.
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	if m.stream == nil {
		// No bus wired; just keep connection open until client disconnects.
		<-r.Context().Done()
		return
	}

	id, ch := m.stream.subscribe()
	defer m.stream.unsubscribe(id)

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				m.logger.Error("marshal SSE event", "error", err)
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}
