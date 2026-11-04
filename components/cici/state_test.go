package cici_test

import (
	"errors"
	"testing"

	"github.com/danielvm/bigbase/components/cici"
)

func TestStateMachineValidTransitions(t *testing.T) {
	valid := []struct {
		from cici.RunStatus
		to   cici.RunStatus
	}{
		{cici.RunStatusQueued, cici.RunStatusRunning},
		{cici.RunStatusQueued, cici.RunStatusCancelled},
		{cici.RunStatusQueued, cici.RunStatusFailure},
		{cici.RunStatusRunning, cici.RunStatusSuccess},
		{cici.RunStatusRunning, cici.RunStatusFailure},
		{cici.RunStatusRunning, cici.RunStatusCancelled},
	}
	for _, tc := range valid {
		if !cici.CanTransition(tc.from, tc.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", tc.from, tc.to)
		}
		if err := cici.ValidateTransition(tc.from, tc.to); err != nil {
			t.Errorf("ValidateTransition(%s, %s) = %v, want nil", tc.from, tc.to, err)
		}
	}
}

func TestStateMachineInvalidTransitions(t *testing.T) {
	invalid := []struct {
		from cici.RunStatus
		to   cici.RunStatus
	}{
		{cici.RunStatusQueued, cici.RunStatusSuccess},
		{cici.RunStatusRunning, cici.RunStatusQueued},
		{cici.RunStatusSuccess, cici.RunStatusRunning},
		{cici.RunStatusFailure, cici.RunStatusRunning},
		{cici.RunStatusCancelled, cici.RunStatusSuccess},
		{cici.RunStatusQueued, cici.RunStatus("bogus")},
		{cici.RunStatus("bogus"), cici.RunStatusRunning},
	}
	for _, tc := range invalid {
		if cici.CanTransition(tc.from, tc.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", tc.from, tc.to)
		}
		err := cici.ValidateTransition(tc.from, tc.to)
		if !errors.Is(err, cici.ErrInvalidTransition) {
			t.Errorf("ValidateTransition(%s, %s) = %v, want ErrInvalidTransition", tc.from, tc.to, err)
		}
	}
}

func TestStateMachineSelfTransitionRejected(t *testing.T) {
	for _, s := range []cici.RunStatus{
		cici.RunStatusQueued,
		cici.RunStatusRunning,
		cici.RunStatusSuccess,
		cici.RunStatusFailure,
		cici.RunStatusCancelled,
	} {
		if cici.CanTransition(s, s) {
			t.Errorf("CanTransition(%s, %s) = true, want false", s, s)
		}
	}
}

func TestRunStatusNormalizesLegacyPending(t *testing.T) {
	got, err := cici.ParseRunStatus("pending")
	if err != nil {
		t.Fatalf("ParseRunStatus(pending) error: %v", err)
	}
	if got != cici.RunStatusQueued {
		t.Fatalf("ParseRunStatus(pending) = %q, want %q", got, cici.RunStatusQueued)
	}
	if got := cici.NormalizeRunStatus("pending"); got != string(cici.RunStatusQueued) {
		t.Fatalf("NormalizeRunStatus(pending) = %q, want %q", got, cici.RunStatusQueued)
	}
	if got := cici.NormalizeRunStatus("running"); got != string(cici.RunStatusRunning) {
		t.Fatalf("NormalizeRunStatus(running) = %q, want %q", got, cici.RunStatusRunning)
	}
}

func TestParseRunStatusRejectsUnknown(t *testing.T) {
	if _, err := cici.ParseRunStatus("exploded"); !errors.Is(err, cici.ErrInvalidRunStatus) {
		t.Fatalf("ParseRunStatus(exploded) = %v, want ErrInvalidRunStatus", err)
	}
}

func TestTerminalRunStatuses(t *testing.T) {
	terminal := []cici.RunStatus{
		cici.RunStatusSuccess,
		cici.RunStatusFailure,
		cici.RunStatusCancelled,
	}
	for _, s := range terminal {
		if !cici.IsTerminal(s) {
			t.Errorf("IsTerminal(%s) = false, want true", s)
		}
	}
	for _, s := range []cici.RunStatus{cici.RunStatusQueued, cici.RunStatusRunning} {
		if cici.IsTerminal(s) {
			t.Errorf("IsTerminal(%s) = true, want false", s)
		}
	}
}
