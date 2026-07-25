// story: BUG-2026-07-25-port-allocator-no-liveness-check
package deploy

import (
	"fmt"
	"net"
	"testing"
)

// resetPickPortCounter isolates each test from pickPortCounter's shared,
// package-level state (the exact stateful-in-memory-counter design that
// causes the production bug this file regression-tests).
func resetPickPortCounter(t *testing.T) {
	t.Helper()
	pickPortMu.Lock()
	pickPortCounter = 0
	pickPortMu.Unlock()
}

func TestPickPort_SkipsOccupiedPort(t *testing.T) {
	resetPickPortCounter(t)

	const base = 20000
	// Occupy exactly the port pickPort's counter would hand out first
	// (base+1), simulating an orphaned process still bound to it after a
	// BigBase restart reset the in-memory counter.
	occupied := base + 1
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", occupied))
	if err != nil {
		t.Fatalf("failed to occupy test port %d: %v", occupied, err)
	}
	defer func() { _ = ln.Close() }()

	got, err := pickPort(base)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got == occupied {
		t.Fatalf("pickPort returned the occupied port %d — collision not detected", occupied)
	}

	// The returned port must actually be free (prove it, don't just trust it).
	ln2, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", got))
	if err != nil {
		t.Fatalf("pickPort returned port %d but it is not actually free: %v", got, err)
	}
	_ = ln2.Close()
}

func TestPickPort_ReturnsErrorWhenExhausted(t *testing.T) {
	resetPickPortCounter(t)

	const base = 21000
	const span = 5

	var listeners []net.Listener
	defer func() {
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}()
	for i := 1; i <= span; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base+i))
		if err != nil {
			t.Fatalf("failed to occupy test port %d: %v", base+i, err)
		}
		listeners = append(listeners, ln)
	}

	_, err := pickPortWithLimit(base, span)
	if err == nil {
		t.Fatal("expected error when every candidate port in range is occupied, got nil")
	}
}

func TestPickPort_NoCollisionWhenNothingOccupied(t *testing.T) {
	resetPickPortCounter(t)

	got, err := pickPort(22000)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != 22001 {
		t.Fatalf("expected first candidate 22001 when nothing is occupied, got %d", got)
	}
}
