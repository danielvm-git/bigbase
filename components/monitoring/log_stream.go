package monitoring

// e86s02: live log streaming over SSE. A dedicated per-entry stream (separate
// from the BusEvent visualizer stream) so the Logs tab can subscribe to new
// log rows as they are created, scoped to the caller's org.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/danielvm/bigbase/kernel"
)

// logSSEStream fans out LogEntry values to subscribed SSE clients.
type logSSEStream struct {
	mu      sync.RWMutex
	clients map[uint64]chan LogEntry
	nextID  uint64
}

func newLogSSEStream() *logSSEStream {
	return &logSSEStream{clients: make(map[uint64]chan LogEntry)}
}

func (s *logSSEStream) subscribe() (uint64, <-chan LogEntry) {
	ch := make(chan LogEntry, 64)
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.clients[id] = ch
	s.mu.Unlock()
	return id, ch
}

func (s *logSSEStream) unsubscribe(id uint64) {
	s.mu.Lock()
	if ch, ok := s.clients[id]; ok {
		close(ch)
		delete(s.clients, id)
	}
	s.mu.Unlock()
}

func (s *logSSEStream) broadcast(l LogEntry) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.clients {
		select {
		case ch <- l:
		default:
			// Drop when a slow client's buffer is full rather than block.
		}
	}
}

// broadcastLog publishes a new log entry to live SSE subscribers. No-op when
// the stream is not initialized (e.g. db-less test harnesses).
func (m *Monitoring) broadcastLog(l LogEntry) {
	if m.logStream != nil {
		m.logStream.broadcast(l)
	}
}

// handleLogStream serves GET /api/monitoring/logs/stream as SSE. It only
// forwards entries belonging to the caller's org (tenant isolation on the
// live path, mirroring handleLogSearch).
func (m *Monitoring) handleLogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
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
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	if m.logStream == nil {
		<-r.Context().Done()
		return
	}
	id, ch := m.logStream.subscribe()
	defer m.logStream.unsubscribe(id)

	for {
		select {
		case <-r.Context().Done():
			return
		case l, open := <-ch:
			if !open {
				return
			}
			if l.OrgID != orgID {
				continue // never leak another org's log on the live stream
			}
			data, err := json.Marshal(l)
			if err != nil {
				m.logger.Error("marshal log SSE", "error", err)
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}
