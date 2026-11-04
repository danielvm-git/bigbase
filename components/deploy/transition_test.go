package deploy_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
)

func TestTransitionStateValidFullLifecycle(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ts", gitDir)

	// Insert a deployment directly (simulating trigger without running full build)
	depID := "dep-ts-001"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	ctx := context.Background()

	// 1. pending → building (valid)
	if err := dep.TransitionState(ctx, depID, string(deploy.StateBuilding)); err != nil {
		t.Fatalf("pending → building should be valid: %v", err)
	}
	verifyStatus(t, database, depID, "building")

	// 2. building → deploying (valid)
	if err := dep.TransitionState(ctx, depID, string(deploy.StateDeploying)); err != nil {
		t.Fatalf("building → deploying should be valid: %v", err)
	}
	verifyStatus(t, database, depID, "deploying")

	// 3. deploying → running (valid)
	if err := dep.TransitionState(ctx, depID, string(deploy.StateRunning)); err != nil {
		t.Fatalf("deploying → running should be valid: %v", err)
	}
	verifyStatus(t, database, depID, "running")

	// 4. Verify status_history has 3 entries
	verifyStatusHistory(t, database, depID, 3)
}

func TestTransitionStateInvalidRejected(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ts2", gitDir)

	depID := "dep-ts-002"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	ctx := context.Background()

	// pending → running is invalid (must go through building, then deploying/running)
	err = dep.TransitionState(ctx, depID, string(deploy.StateRunning))
	if err == nil {
		t.Fatalf("pending → running should be invalid, but got nil")
	}
	if !errors.Is(err, deploy.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got: %v", err)
	}

	// Status unchanged after rejected transition
	verifyStatus(t, database, depID, "pending")
	verifyStatusHistory(t, database, depID, 0)
}

func TestTransitionStateIdempotent(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ts3", gitDir)

	depID := "dep-ts-003"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'building', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	ctx := context.Background()

	// building → building (same status) should be a no-op, not an error
	if err := dep.TransitionState(ctx, depID, string(deploy.StateBuilding)); err != nil {
		t.Errorf("idempotent transition should return nil, got: %v", err)
	}

	// Status unchanged
	verifyStatus(t, database, depID, "building")
	// No history entries added for idempotent transition
	verifyStatusHistory(t, database, depID, 0)
}

func TestTransitionStateLenientMode(t *testing.T) {
	// Lenient mode exists only for legacy rows that still carry a status the
	// state machine no longer declares (e.g. "replaced" written by older
	// builds). Normal flows write only declared DeploymentStates (#303).
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ts4", gitDir)

	depID := "dep-ts-004"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'replaced', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	ctx := context.Background()

	// "replaced" is not in the state machine → lenient mode should allow
	if err := dep.TransitionState(ctx, depID, string(deploy.StateRunning)); err != nil {
		t.Errorf("lenient mode: replaced → running should be allowed, got: %v", err)
	}

	verifyStatus(t, database, depID, "running")
	verifyStatusHistory(t, database, depID, 1)
}

func TestConcurrentTransitionSingleTerminalOutcome(t *testing.T) {
	// #302: competing transitions must not both write. failed and rolled_back
	// are both reachable from running but neither allows an outgoing transition,
	// so exactly one can win and the loser must report a deterministic error.
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ts-race", gitDir)

	depID := "dep-ts-race"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'running', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	ctx := context.Background()
	targets := []string{string(deploy.StateFailed), string(deploy.StateRolledBack)}

	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		errs    = make([]error, len(targets))
		mu      sync.Mutex
		results = make([]string, 0, len(targets))
	)
	for i, target := range targets {
		wg.Add(1)
		go func(i int, target string) {
			defer wg.Done()
			<-start
			err := dep.TransitionState(ctx, depID, target)
			mu.Lock()
			errs[i] = err
			results = append(results, target)
			mu.Unlock()
		}(i, target)
	}
	close(start)
	wg.Wait()

	successes := 0
	for i, err := range errs {
		if err == nil {
			successes++
		} else if !strings.Contains(err.Error(), "invalid transition") &&
			!strings.Contains(err.Error(), "changed concurrently") {
			t.Errorf("loser %d returned non-deterministic error: %v", i, err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful transitions = %d, want exactly 1 (errs: %v)", successes, errs)
	}

	var final string
	if err := database.QueryRowContext(ctx,
		"SELECT status FROM deployments WHERE id = ?", depID).Scan(&final); err != nil {
		t.Fatalf("query final status: %v", err)
	}
	if final != string(deploy.StateFailed) && final != string(deploy.StateRolledBack) {
		t.Fatalf("final status = %q, want a single terminal outcome", final)
	}

	// Only the winning transition may append history.
	verifyStatusHistory(t, database, depID, 1)
}

func TestTransitionStateNotFound(t *testing.T) {
	dep, _, _, _ := setupDeploy(t)

	err := dep.TransitionState(context.Background(), "missing-deployment", string(deploy.StateBuilding))
	if err == nil {
		t.Fatal("expected error for missing deployment")
	}
}

func TestTransitionStateStatusHistoryPersistence(t *testing.T) {
	dep, _, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-ts5", gitDir)

	depID := "dep-ts-005"
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
		 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	ctx := context.Background()

	// pending → building
	_ = dep.TransitionState(ctx, depID, string(deploy.StateBuilding))
	_ = dep.TransitionState(ctx, depID, string(deploy.StateRunning))

	// Verify status_history JSON array
	verifyStatusHistory(t, database, depID, 2)

	// Check that entries have valid timestamps
	var historyJSON string
	_ = database.QueryRowContext(ctx,
		"SELECT status_history FROM deployments WHERE id = ?", depID).Scan(&historyJSON)

	var history []map[string]string
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("parse status_history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	if history[0]["from"] != "pending" || history[0]["to"] != "building" {
		t.Errorf("entry 0: %+v, want pending→building", history[0])
	}
	if history[1]["from"] != "building" || history[1]["to"] != "running" {
		t.Errorf("entry 1: %+v, want building→running", history[1])
	}
	if history[0]["timestamp"] == "" || history[1]["timestamp"] == "" {
		t.Error("timestamps should not be empty")
	}
}

// verifyStatus checks the current status of a deployment in the DB.
func verifyStatus(t *testing.T, database *db.DB, depID, expected string) {
	t.Helper()
	var status string
	err := database.QueryRowContext(context.Background(),
		"SELECT status FROM deployments WHERE id = ?", depID).Scan(&status)
	if err != nil {
		t.Fatalf("query status for %s: %v", depID, err)
	}
	if status != expected {
		t.Errorf("status = %q, want %q", status, expected)
	}
}

// verifyStatusHistory checks the number of entries in status_history.
func verifyStatusHistory(t *testing.T, database *db.DB, depID string, expectedCount int) {
	t.Helper()
	var historyJSON string
	err := database.QueryRowContext(context.Background(),
		"SELECT status_history FROM deployments WHERE id = ?", depID).Scan(&historyJSON)
	if err != nil {
		t.Fatalf("query status_history for %s: %v", depID, err)
	}

	var history []interface{}
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("parse status_history: %v (raw: %s)", err, historyJSON)
	}
	if len(history) != expectedCount {
		t.Errorf("status_history entries = %d, want %d (raw: %s)", len(history), expectedCount, historyJSON)
	}
}
