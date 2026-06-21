package deploy

import "strings"

const maxDeployLogLines = 500
const maxDeployLogDeployments = 100

func (d *Deploy) initDeployLogs(id string) {
	d.deployLogsMu.Lock()
	defer d.deployLogsMu.Unlock()
	if d.deployLogs == nil {
		d.deployLogs = make(map[string][]string)
	}
	for len(d.deployLogs) >= maxDeployLogDeployments {
		var victim string
		for k := range d.deployLogs {
			if k != id {
				victim = k
				break
			}
		}
		if victim == "" {
			break
		}
		d.deleteDeployLogs(victim)
	}
	d.deployLogs[id] = nil
}

func (d *Deploy) appendDeployLog(id, line string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	d.deployLogsMu.Lock()
	if d.deployLogs == nil {
		d.deployLogs = make(map[string][]string)
	}
	buf := append(d.deployLogs[id], line)
	if len(buf) > maxDeployLogLines {
		buf = buf[len(buf)-maxDeployLogLines:]
	}
	d.deployLogs[id] = buf
	d.deployLogsMu.Unlock()

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
	d.deployLogsMu.RLock()
	defer d.deployLogsMu.RUnlock()
	if d.deployLogs == nil {
		return nil
	}
	src := d.deployLogs[id]
	if len(src) == 0 {
		return []string{}
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func (d *Deploy) deleteDeployLogs(id string) {
	d.deployLogsMu.Lock()
	delete(d.deployLogs, id)
	d.deployLogsMu.Unlock()
}
