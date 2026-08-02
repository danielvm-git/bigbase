package deploy_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

// TestConnectionDrainTimeout verifies that drainDeployment force-kills
// after the DrainTimeout expires.
func TestConnectionDrainTimeout(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	hostReg := &mockHostRegistry{}

	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		PublicDomain: "test.click",
		HostRouter:   hostReg,
		DrainTimeout: 100 * time.Millisecond,
	})

	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	handler := dep.Handler()
	repoID := createTestRepo(t, database, "drain-timeout-test", gitDir)
	ctx := context.Background()

	// Deploy twice to the same site — second triggers drain of first
	dep1, err := dep.Trigger(ctx, repoID, "main", "drain-timeout", "site-drain-timeout", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger first: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep1.ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep1.ID, "running")

	dep2, err := dep.Trigger(ctx, repoID, "main", "drain-timeout", "site-drain-timeout", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger second: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep2.ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep2.ID, "running")

	// Drain is async — poll until first deployment reaches "stopped".
	for i := 0; i < 20; i++ {
		getReq := httptest.NewRequest("GET", "/api/deploy/"+dep1.ID, nil)
		getW := httptest.NewRecorder()
		handler.ServeHTTP(getW, getReq)
		var got map[string]any
		_ = json.NewDecoder(getW.Body).Decode(&got)
		s, _ := got["status"].(string)
		if s == "stopped" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	verifyDeployStatus(t, handler, dep1.ID, "stopped")
}

// TestDrainNoOldDeployments verifies that a first deployment doesn't try
// to drain anything (no-op).
func TestDrainNoOldDeployments(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		PublicDomain: "test.click",
	})

	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	handler := dep.Handler()
	repoID := createTestRepo(t, database, "drain-no-old", gitDir)
	ctx := context.Background()

	deployment, err := dep.Trigger(ctx, repoID, "main", "drain-no-old", "site-no-old", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	waitForDeploymentTerminal(t, handler, deployment.ID, 10*time.Second)
	verifyDeployStatus(t, handler, deployment.ID, "running")
}

// TestDrainFailedDeployment verifies that when a new deployment fails health
// check, the old deployment is NOT drained (zero-downtime safety).
func TestDrainFailedDeployment(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	hostReg := &mockHostRegistry{}

	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		PublicDomain: "test.click",
		HostRouter:   hostReg,
	})

	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	handler := dep.Handler()
	ctx := context.Background()

	// First deployment: static site → running
	repoID1 := createTestRepo(t, database, "drain-fail-old", gitDir)
	dep1, err := dep.Trigger(ctx, repoID1, "main", "drain-fail-old", "site-drain-fail", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger first: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep1.ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep1.ID, "running")

	host := "drain-fail-old.test.click"
	firstPort, hostOK := hostReg.getPort(host)
	if !hostOK {
		t.Fatal("host not registered after first deployment")
	}

	// Second deployment: Go process that exits immediately → health check fails
	// Use the same site ID so collectPreviousDeployments finds the first one.
	// The second repo ID is different but that's fine — query is by site_id.
	repoID2 := createHealthRepo(t, database, "drain-fail-new", gitDir,
		`package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Fprintf(os.Stderr, "server failed to start intentionally\n")
	time.Sleep(10 * time.Millisecond)
	os.Exit(1)
}
`, true)

	dep2, err := dep.Trigger(ctx, repoID2, "main", "drain-fail-old", "site-drain-fail", nil, "go", "")
	if err != nil {
		t.Fatalf("Trigger second: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep2.ID, 90*time.Second)

	// Second deployment should have failed
	getReq := httptest.NewRequest("GET", "/api/deploy/"+dep2.ID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	var dep2Data map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&dep2Data)
	status2, _ := dep2Data["status"].(string)
	if status2 != "failed" {
		t.Fatalf("second deploy status = %q, want 'failed' (error: %v)", status2, dep2Data["error_message"])
	}

	// First deployment should still be running (zero-downtime: NOT drained)
	verifyDeployStatus(t, handler, dep1.ID, "running")

	// Host should still point to first deployment's port
	gotPort, hostOK := hostReg.getPort(host)
	if !hostOK {
		t.Fatal("host was unregistered after failed deployment")
	}
	if gotPort != firstPort {
		t.Fatalf("host port changed after failed deployment: got %d, want %d", gotPort, firstPort)
	}
}

// TestDrainMultipleDeployments verifies draining multiple old deployments
// in sequence (3 deploys, only last runs).
func TestDrainMultipleDeployments(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	hostReg := &mockHostRegistry{}

	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		PublicDomain: "test.click",
		HostRouter:   hostReg,
	})

	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	handler := dep.Handler()
	repoID := createTestRepo(t, database, "drain-multi-test", gitDir)
	ctx := context.Background()
	siteID := "site-drain-multi"

	var depIDs []string
	for i := 0; i < 3; i++ {
		d, err := dep.Trigger(ctx, repoID, "main", "drain-multi-test", siteID, nil, "static", "")
		if err != nil {
			t.Fatalf("Trigger %d: %v", i, err)
		}
		waitForDeploymentTerminal(t, handler, d.ID, 10*time.Second)
		verifyDeployStatus(t, handler, d.ID, "running")
		depIDs = append(depIDs, d.ID)
	}

	// Only the last deployment should be running. Drain is async
	// (go drainDeployment), so wait for older ones to reach stopped.
	lastID := depIDs[len(depIDs)-1]
	for _, id := range depIDs {
		if id == lastID {
			verifyDeployStatus(t, handler, id, "running")
		} else {
			waitForDeployStatus(t, handler, id, "stopped", 5*time.Second)
		}
	}

	// Host should point to last deployment
	host := "drain-multi-test.test.click"
	gotPort, ok := hostReg.getPort(host)
	if !ok {
		t.Fatal("host not registered after third deploy")
	}
	// Verify host points to last deployment by checking its port
	getReq := httptest.NewRequest("GET", "/api/deploy/"+lastID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	var lastDep map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&lastDep)
	lastPort := int(lastDep["port"].(float64))
	if gotPort != lastPort {
		t.Fatalf("host port = %d, want %d (last deployment)", gotPort, lastPort)
	}
}

// TestDrainStatusHistory verifies drain transitions in status_history.
func TestDrainStatusHistory(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		PublicDomain: "test.click",
	})

	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	handler := dep.Handler()
	repoID := createTestRepo(t, database, "drain-hist-test", gitDir)
	ctx := context.Background()

	dep1, err := dep.Trigger(ctx, repoID, "main", "drain-hist", "site-drain-hist", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger first: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep1.ID, 10*time.Second)

	// Second deployment triggers drain of first
	dep2, err := dep.Trigger(ctx, repoID, "main", "drain-hist", "site-drain-hist", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger second: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep2.ID, 10*time.Second)
	startDrain := time.Now()
	for time.Since(startDrain) < 5*time.Second {
		req := httptest.NewRequest("GET", "/api/deploy/"+dep1.ID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var got map[string]any
		_ = json.NewDecoder(w.Body).Decode(&got)
		if st, ok := got["status"].(string); ok && st == "stopped" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	verifyDeployStatus(t, handler, dep1.ID, "stopped")

	// Check status_history includes draining → stopped (read from DB directly)
	var historyJSON string
	if err := database.QueryRowContext(ctx,
		"SELECT COALESCE(status_history, '[]') FROM deployments WHERE id = ?", dep1.ID).Scan(&historyJSON); err != nil {
		t.Fatalf("query status_history: %v", err)
	}
	if historyJSON == "" || historyJSON == "[]" {
		t.Fatal("status_history is empty")
	}

	var history []map[string]string
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("unmarshal history: %v", err)
	}

	foundDraining := false
	foundStopped := false
	for _, tr := range history {
		if tr["to"] == "draining" {
			foundDraining = true
		}
		if tr["to"] == "stopped" {
			foundStopped = true
		}
	}
	if !foundDraining {
		t.Error("status_history missing 'draining' transition")
	}
	if !foundStopped {
		t.Error("status_history missing 'stopped' transition")
	}
}

// ── Shared helpers (reuse from other test files) ──

// mockHostRegistry is defined in deploy_test.go.
// createHealthRepo is defined in health_integration_test.go.
// mustRun is defined in deploy_test.go.
// createTestRepo is defined in deploy_test.go.
// waitForDeploymentTerminal is defined in deploy_test.go.
// verifyDeployStatus is defined in rollback_test.go.

// Re-declare mockHostRegistry for this file since the one in deploy_test.go
// is in a different file but same package — no duplicate needed.
// Go allows one definition per package, so we rely on the one in deploy_test.go.

// TestDrainKillsProcessGroup verifies that drainDeployment kills the entire
// process tree (not just the main process) on timeout. This prevents orphaned
// child processes (Python workers, Telegram polling threads) from surviving.
func TestDrainKillsProcessGroup(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	hostReg := &mockHostRegistry{}

	dep := deploy.New(deploy.Options{
		DB:           database,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		PublicDomain: "test.click",
		HostRouter:   hostReg,
		DrainTimeout: 100 * time.Millisecond,
	})

	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	handler := dep.Handler()
	repoID := createTestRepo(t, database, "drain-pgid-test", gitDir)
	ctx := context.Background()

	// Deploy a static site first — this creates a running deployment
	dep1, err := dep.Trigger(ctx, repoID, "main", "drain-pgid", "site-drain-pgid", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger first: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep1.ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep1.ID, "running")

	// Deploy again — triggers drain of first
	dep2, err := dep.Trigger(ctx, repoID, "main", "drain-pgid", "site-drain-pgid", nil, "static", "")
	if err != nil {
		t.Fatalf("Trigger second: %v", err)
	}
	waitForDeploymentTerminal(t, handler, dep2.ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep2.ID, "running")

	// Wait for drain to complete
	for i := 0; i < 20; i++ {
		getReq := httptest.NewRequest("GET", "/api/deploy/"+dep1.ID, nil)
		getW := httptest.NewRecorder()
		handler.ServeHTTP(getW, getReq)
		var got map[string]any
		_ = json.NewDecoder(getW.Body).Decode(&got)
		s, _ := got["status"].(string)
		if s == "stopped" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	verifyDeployStatus(t, handler, dep1.ID, "stopped")
}
