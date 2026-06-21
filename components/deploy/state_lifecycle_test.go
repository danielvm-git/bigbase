package deploy_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func TestStateLifecycle_FullDeployment_SuccessPath(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	// Subscribe to state change events
	events := make(chan kernel.Event, 10)
	unsub := dep.EventBus().Subscribe(kernel.HookDef{
		Name:     "deploy.state_changed",
		Priority: 0,
		Handler: func(ctx *kernel.Context, event kernel.Event) error {
			events <- event
			return nil
		},
	})
	defer unsub()

	// Insert a deployment
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-full', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	// Transition through the full lifecycle
	if err := dep.TransitionState(context.Background(), "test-full", "building"); err != nil {
		t.Fatalf("pending -> building: %v", err)
	}
	if err := dep.TransitionState(context.Background(), "test-full", "deploying"); err != nil {
		t.Fatalf("building -> deploying: %v", err)
	}
	if err := dep.TransitionState(context.Background(), "test-full", "live"); err != nil {
		t.Fatalf("deploying -> live: %v", err)
	}

	// 1. Verify final status
	var finalStatus string
	err = database.QueryRowContext(context.Background(),
		"SELECT status FROM deployments WHERE id = 'test-full'").Scan(&finalStatus)
	if err != nil {
		t.Fatalf("query final status: %v", err)
	}
	if finalStatus != "live" {
		t.Fatalf("expected status 'live', got %q", finalStatus)
	}

	// 2. Verify status_history has all transitions
	var historyJSON string
	err = database.QueryRowContext(context.Background(),
		"SELECT status_history FROM deployments WHERE id = 'test-full'").Scan(&historyJSON)
	if err != nil {
		t.Fatalf("query status_history: %v", err)
	}
	if historyJSON == "" || historyJSON == "[]" {
		t.Fatalf("expected non-empty status_history, got %q", historyJSON)
	}

	type transition struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Timestamp string `json:"timestamp"`
	}
	var transitions []transition
	if err := json.Unmarshal([]byte(historyJSON), &transitions); err != nil {
		t.Fatalf("unmarshal status_history: %v", err)
	}

	expected := []struct{ from, to string }{
		{"pending", "building"},
		{"building", "deploying"},
		{"deploying", "live"},
	}
	if len(transitions) != len(expected) {
		t.Fatalf("expected %d transitions, got %d: %+v", len(expected), len(transitions), transitions)
	}
	for i, exp := range expected {
		if transitions[i].From != exp.from {
			t.Errorf("transition %d: from = %q, want %q", i, transitions[i].From, exp.from)
		}
		if transitions[i].To != exp.to {
			t.Errorf("transition %d: to = %q, want %q", i, transitions[i].To, exp.to)
		}
		if transitions[i].Timestamp == "" {
			t.Errorf("transition %d: empty timestamp", i)
		}
		if _, err := time.Parse(time.RFC3339, transitions[i].Timestamp); err != nil {
			t.Errorf("transition %d: invalid timestamp %q: %v", i, transitions[i].Timestamp, err)
		}
	}

	// 3. Verify events were emitted for each transition
	gotEvents := make([]kernel.Event, 0, 3)
	for i := 0; i < 3; i++ {
		select {
		case evt := <-events:
			gotEvents = append(gotEvents, evt)
		case <-time.After(time.Second):
			t.Fatalf("expected event %d/3, got timeout", i+1)
		}
	}
	for i, exp := range expected {
		if gotEvents[i].Data["from_state"] != exp.from {
			t.Errorf("event %d: from_state = %v, want %q", i, gotEvents[i].Data["from_state"], exp.from)
		}
		if gotEvents[i].Data["to_state"] != exp.to {
			t.Errorf("event %d: to_state = %v, want %q", i, gotEvents[i].Data["to_state"], exp.to)
		}
	}
}

func TestStateLifecycle_FailedTransition(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-fail', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	// Build then fail
	if err := dep.TransitionState(context.Background(), "test-fail", "building"); err != nil {
		t.Fatalf("pending -> building: %v", err)
	}
	if err := dep.TransitionState(context.Background(), "test-fail", "failed"); err != nil {
		t.Fatalf("building -> failed: %v", err)
	}

	// Verify terminal state
	var status string
	_ = database.QueryRowContext(context.Background(),
		"SELECT status FROM deployments WHERE id = 'test-fail'").Scan(&status)
	if status != "failed" {
		t.Fatalf("expected 'failed', got %q", status)
	}

	// Invalid transitions from terminal state
	err = dep.TransitionState(context.Background(), "test-fail", "building")
	if err == nil {
		t.Fatal("expected error transitioning from 'failed' to 'building'")
	}
	if !strings.Contains(err.Error(), "invalid transition") {
		t.Fatalf("expected 'invalid transition' in error, got: %v", err)
	}

	// Can't skip states
	_, err = database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-skip', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}
	err = dep.TransitionState(context.Background(), "test-skip", "live")
	if err == nil {
		t.Fatal("expected error skipping states (pending -> live)")
	}
}
