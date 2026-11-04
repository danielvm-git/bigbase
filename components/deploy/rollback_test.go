package deploy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/kernel"
)

func TestRollbackStaticSite(t *testing.T) {
	// GIVEN: a static site deployed twice
	dep, handler, database, gitDir := setupDeploy(t)
	_ = dep

	repoID := createTestRepo(t, database, "repo-rollback-1", gitDir)

	// Create first deployment
	body1 := map[string]string{"repo_id": repoID, "site_id": "site-rollback-1"}
	var buf1 bytes.Buffer
	_ = json.NewEncoder(&buf1).Encode(body1)
	req1 := httptest.NewRequest("POST", "/api/deploy", &buf1)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first deploy expected 201, got %d: %s", w1.Code, w1.Body.String())
	}

	var dep1 map[string]any
	_ = json.NewDecoder(w1.Body).Decode(&dep1)
	dep1ID := dep1["id"].(string)

	// Wait for first deployment to reach running
	waitForDeploymentTerminal(t, handler, dep1ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep1ID, "running")

	// Create second deployment (same repo, same site)
	body2 := map[string]string{"repo_id": repoID, "site_id": "site-rollback-1"}
	var buf2 bytes.Buffer
	_ = json.NewEncoder(&buf2).Encode(body2)
	req2 := httptest.NewRequest("POST", "/api/deploy", &buf2)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("second deploy expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var dep2 map[string]any
	_ = json.NewDecoder(w2.Body).Decode(&dep2)
	dep2ID := dep2["id"].(string)

	// Wait for second deployment to reach running
	waitForDeploymentTerminal(t, handler, dep2ID, 10*time.Second)
	verifyDeployStatus(t, handler, dep2ID, "running")

	// First deployment drains asynchronously after the new one is healthy.
	waitForDeployStatus(t, handler, dep1ID, "stopped", 5*time.Second)

	// WHEN: rollback the current deployment
	rollbackReq := httptest.NewRequest("POST", "/api/deploy/"+dep2ID+"/rollback", nil)
	rollbackW := httptest.NewRecorder()
	handler.ServeHTTP(rollbackW, rollbackReq)

	if rollbackW.Code != http.StatusOK {
		t.Fatalf("rollback expected 200, got %d: %s", rollbackW.Code, rollbackW.Body.String())
	}

	var rollbackResp map[string]any
	_ = json.NewDecoder(rollbackW.Body).Decode(&rollbackResp)

	// THEN: first deployment is running again, second is rolled back
	verifyDeployStatus(t, handler, dep1ID, "running")
	verifyDeployStatus(t, handler, dep2ID, "rolled_back")

	// AND: rollback response includes metadata
	rolledBackTo, ok := rollbackResp["rolled_back_to"].(string)
	if !ok || rolledBackTo == "" {
		t.Fatalf("expected rolled_back_to in response, got: %v", rollbackResp)
	}
	rolledBackFrom, ok := rollbackResp["rolled_back_from"].(string)
	if !ok || rolledBackFrom == "" {
		t.Fatalf("expected rolled_back_from in response, got: %v", rollbackResp)
	}
	if rolledBackFrom != dep1ID {
		t.Errorf("rolled_back_from = %q, want %q", rolledBackFrom, dep1ID)
	}
	if rolledBackTo != dep2ID {
		t.Errorf("rolled_back_to = %q, want %q", rolledBackTo, dep2ID)
	}
}

func TestRollbackNoPrevious(t *testing.T) {
	// GIVEN: a site with only one deployment
	dep, handler, database, gitDir := setupDeploy(t)
	_ = dep

	repoID := createTestRepo(t, database, "repo-rollback-noprev", gitDir)

	body := map[string]string{"repo_id": repoID, "site_id": "site-rollback-noprev"}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("deploy expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	depID := created["id"].(string)

	waitForDeploymentTerminal(t, handler, depID, 10*time.Second)
	verifyDeployStatus(t, handler, depID, "running")

	// WHEN: rollback when there's no previous deployment
	rollbackReq := httptest.NewRequest("POST", "/api/deploy/"+depID+"/rollback", nil)
	rollbackW := httptest.NewRecorder()
	handler.ServeHTTP(rollbackW, rollbackReq)

	// THEN: should get 409 Conflict with clear error
	if rollbackW.Code != http.StatusConflict {
		t.Fatalf("expected 409 when no previous deployment, got %d: %s", rollbackW.Code, rollbackW.Body.String())
	}

	var errResp map[string]string
	_ = json.NewDecoder(rollbackW.Body).Decode(&errResp)
	if errResp["error"] == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestRollbackInvalidMethod(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	// GET is not allowed
	req := httptest.NewRequest("GET", "/api/deploy/some-id/rollback", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", w.Code)
	}
}

func TestRollbackNotFound(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	req := httptest.NewRequest("POST", "/api/deploy/nonexistent/rollback", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent deployment, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRollbackSkipsInFlightDeployments is the #304 regression: an in-flight
// deployment (deploying / draining) must never be chosen as the "previous
// successful" rollback target, even when it is the most recent row.
func TestRollbackSkipsInFlightDeployments(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	buildsDir := t.TempDir()

	dep := deploy.New(deploy.Options{
		DB:        database,
		Logger:    logger,
		BuildsDir: buildsDir,
		GitDir:    t.TempDir(),
	})
	gitDir := t.TempDir()
	gitComp := git.New(git.Options{DB: database, Logger: logger, Dir: gitDir})
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	const siteID = "site-rollback-filter"
	insert := func(id, status, createdAt string) {
		t.Helper()
		_, err := database.ExecContext(context.Background(),
			`INSERT INTO deployments (id, repo_id, site_id, status, port, app_type, created_at)
			 VALUES (?, 'repo-rf', ?, ?, 0, 'static', ?)`,
			id, siteID, status, createdAt)
		if err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	insert("dep-current", string(deploy.StateRunning), "2026-01-01T00:00:05Z")
	insert("dep-deploying", string(deploy.StateDeploying), "2026-01-01T00:00:04Z")
	insert("dep-draining", string(deploy.StateDraining), "2026-01-01T00:00:03Z")
	insert("dep-stopped", string(deploy.StateStopped), "2026-01-01T00:00:02Z")
	insert("dep-failed", string(deploy.StateFailed), "2026-01-01T00:00:01Z")

	// The stopped deployment is the only successful candidate and needs
	// on-disk artifacts for rollback to restore it.
	distDir := filepath.Join(buildsDir, "dep-stopped", "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatalf("create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<h1>old</h1>"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	event, err := dep.Rollback(context.Background(), "dep-current")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if event.RolledBackFrom != "dep-stopped" {
		t.Fatalf("rollback target = %q, want dep-stopped (in-flight rows must be skipped)", event.RolledBackFrom)
	}
	verifyDeploymentStatusRow(t, database, "dep-stopped", "running")
	verifyDeploymentStatusRow(t, database, "dep-deploying", "deploying")
	verifyDeploymentStatusRow(t, database, "dep-draining", "draining")
}

func verifyDeploymentStatusRow(t *testing.T, database *db.DB, id, want string) {
	t.Helper()
	var status string
	if err := database.QueryRowContext(context.Background(),
		"SELECT status FROM deployments WHERE id = ?", id).Scan(&status); err != nil {
		t.Fatalf("query status for %s: %v", id, err)
	}
	if status != want {
		t.Fatalf("deployment %s status = %q, want %q", id, status, want)
	}
}

// verifyDeployStatus checks that a deployment has the expected status.
func verifyDeployStatus(t *testing.T, handler http.Handler, depID, expected string) {
	t.Helper()
	getReq := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get deployment %s: expected 200, got %d: %s", depID, getW.Code, getW.Body.String())
	}
	var got map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&got)
	status, _ := got["status"].(string)
	if status != expected {
		t.Fatalf("deployment %s status = %q, want %q", depID, status, expected)
	}
}
