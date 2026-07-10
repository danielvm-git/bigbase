package deploy_test

import (
	"context"
	"sync"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

func TestTransitionStateEmitsEvent(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ev", gitDir)

	depID := "dep-ev-001"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	bus := kernel.NewEventBus()
	// Must set AFTER setupDeploy (kernel.Start overrides eventBus)
	dep.EventBus(bus)

	var mu sync.Mutex
	var events []string

	bus.Subscribe(kernel.HookDef{
		Name: "deploy.state_changed",
		Handler: func(ctx *kernel.Context, e kernel.Event) error {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e.Data["to_state"].(string))
			return nil
		},
	})

	ctx := context.Background()
	if err := dep.TransitionState(ctx, depID, string(deploy.StateBuilding)); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if err := dep.TransitionState(ctx, depID, string(deploy.StateRunning)); err != nil {
		t.Fatalf("transition: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(events), events)
	}
	if events[0] != "building" || events[1] != "running" {
		t.Errorf("events = %v, want [building running]", events)
	}
}

func TestTransitionStateNilEventBus(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-nev", gitDir)

	depID := "dep-nev-001"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	// eventBus is nil (no EventBus() call)
	ctx := context.Background()
	if err := dep.TransitionState(ctx, depID, string(deploy.StateBuilding)); err != nil {
		t.Fatalf("nil eventBus should not block transition: %v", err)
	}
	verifyStatus(t, database, depID, "building")
}
