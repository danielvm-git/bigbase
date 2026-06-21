package deploy

import (
	"testing"
)

func TestStateMachineValidTransitions(t *testing.T) {
	sm := newStateMachine()

	tests := []struct {
		from     DeploymentState
		expected []DeploymentState
	}{
		{StatePending, []DeploymentState{StateBuilding}},
		{StateBuilding, []DeploymentState{StateDeploying, StateRunning, StateFailed}},
		{StateDeploying, []DeploymentState{StateRunning, StateFailed}},
		{StateRunning, []DeploymentState{StateFailed}},
		{StateFailed, nil},
	}

	for _, tt := range tests {
		got := sm.ValidTransitions(string(tt.from))
		if len(got) != len(tt.expected) {
			t.Errorf("ValidTransitions(%s) = %v, want %v", tt.from, got, tt.expected)
			continue
		}
		for i, v := range got {
			if v != string(tt.expected[i]) {
				t.Errorf("ValidTransitions(%s)[%d] = %s, want %s", tt.from, i, v, tt.expected[i])
			}
		}
	}
}

func TestStateMachineCanTransition(t *testing.T) {
	sm := newStateMachine()

	valid := []struct{ from, to string }{
		{"pending", "building"},
		{"building", "deploying"},
		{"building", "running"},
		{"building", "failed"},
		{"deploying", "running"},
		{"deploying", "failed"},
		{"running", "failed"},
	}
	for _, v := range valid {
		if !sm.CanTransition(v.from, v.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", v.from, v.to)
		}
	}

	invalid := []struct{ from, to string }{
		{"pending", "running"},
		{"pending", "deploying"},
		{"pending", "failed"},
		{"running", "building"},
		{"running", "deploying"},
		{"running", "pending"},
		{"failed", "building"},
		{"failed", "running"},
		{"failed", "pending"},
	}
	for _, v := range invalid {
		if sm.CanTransition(v.from, v.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", v.from, v.to)
		}
	}
}

func TestStateMachineIsValidState(t *testing.T) {
	sm := newStateMachine()

	valid := []string{"pending", "building", "deploying", "running", "failed"}
	for _, s := range valid {
		if !sm.IsValidState(s) {
			t.Errorf("IsValidState(%s) = false, want true", s)
		}
	}

	invalid := []string{"live", "replaced", "", "unknown"}
	for _, s := range invalid {
		if sm.IsValidState(s) {
			t.Errorf("IsValidState(%s) = true, want false", s)
		}
	}
}

func TestStatusTransitionStruct(t *testing.T) {
	tr := StatusTransition{
		From:      "pending",
		To:        "building",
		Timestamp: "2026-01-01T00:00:00Z",
	}
	if tr.From != "pending" || tr.To != "building" {
		t.Errorf("StatusTransition fields incorrect: %+v", tr)
	}
}

func TestStateConstants(t *testing.T) {
	if StatePending != "pending" {
		t.Errorf("StatePending = %s, want pending", StatePending)
	}
	if StateBuilding != "building" {
		t.Errorf("StateBuilding = %s, want building", StateBuilding)
	}
	if StateDeploying != "deploying" {
		t.Errorf("StateDeploying = %s, want deploying", StateDeploying)
	}
	if StateRunning != "running" {
		t.Errorf("StateRunning = %s, want running", StateRunning)
	}
	if StateFailed != "failed" {
		t.Errorf("StateFailed = %s, want failed", StateFailed)
	}
}
