package deploy_test

import (
	"context"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

type hostRegistrySpy struct {
	hosts map[string]int
}

func (h *hostRegistrySpy) RegisterDeploymentHost(host string, port int, siteID string, _ []string, _ map[string]string) error {
	if h.hosts == nil {
		h.hosts = make(map[string]int)
	}
	h.hosts[host] = port
	return nil
}

func (h *hostRegistrySpy) UnregisterDeploymentHost(host string) {
	delete(h.hosts, host)
}

func TestDeployRestoresHostsOnStart(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	if err := database.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	if err := database.Migrate(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		site_id TEXT NOT NULL DEFAULT '',
		branch TEXT NOT NULL DEFAULT 'main',
		commit_sha TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		url TEXT DEFAULT '',
		port INTEGER DEFAULT 0,
		app_type TEXT DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, url, port, created_at)
		 VALUES ('dep1', 'repo1', 's1', 'main', 'running', 'https://my-app.bigbase.click', 10499, '2026-06-04T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	spy := &hostRegistrySpy{}
	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		PublicDomain: "bigbase.click",
		HostRouter:   spy,
	})
	if err := dep.Start(&kernel.Context{}); err != nil {
		t.Fatalf("deploy start: %v", err)
	}

	if spy.hosts["my-app.bigbase.click"] != 10499 {
		t.Fatalf("hosts = %#v, want my-app.bigbase.click:10499", spy.hosts)
	}
}
