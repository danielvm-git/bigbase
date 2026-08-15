package deploy_test

// e86s04: deploy.state_changed and deploy.failed events must carry org_id
// (resolved via the deployment's site) so monitoring can ingest org-scoped rows.

import (
	"context"
	"sync"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

func TestDeployEventsIncludeOrgID(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-org-ev", gitDir)

	if _, err := database.Exec(
		`CREATE TABLE IF NOT EXISTS sites (id TEXT PRIMARY KEY, name TEXT, git_repo_id TEXT, org_id INTEGER)`); err != nil {
		t.Fatalf("create sites: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO sites (id, name, git_repo_id, org_id) VALUES ('site-42', 'app', ?, 42)`, repoID); err != nil {
		t.Fatalf("insert site: %v", err)
	}
	depID := "dep-org-ev-1"
	if _, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, 'site-42', 'pending', '', 0, '')`, depID, repoID); err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	bus := kernel.NewEventBus()
	dep.EventBus(bus)

	var mu sync.Mutex
	orgByHook := map[string]int64{}
	presence := map[string]bool{}
	capture := func(hook string) kernel.HookDef {
		return kernel.HookDef{
			Name: hook,
			Handler: func(_ *kernel.Context, e kernel.Event) error {
				mu.Lock()
				defer mu.Unlock()
				if v, ok := e.Data["org_id"].(int64); ok {
					presence[hook] = true
					orgByHook[hook] = v
				}
				return nil
			},
		}
	}
	bus.Subscribe(capture("deploy.state_changed"))
	bus.Subscribe(capture("deploy.failed"))

	ctx := context.Background()
	if err := dep.TransitionState(ctx, depID, string(deploy.StateBuilding)); err != nil {
		t.Fatalf("transition building: %v", err)
	}
	// pending→…→failed drives emitDeployFailed too.
	if err := dep.TransitionState(ctx, depID, string(deploy.StateFailed)); err != nil {
		t.Fatalf("transition failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, hook := range []string{"deploy.state_changed", "deploy.failed"} {
		if !presence[hook] {
			t.Errorf("%s event missing org_id field", hook)
		} else if orgByHook[hook] != 42 {
			t.Errorf("%s org_id = %d, want 42", hook, orgByHook[hook])
		}
	}
}
