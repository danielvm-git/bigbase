package deploy_test

import (
	"context"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

// fleetShape models one live *.bigbase.click deployment shape. The matrix mirrors the
// 2026-06-23 fleet audit (static, Node static, Node SSR, Go, Python) that found 5 of 6
// sites down — every failure was a missing proxy host registration on restart.
//
// This is a CHARACTERIZATION test: it must stay green on the CURRENT code AND through the
// Supervisor refactor (see specs/adr/0004-no-docker-go-supervisor.md, issue #40). It pins
// the safety-net invariant: a deployment with status='running' and port>0 is re-registered
// in the proxy on every BigBase restart, for every app shape.
type fleetShape struct {
	id      string
	host    string
	port    int
	appType string
}

// liveFleetShapes covers the distinct app_type dimensions deployed at *.bigbase.click.
// Hostnames mirror the Sites dashboard so the fixture reads as the real fleet.
var liveFleetShapes = []fleetShape{
	{id: "f-static", host: "cleaninstallguide.bigbase.click", port: 10001, appType: "static"},
	{id: "f-node", host: "docklock.bigbase.click", port: 10002, appType: "node"},
	{id: "f-node-ssr", host: "docklocker.bigbase.click", port: 10003, appType: "node"},
	{id: "f-go", host: "bolao.bigbase.click", port: 10004, appType: "go"},
	{id: "f-python", host: "add-tutorial-requests-site.bigbase.click", port: 10005, appType: "python"},
	{id: "f-svelte", host: "big-dock-locker-site.bigbase.click", port: 10006, appType: "node"},
}

// TestRestoreFleetHostsOnRestart proves every running deployment's host is re-registered
// after a restart, across the full app-shape matrix. Drives the public Deploy lifecycle
// (Start/Stop) only — no internals, no real processes (resumeCandidates no-ops without
// build dirs). Green now; the refactor must keep it green.
func TestRestoreFleetHostsOnRestart(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	if err := database.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	// dep0 creates the deployments table via the real migrations (full schema).
	dep0 := deploy.New(deploy.Options{
		DB: database, Logger: logger, BuildsDir: t.TempDir(), GitDir: t.TempDir(),
		PublicDomain: "bigbase.click",
	})
	if err := dep0.Start(&kernel.Context{}); err != nil {
		t.Fatalf("dep0 start: %v", err)
	}

	for _, s := range liveFleetShapes {
		insertRunningDeployment(t, database, s)
	}
	// Boundary rows (T5): neither must be re-registered.
	insertRunningDeployment(t, database, fleetShape{id: "b-failed", host: "down.bigbase.click", port: 10090, appType: "go"})
	mustExec(t, database, "UPDATE deployments SET status = 'failed' WHERE id = 'b-failed'")
	insertRunningDeployment(t, database, fleetShape{id: "b-zeroport", host: "noport.bigbase.click", port: 0, appType: "static"})

	// Simulate a BigBase restart: Stop wipes in-memory app state; the DB persists.
	if err := dep0.Stop(&kernel.Context{}); err != nil {
		t.Fatalf("dep0 stop: %v", err)
	}

	// Fresh Deploy over the same DB — restoreRunningDeploymentHosts runs in Start().
	spy := &hostRegistrySpy{}
	dep2 := deploy.New(deploy.Options{
		DB: database, Logger: logger, BuildsDir: t.TempDir(), GitDir: t.TempDir(),
		PublicDomain: "bigbase.click", HostRouter: spy,
	})
	if err := dep2.Start(&kernel.Context{}); err != nil {
		t.Fatalf("dep2 start: %v", err)
	}
	t.Cleanup(func() { _ = dep2.Stop(&kernel.Context{}) })

	for _, s := range liveFleetShapes {
		if got := spy.hosts[s.host]; got != s.port {
			t.Errorf("%s (%s): host %q registered on :%d, want :%d — running deployment lost its route on restart",
				s.id, s.appType, s.host, got, s.port)
		}
	}
	if _, ok := spy.hosts["down.bigbase.click"]; ok {
		t.Error("failed deployment was re-registered on restart; only status='running' must be restored")
	}
	if _, ok := spy.hosts["noport.bigbase.click"]; ok {
		t.Error("port=0 deployment was re-registered on restart; port must be > 0")
	}
}

func insertRunningDeployment(t *testing.T, database *db.DB, s fleetShape) {
	t.Helper()
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, port, url, app_type, passthrough_paths, created_at)
		 VALUES (?, 'repo-'||?, 'site-'||?, 'main', 'running', ?, ?, ?, '', datetime('now'))`,
		s.id, s.id, s.id, s.port, "https://"+s.host, s.appType)
	if err != nil {
		t.Fatalf("insert deployment %s: %v", s.id, err)
	}
}

func mustExec(t *testing.T, database *db.DB, query string) {
	t.Helper()
	if _, err := database.ExecContext(context.Background(), query); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
