package deploy

import (
	"strings"
	"sync"
)

var maxDeployLogLines = 500
var maxDeployLogDeployments = 100

type logDeployments struct {
	mu     sync.Mutex
	buffer map[string][]string
	order  []string
}

func (l *logDeployments) init(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buffer == nil {
		l.buffer = make(map[string][]string)
	}
	for len(l.buffer) >= maxDeployLogDeployments {
		victim := l.order[0]
		if victim == id {
			break
		}
		delete(l.buffer, victim)
		l.order = l.order[1:]
	}
	l.buffer[id] = nil
	l.order = append(l.order, id)
}

func (l *logDeployments) append(id, line string) {
	l.mu.Lock()
	if l.buffer == nil {
		l.buffer = make(map[string][]string)
	}
	buf := append(l.buffer[id], line)
	if len(buf) > maxDeployLogLines {
		buf = buf[len(buf)-maxDeployLogLines:]
	}
	l.buffer[id] = buf
	l.mu.Unlock()
}

func (l *logDeployments) get(id string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buffer == nil {
		return nil
	}
	src := l.buffer[id]
	if len(src) == 0 {
		return []string{}
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func (l *logDeployments) remove(id string) {
	l.mu.Lock()
	delete(l.buffer, id)
	for i, oid := range l.order {
		if oid == id {
			l.order = append(l.order[:i], l.order[i+1:]...)
			break
		}
	}
	l.mu.Unlock()
}

func (d *Deploy) initDeployLogs(id string) {
	d.logs.init(id)
}

func (d *Deploy) appendDeployLog(id, line string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	d.logs.append(id, line)

	// Broadcast to WebSocket subscribers (non-blocking).
	d.logHubsMu.RLock()
	h := d.logHubs[id]
	d.logHubsMu.RUnlock()
	if h != nil {
		h.broadcast(line)
	}
}

func (d *Deploy) appendDeployLogBlock(id, text string) {
	for _, line := range strings.Split(text, "\n") {
		d.appendDeployLog(id, line)
	}
}

func (d *Deploy) getDeployLogs(id string) []string {
	return d.logs.get(id)
}

func (d *Deploy) deleteDeployLogs(id string) {
	d.logs.remove(id)
}
