package deploy_test

import (
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
)

func TestStateMachine_CanTransition_ValidTransitions(t *testing.T) {
	sm := deploy.NewStateMachine()

	tests := []struct {
		from string
		to   string
		want bool
	}{
		// Valid transitions
		{"pending", "building", true},
		{"pending", "failed", true},
		{"building", "deploying", true},
		{"building", "running", true},
		{"building", "failed", true},
		{"deploying", "live", true},
		{"deploying", "running", true},
		{"deploying", "failed", true},
		{"live", "failed", true},
		{"running", "failed", true},

		// Invalid transitions
		{"pending", "live", false},        // skip deploying
		{"pending", "deploying", false},    // skip building
		{"building", "live", false},        // skip deploying
		{"pending", "pending", false},      // no self-transition
		{"live", "building", false},        // can't go back
		{"live", "deploying", false},       // can't go back
		{"live", "live", false},            // no self-transition
		{"failed", "building", false},      // terminal
		{"failed", "live", false},          // terminal
		{"failed", "failed", false},        // terminal (no self)
		{"", "building", false},            // empty from
		{"pending", "", false},             // empty to
		{"running", "building", false},      // can't go back
		{"running", "running", false},       // no self-transition
	}

	for _, tt := range tests {
		got := sm.CanTransition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestStateMachine_ValidTransitions(t *testing.T) {
	sm := deploy.NewStateMachine()

	tests := []struct {
		state string
		want  []string
	}{
		{"pending", []string{"building", "failed"}},
		{"building", []string{"deploying", "running", "failed"}},
		{"deploying", []string{"live", "running", "failed"}},
		{"live", []string{"failed"}},
		{"running", []string{"failed"}},
		{"failed", []string{}},
		{"unknown", []string{}},
		{"", []string{}},
	}

	for _, tt := range tests {
		got := sm.ValidTransitions(tt.state)
		if !stringSliceEqual(got, tt.want) {
			t.Errorf("ValidTransitions(%q) = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestStateMachine_Constants(t *testing.T) {
	if got := deploy.StatePending; got != "pending" {
		t.Errorf("StatePending = %q, want 'pending'", got)
	}
	if got := deploy.StateBuilding; got != "building" {
		t.Errorf("StateBuilding = %q, want 'building'", got)
	}
	if got := deploy.StateDeploying; got != "deploying" {
		t.Errorf("StateDeploying = %q, want 'deploying'", got)
	}
	if got := deploy.StateLive; got != "live" {
		t.Errorf("StateLive = %q, want 'live'", got)
	}
	if got := deploy.StateFailed; got != "failed" {
		t.Errorf("StateFailed = %q, want 'failed'", got)
	}
}

func TestStatusTransition_Struct(t *testing.T) {
	tr := deploy.StatusTransition{
		From:      "pending",
		To:        "building",
		Timestamp: "2026-06-21T00:00:00Z",
	}
	if tr.From != "pending" {
		t.Errorf("StatusTransition.From = %q, want 'pending'", tr.From)
	}
	if tr.To != "building" {
		t.Errorf("StatusTransition.To = %q, want 'building'", tr.To)
	}
	if tr.Timestamp != "2026-06-21T00:00:00Z" {
		t.Errorf("StatusTransition.Timestamp = %q, want '2026-06-21T00:00:00Z'", tr.Timestamp)
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
