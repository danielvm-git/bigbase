package deploy

import (
	"testing"
)

func TestStateMachineValidTransitions(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		from     DeploymentState
		expected []DeploymentState
	}{
		{StatePending, []DeploymentState{StateBuilding}},
		{StateBuilding, []DeploymentState{StateDeploying, StateRunning, StateFailed}},
		{StateDeploying, []DeploymentState{StateRunning, StateFailed}},
		{StateRunning, []DeploymentState{StateFailed, StateRolledBack, StateDraining}},
		{StateDraining, []DeploymentState{StateStopped, StateFailed, StateRunning}},
		{StateStopped, []DeploymentState{StateRunning}},
		{StateFailed, nil},
		{StateRolledBack, nil},
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
	sm := NewStateMachine()

	valid := []struct{ from, to string }{
		{"pending", "building"},
		{"building", "deploying"},
		{"building", "running"},
		{"building", "failed"},
		{"deploying", "running"},
		{"deploying", "failed"},
		{"running", "failed"},
		{"running", "rolled_back"},
		{"running", "draining"},
		{"draining", "stopped"},
		{"draining", "failed"},
		{"draining", "running"},
		{"stopped", "running"},
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
		{"rolled_back", "running"},
		{"rolled_back", "failed"},
		{"rolled_back", "pending"},
		{"running", "stopped"},
		{"pending", "draining"},
		{"draining", "building"},
		{"stopped", "building"},
		{"stopped", "failed"},
	}
	for _, v := range invalid {
		if sm.CanTransition(v.from, v.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", v.from, v.to)
		}
	}
}

func TestStateMachineIsValidState(t *testing.T) {
	sm := NewStateMachine()

	valid := []string{"pending", "building", "deploying", "running", "failed", "rolled_back", "draining", "stopped"}
	for _, s := range valid {
		if !sm.IsValidState(s) {
			t.Errorf("IsValidState(%s) = false, want true", s)
		}
	}

	invalid := []string{"live", "replaced", "", "unknown", "drain"}
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

func TestStateMachineIsActive(t *testing.T) {
	sm := NewStateMachine()

	// Active = live or in-flight. A site delete must wait for these.
	active := []string{"pending", "building", "deploying", "draining", "running"}
	for _, s := range active {
		if !sm.IsActive(s) {
			t.Errorf("IsActive(%s) = false, want true", s)
		}
	}

	for _, s := range []string{"stopped", "failed", "rolled_back", "", "unknown", "replaced"} {
		if sm.IsActive(s) {
			t.Errorf("IsActive(%s) = true, want false", s)
		}
	}
}

func TestStateMachineIsTerminal(t *testing.T) {
	sm := NewStateMachine()

	// Terminal = the build/start pipeline has finished. No automatic build
	// transition is expected, though rollback may revive a stopped deployment.
	for _, s := range []string{"running", "stopped", "failed", "rolled_back"} {
		if !sm.IsTerminal(s) {
			t.Errorf("IsTerminal(%s) = false, want true", s)
		}
	}

	for _, s := range []string{"pending", "building", "deploying", "draining", "", "unknown"} {
		if sm.IsTerminal(s) {
			t.Errorf("IsTerminal(%s) = true, want false", s)
		}
	}
}

func TestStateMachineCanDelete(t *testing.T) {
	sm := NewStateMachine()

	// In-flight deployments cannot be deleted out from under the engine.
	for _, s := range []string{"pending", "building", "deploying", "draining"} {
		if sm.CanDelete(s) {
			t.Errorf("CanDelete(%s) = true, want false", s)
		}
	}

	// Running is deletable because the delete path stops it first; finished
	// states are always deletable; unknown legacy rows stay deletable.
	for _, s := range []string{"running", "stopped", "failed", "rolled_back", "", "replaced"} {
		if !sm.CanDelete(s) {
			t.Errorf("CanDelete(%s) = false, want true", s)
		}
	}
}

func TestStateMachineStoppedIsRevivable(t *testing.T) {
	sm := NewStateMachine()

	// e45 declared stopped terminal, but rollback (TestRollbackStaticSite)
	// restores a drained deployment, so stopped → running is intentional.
	if !sm.CanTransition("stopped", "running") {
		t.Error("CanTransition(stopped, running) = false, want true (rollback relies on it)")
	}
	if !sm.IsTerminal("stopped") {
		t.Error("IsTerminal(stopped) = false, want true (build pipeline is complete)")
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
	if StateRolledBack != "rolled_back" {
		t.Errorf("StateRolledBack = %s, want rolled_back", StateRolledBack)
	}
}
