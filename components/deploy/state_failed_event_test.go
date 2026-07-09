package deploy_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

func TestDeployFailedEvent(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	gitComp := newGitStub(gitDir)
	buildsDir := t.TempDir()

	var mu sync.Mutex
	var failedEvents int

	dep := deploy.New(deploy.Options{
		DB:        database,
		Logger:    logger,
		BuildsDir: buildsDir,
		GitDir:    gitDir,
	})
	k := kernel.New(logger)
	k.EventBus().Subscribe(kernel.HookDef{
		Name: "deploy.failed",
		Handler: func(_ *kernel.Context, ev kernel.Event) error {
			mu.Lock()
			defer mu.Unlock()
			failedEvents++
			if ev.Data["deployment_id"] == nil {
				t.Error("missing deployment_id")
			}
			return nil
		},
	})
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, error_message, build_log, created_at)
		 VALUES ('dep-fail-event', 'repo-1', 'site-1', 'main', 'building', '', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := dep.TransitionState(context.Background(), "dep-fail-event", "failed"); err != nil {
		t.Fatalf("TransitionState: %v", err)
	}
	if err := dep.TransitionState(context.Background(), "dep-fail-event", "failed"); err != nil {
		t.Fatalf("idempotent TransitionState: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	count := failedEvents
	mu.Unlock()
	if count != 1 {
		t.Fatalf("expected exactly one deploy.failed event, got %d", count)
	}
}
