package deploy_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/danielvm/bigbase/kernel"
)

func TestStateEvents_EmittedOnTransition(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-events', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

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

	if err := dep.TransitionState(context.Background(), "test-events", "building"); err != nil {
		t.Fatalf("transition: %v", err)
	}

	select {
	case evt := <-events:
		if evt.Name != "deploy.state_changed" {
			t.Errorf("event name = %q, want 'deploy.state_changed'", evt.Name)
		}
		if evt.Data["deployment_id"] != "test-events" {
			t.Errorf("deployment_id = %v", evt.Data["deployment_id"])
		}
		if evt.Data["from_state"] != "pending" {
			t.Errorf("from_state = %v", evt.Data["from_state"])
		}
		if evt.Data["to_state"] != "building" {
			t.Errorf("to_state = %v", evt.Data["to_state"])
		}
		if evt.Data["timestamp"] == "" {
			t.Errorf("timestamp is empty")
		}
	default:
		t.Fatal("expected event on valid transition")
	}
}

func TestStateEvents_NoEventOnInvalidTransition(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-noevent', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	events := make(chan kernel.Event, 5)
	unsub := dep.EventBus().Subscribe(kernel.HookDef{
		Name:     "deploy.state_changed",
		Priority: 0,
		Handler: func(ctx *kernel.Context, event kernel.Event) error {
			events <- event
			return nil
		},
	})
	defer unsub()

	err = dep.TransitionState(context.Background(), "test-noevent", "live")
	if err == nil {
		t.Fatal("expected error for invalid transition pending -> live")
	}

	select {
	case evt := <-events:
		t.Fatalf("unexpected event on invalid transition: %+v", evt)
	default:
		// Expected — no event emitted
	}
}

func TestStateEvents_MultipleTransitions_EmitEach(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-multi', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

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

	transitions := []struct{ from, to string }{
		{"pending", "building"},
		{"building", "deploying"},
		{"deploying", "live"},
	}
	for _, tr := range transitions {
		if err := dep.TransitionState(context.Background(), "test-multi", tr.to); err != nil {
			t.Fatalf("transition %s -> %s: %v", tr.from, tr.to, err)
		}
	}

	got := make([]kernel.Event, 0, 3)
	for i := 0; i < 3; i++ {
		select {
		case evt := <-events:
			got = append(got, evt)
		default:
			t.Fatalf("expected event %d/3, got none", i+1)
		}
	}

	for i, tr := range transitions {
		if got[i].Data["from_state"] != tr.from {
			t.Errorf("event %d: from_state = %v, want %q", i, got[i].Data["from_state"], tr.from)
		}
		if got[i].Data["to_state"] != tr.to {
			t.Errorf("event %d: to_state = %v, want %q", i, got[i].Data["to_state"], tr.to)
		}
	}
}

func TestStateEvents_EventData_JSONSerializable(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-json', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	events := make(chan kernel.Event, 5)
	unsub := dep.EventBus().Subscribe(kernel.HookDef{
		Name:     "deploy.state_changed",
		Priority: 0,
		Handler: func(ctx *kernel.Context, event kernel.Event) error {
			events <- event
			return nil
		},
	})
	defer unsub()

	if err := dep.TransitionState(context.Background(), "test-json", "building"); err != nil {
		t.Fatalf("transition: %v", err)
	}

	select {
	case evt := <-events:
		data, err := json.Marshal(evt)
		if err != nil {
			t.Fatalf("marshal event: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("marshaled event is empty")
		}
	default:
		t.Fatal("expected event")
	}
}
