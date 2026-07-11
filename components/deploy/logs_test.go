package deploy

import (
	"testing"
)

func TestDeployLogFIFOEviction(t *testing.T) {
	// Create a logDeployments with capacity 3.
	// We override the constants for this test only.
	origDeployments := maxDeployLogDeployments
	origLines := maxDeployLogLines
	maxDeployLogDeployments = 3
	maxDeployLogLines = 10
	defer func() {
		maxDeployLogDeployments = origDeployments
		maxDeployLogLines = origLines
	}()

	logs := &logDeployments{}

	// Push A, B, C (fill to capacity)
	logs.init("A")
	logs.append("A", "log line 1 for A")
	logs.init("B")
	logs.append("B", "log line 1 for B")
	logs.init("C")
	logs.append("C", "log line 1 for C")

	// Verify A, B, C all exist
	if got := logs.get("A"); len(got) == 0 {
		t.Fatal("A should exist before eviction")
	}
	if got := logs.get("B"); len(got) == 0 {
		t.Fatal("B should exist before eviction")
	}
	if got := logs.get("C"); len(got) == 0 {
		t.Fatal("C should exist before eviction")
	}

	// Push D — should evict A (oldest)
	logs.init("D")
	logs.append("D", "log line 1 for D")

	if got := logs.get("A"); len(got) != 0 {
		t.Fatalf("A should be evicted (oldest), but still has %d log lines", len(got))
	}
	if got := logs.get("B"); len(got) == 0 {
		t.Fatal("B should still exist after D eviction")
	}
	if got := logs.get("C"); len(got) == 0 {
		t.Fatal("C should still exist after D eviction")
	}
	if got := logs.get("D"); len(got) == 0 {
		t.Fatal("D should exist after insertion")
	}

	// Push E — should evict B (now oldest)
	logs.init("E")
	logs.append("E", "log line 1 for E")

	if got := logs.get("B"); len(got) != 0 {
		t.Fatalf("B should be evicted (now oldest), but still has %d log lines", len(got))
	}
	if got := logs.get("C"); len(got) == 0 {
		t.Fatal("C should still exist after E eviction")
	}
	if got := logs.get("D"); len(got) == 0 {
		t.Fatal("D should still exist after E eviction")
	}
	if got := logs.get("E"); len(got) == 0 {
		t.Fatal("E should exist after insertion")
	}

	// Verify deterministic order: order should be [C, D, E]
	if len(logs.order) != 3 {
		t.Fatalf("expected 3 entries in order, got %d: %v", len(logs.order), logs.order)
	}
	if logs.order[0] != "C" || logs.order[1] != "D" || logs.order[2] != "E" {
		t.Fatalf("expected order [C D E], got %v", logs.order)
	}
}
