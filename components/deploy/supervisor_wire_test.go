package deploy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// TestTriggerRunsThroughSupervisor proves that when a running deployment has an
// existing build directory, resumeCandidates routes it through the Supervisor
// (and thus calls Runner.Spawn) rather than directly calling serveStatic/startApp.
// This is the wiring test for e53s04.
func TestTriggerRunsThroughSupervisor(t *testing.T) {
	database := db.New(db.Options{Path: ":memory:", Logger: noopLogger{}})
	if err := database.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	// dep0 creates the deployments table schema.
	dep0 := New(Options{DB: database, Logger: noopLogger{}, BuildsDir: t.TempDir(), GitDir: t.TempDir()})
	if err := dep0.Start(&kernel.Context{}); err != nil {
		t.Fatalf("dep0 start: %v", err)
	}
	_ = dep0.Stop(&kernel.Context{})

	// git_repos table is owned by the git component; create it inline for test isolation.
	_, err := database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (id TEXT PRIMARY KEY, name TEXT NOT NULL DEFAULT '')`)
	if err != nil {
		t.Fatalf("create git_repos: %v", err)
	}
	_, err = database.ExecContext(context.Background(),
		`INSERT INTO git_repos (id, name) VALUES ('repo-1', 'my-static-app')`)
	if err != nil {
		t.Fatalf("insert git_repos: %v", err)
	}

	// Build dir for "dep-1" must exist so resumeCandidates doesn't skip it.
	buildsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(buildsDir, "dep-1"), 0o755); err != nil {
		t.Fatalf("mkdirAll: %v", err)
	}

	_, err = database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, port, url, app_type, passthrough_paths, created_at)
		 VALUES ('dep-1', 'repo-1', 'site-1', 'main', 'running', 10199, 'https://my-static-app.bigbase.click', 'static', '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	fakeRunner := &FakeRunner{queue: []*FakeInstance{newFakeInstance(nil)}}
	reg := &supervisorRegistry{}

	dep := New(Options{
		DB:           database,
		Logger:       noopLogger{},
		BuildsDir:    buildsDir,
		GitDir:       t.TempDir(),
		PublicDomain: "bigbase.click",
		HostRouter:   reg,
		Runner:       fakeRunner,
	})
	if err := dep.Start(&kernel.Context{}); err != nil {
		t.Fatalf("dep start: %v", err)
	}
	t.Cleanup(func() { _ = dep.Stop(&kernel.Context{}) })

	// resumeCandidates runs in a goroutine; wait for Spawn to be called.
	eventually(t, func() bool { return fakeRunner.calls >= 1 })

	if fakeRunner.calls != 1 {
		t.Errorf("Runner.Spawn calls = %d, want 1 (resumeCandidates must route through Supervisor)", fakeRunner.calls)
	}
}
