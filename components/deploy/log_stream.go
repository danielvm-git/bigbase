package deploy

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// logHub is a per-deployment fan-out for WebSocket log subscribers.
// Subscribers receive all historical lines first, then live lines as they arrive.
type logHub struct {
	mu   sync.Mutex
	hist []string
	subs []chan string
	done bool
}

func (h *logHub) subscribe() (<-chan string, func()) {
	ch := make(chan string, 256)
	h.mu.Lock()
	// Deliver all historical lines to the new subscriber.
	for _, line := range h.hist {
		select {
		case ch <- line:
		default:
		}
	}
	if h.done {
		close(ch)
		h.mu.Unlock()
		return ch, func() {}
	}
	h.subs = append(h.subs, ch)
	h.mu.Unlock()

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		for i, s := range h.subs {
			if s == ch {
				h.subs = append(h.subs[:i], h.subs[i+1:]...)
				return
			}
		}
	}
	return ch, unsub
}

func (h *logHub) broadcast(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return
	}
	h.hist = append(h.hist, line)
	for _, ch := range h.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (h *logHub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return
	}
	h.done = true
	for _, ch := range h.subs {
		close(ch)
	}
	h.subs = nil
}

// getOrCreateHub returns the logHub for a deployment, creating it if needed.
func (d *Deploy) getOrCreateHub(id string) *logHub {
	d.logHubsMu.Lock()
	defer d.logHubsMu.Unlock()
	if h, ok := d.logHubs[id]; ok {
		return h
	}
	h := &logHub{}
	d.logHubs[id] = h
	return h
}

// closeLogStream signals all WebSocket subscribers that the deployment finished.
func (d *Deploy) closeLogStream(id string) {
	d.logHubsMu.RLock()
	h := d.logHubs[id]
	d.logHubsMu.RUnlock()
	if h != nil {
		h.close()
	}
}

// handleLogsStream upgrades to WebSocket and streams log lines for a deployment.
func (d *Deploy) handleLogsStream(w http.ResponseWriter, r *http.Request, id string) {
	// Verify the deployment exists.
	var exists int
	if err := d.db.QueryRowContext(r.Context(),
		"SELECT COUNT(*) FROM deployments WHERE id = ?", id).Scan(&exists); err != nil || exists == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	hub := d.getOrCreateHub(id)
	ch, unsub := hub.subscribe()
	defer unsub()

	for line := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			return
		}
	}
	_ = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
