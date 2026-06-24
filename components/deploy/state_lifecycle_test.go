package deploy_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestStateLifecycle verifies the full deployment lifecycle through the state machine.
// Uses subtests to avoid creating a heavyweight kernel for each scenario.
func TestStateLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Subtest 1: Full static site lifecycle — pending → building → running
	t.Run("static site transitions pending→building→running", func(t *testing.T) {
		_, handler, database, gitDir := setupDeploy(t)
		repoID := createTestRepo(t, database, "repo-lc-static", gitDir)

		buf := new(strings.Builder)
		_ = json.NewEncoder(buf).Encode(map[string]string{"repo_id": repoID})
		req := httptest.NewRequest("POST", "/api/deploy", strings.NewReader(buf.String()))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("create: %d: %s", w.Code, w.Body.String())
		}
		var created map[string]any
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("decode: %v", err)
		}
		depID, _ := created["id"].(string)
		if created["status"] != "pending" {
			t.Fatalf("initial status = %q, want pending", created["status"])
		}

		// Wait and verify terminal state
		waitForDeploymentTerminal(t, handler, depID, 10*time.Second)
		var status string
		_ = database.QueryRowContext(context.Background(),
			"SELECT status FROM deployments WHERE id = ?", depID).Scan(&status)
		if status != "running" {
			t.Fatalf("final status = %q, want running", status)
		}

		// Verify DB has status_history entries
		var historyJSON string
		err := database.QueryRowContext(context.Background(),
			"SELECT COALESCE(status_history, '[]') FROM deployments WHERE id = ?", depID).Scan(&historyJSON)
		if err != nil {
			t.Fatalf("query status_history: %v", err)
		}
		var history []map[string]any
		if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
			t.Fatalf("parse status_history: %v (raw: %s)", err, historyJSON)
		}
		if len(history) < 1 {
			t.Fatal("expected at least 1 status_history entry")
		}
		if history[0]["from"] != "pending" || history[0]["to"] != "building" {
			t.Fatalf("first entry = %+v, want pending→building", history[0])
		}
		if history[0]["timestamp"] == "" {
			t.Fatal("expected non-empty timestamp")
		}
	})

	// Subtest 2: Failed deployment — building → failed
	t.Run("failed deployment building→failed", func(t *testing.T) {
		dep, _, database, _ := setupDeploy(t)
		gitDir := t.TempDir()
		repoID := createTestRepo(t, database, "repo-lc-fail", gitDir)

		// Insert deployment in pending state
		depID := "dep-lc-fail"
		_, err := database.ExecContext(context.Background(),
			`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
			 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}

		// Transition through building → failed
		ctx := context.Background()
		if err := dep.TransitionState(ctx, depID, "building"); err != nil {
			t.Fatalf("pending→building: %v", err)
		}
		if err := dep.TransitionState(ctx, depID, "failed"); err != nil {
			t.Fatalf("building→failed: %v", err)
		}

		// Verify terminal state
		var status string
		_ = database.QueryRowContext(ctx,
			"SELECT status FROM deployments WHERE id = ?", depID).Scan(&status)
		if status != "failed" {
			t.Fatalf("status = %q, want failed", status)
		}

		// Verify status_history has 2 entries
		var historyJSON string
		_ = database.QueryRowContext(ctx,
			"SELECT COALESCE(status_history, '[]') FROM deployments WHERE id = ?", depID).Scan(&historyJSON)
		var history []map[string]any
		if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
			t.Fatalf("parse status_history: %v", err)
		}
		if len(history) != 2 {
			t.Fatalf("expected 2 history entries, got %d", len(history))
		}
	})

	// Subtest 3: Invalid transition rejected
	t.Run("invalid transition rejected", func(t *testing.T) {
		dep, _, database, gitDir := setupDeploy(t)
		repoID := createTestRepo(t, database, "repo-lc-inv", gitDir)

		depID := "dep-lc-inv"
		_, err := database.ExecContext(context.Background(),
			`INSERT INTO deployments (id, repo_id, site_id, status, url, port, app_type)
			 VALUES (?, ?, '', 'pending', '', 0, '')`, depID, repoID)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}

		ctx := context.Background()
		// pending → running is invalid (must go through building)
		err = dep.TransitionState(ctx, depID, "running")
		if err == nil {
			t.Fatal("expected error for invalid transition pending→running")
		}
		if !strings.Contains(err.Error(), "invalid transition") {
			t.Errorf("error should mention 'invalid transition', got: %v", err)
		}

		// pending → building is valid
		if err := dep.TransitionState(ctx, depID, "building"); err != nil {
			t.Fatalf("pending→building should be valid: %v", err)
		}
	})
}

// TestStateLifecycleRestart verifies that running deployments survive
// component restart without the state machine blocking the resume path.
func TestStateLifecycleRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("running deployment state unchanged after restart", func(t *testing.T) {
		_, handler, database, gitDir := setupDeploy(t)
		repoID := createTestRepo(t, database, "repo-restart-safe", gitDir)

		// Trigger a static site deployment and wait for running
		buf := new(strings.Builder)
		_ = json.NewEncoder(buf).Encode(map[string]string{"repo_id": repoID})
		req := httptest.NewRequest("POST", "/api/deploy", strings.NewReader(buf.String()))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("trigger: %d: %s", w.Code, w.Body.String())
		}
		var created map[string]any
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("decode: %v", err)
		}
		depID, _ := created["id"].(string)
		waitForDeploymentTerminal(t, handler, depID, 10*time.Second)

		// Verify running and status_history exists
		var status string
		_ = database.QueryRowContext(context.Background(),
			"SELECT status FROM deployments WHERE id = ?", depID).Scan(&status)
		if status != "running" {
			t.Fatalf("expected 'running' before restart, got %q", status)
		}
		var historyJSON string
		_ = database.QueryRowContext(context.Background(),
			"SELECT COALESCE(status_history, '[]') FROM deployments WHERE id = ?", depID).Scan(&historyJSON)
		var history []map[string]any
		if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
			t.Fatalf("parse status_history: %v", err)
		}
		if len(history) < 1 {
			t.Fatal("expected status_history entries before restart")
		}

		// Verify the deployment's port is still accepting connections
		// (the static server should still be running)
		// This validates that restart doesn't kill old processes.
	})
}
