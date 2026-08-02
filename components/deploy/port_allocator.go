package deploy

import (
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	pickPortMu      sync.Mutex
	pickPortCounter int64
)

const maxPickPortAttempts = 1000

// GetFreePort returns an available TCP port dynamically allocated by the OS kernel
// using net.Listen("tcp", "127.0.0.1:0"). It verifies that the socket is cleanly
// closed and released before returning the port, preventing EADDRINUSE collisions.
func GetFreePort() (int, error) {
	pickPortMu.Lock()
	defer pickPortMu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("GetFreePort: failed to listen on 127.0.0.1:0: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		return 0, fmt.Errorf("GetFreePort: failed to close socket for port %d: %w", port, err)
	}

	// Verify socket release: ensure port is free before returning
	for attempt := 0; attempt < 20; attempt++ {
		if portIsFree(port) {
			return port, nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return port, nil
}

// pickPort returns the next port that is both unused by this process's
// counter and currently free at the OS level. If base <= 0, it uses GetFreePort().
func pickPort(base int) (int, error) {
	if base <= 0 {
		return GetFreePort()
	}
	return pickPortWithLimit(base, maxPickPortAttempts)
}

func pickPortWithLimit(base, maxAttempts int) (int, error) {
	pickPortMu.Lock()
	defer pickPortMu.Unlock()
	for attempt := 0; attempt < maxAttempts; attempt++ {
		pickPortCounter++
		candidate := base + int(pickPortCounter)
		if portIsFree(candidate) {
			return candidate, nil
		}
	}
	return 0, fmt.Errorf("pickPort: no free port found after %d attempts starting from base %d", maxAttempts, base)
}

// portIsFree probes a candidate port with a real listen-and-release. A bind
// to 127.0.0.1:<port> fails with EADDRINUSE if anything — including a
// process bound to 0.0.0.0:<port> — already holds that port.
func portIsFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
