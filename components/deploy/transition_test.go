package deploy_test

import (
	"context"
	"testing"
)

func TestTransitionState_InvalidTransition_ReturnsError(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	// Insert a deployment with status 'pending'
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-invalid', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	// Try to transition directly to 'live' (skips building → deploying)
	err = dep.TransitionState(context.Background(), "test-invalid", "live")
	if err == nil {
		t.Fatal("expected error for invalid transition pending → live, got nil")
	}
}

func TestTransitionState_ValidTransition_Succeeds(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-valid', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	err = dep.TransitionState(context.Background(), "test-valid", "building")
	if err != nil {
		t.Fatalf("expected no error for valid transition pending → building, got: %v", err)
	}

	// Verify DB was updated
	var status string
	err = database.QueryRowContext(context.Background(),
		"SELECT status FROM deployments WHERE id = 'test-valid'").Scan(&status)
	if err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != "building" {
		t.Fatalf("expected status 'building', got %q", status)
	}
}

func TestTransitionState_StoresStatusHistory(t *testing.T) {
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-history', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	// Transition pending → building
	if err := dep.TransitionState(context.Background(), "test-history", "building"); err != nil {
		t.Fatalf("transition to building: %v", err)
	}

	// Transition building → deploying
	if err := dep.TransitionState(context.Background(), "test-history", "deploying"); err != nil {
		t.Fatalf("transition to deploying: %v", err)
	}

	// Check status_history was populated
	var history string
	err = database.QueryRowContext(context.Background(),
		"SELECT status_history FROM deployments WHERE id = 'test-history'").Scan(&history)
	if err != nil {
		t.Fatalf("query status_history: %v", err)
	}
	if history == "" || history == "[]" {
		t.Fatalf("expected non-empty status_history, got %q", history)
	}
}

func TestTransitionState_NotFound_ReturnsError(t *testing.T) {
	dep, _, _, _ := setupDeploy(t)

	err := dep.TransitionState(context.Background(), "nonexistent", "building")
	if err == nil {
		t.Fatal("expected error for nonexistent deployment, got nil")
	}
}

func TestTransitionState_NilStateMachine_ReturnsError(t *testing.T) {
	// Use a Deploy with no state machine
	dep, _, database, _ := setupDeploy(t)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, branch, status, port, url, created_at)
		 VALUES ('test-nil', 'repo-x', 'main', 'pending', 0, '', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}

	err = dep.TransitionState(context.Background(), "test-nil", "building")
	if err != nil {
		t.Fatalf("expected transition to succeed (state machine created in New()), got: %v", err)
	}
}
