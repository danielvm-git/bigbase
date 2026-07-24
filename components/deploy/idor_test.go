package deploy_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
)

// setupIDORTest creates the sites table with org_id and inserts test data for two orgs.
func setupIDORTest(t *testing.T, database *db.DB) {
	t.Helper()
	_, err := database.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		org_id INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("sites table: %v", err)
	}
	_, err = database.Exec(`INSERT INTO sites (id, name, git_repo_id, org_id) VALUES
		('site-org-a', 'Site A', 'repo-a', 100),
		('site-org-b', 'Site B', 'repo-b', 200)
		ON CONFLICT(id) DO UPDATE SET org_id=excluded.org_id`)
	if err != nil {
		t.Fatalf("insert sites: %v", err)
	}
}

// insertTestDeployment inserts a deployment with the given site_id.
func insertTestDeployment(t *testing.T, database *db.DB, depID, siteID, status string) {
	t.Helper()
	_, err := database.Exec(
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, created_at)
		 VALUES (?, ?, ?, 'main', ?, datetime('now'))`,
		depID, "repo-"+depID, siteID, status)
	if err != nil {
		t.Fatalf("insert deployment %s: %v", depID, err)
	}
}

// requestWithOrg creates a request with the given org_id in context.
func requestWithOrg(method, path string, body any, orgID int64) *http.Request {
	var buf *bytes.Buffer
	if body != nil {
		buf = new(bytes.Buffer)
		_ = json.NewEncoder(buf).Encode(body)
	}
	var req *http.Request
	if buf != nil {
		req = httptest.NewRequest(method, path, buf)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	return req.WithContext(auth.WithOrgID(req.Context(), orgID))
}

// TestIDOR_ListScopedByOrg verifies HandleList only returns deployments for the caller's org.
func TestIDOR_ListScopedByOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)

	// Insert deployments for two different orgs
	insertTestDeployment(t, database, "dep-a1", "site-org-a", "running")
	insertTestDeployment(t, database, "dep-b1", "site-org-b", "running")

	// Org A user should only see dep-a1
	req := requestWithOrg("GET", "/api/deploy", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got: %v", body)
	}

	// Should only contain dep-a1, not dep-b1
	if len(data) != 1 {
		t.Fatalf("expected 1 deployment for org 100, got %d: %v", len(data), body)
	}
	dep0 := data[0].(map[string]any)
	if dep0["id"] != "dep-a1" {
		t.Fatalf("expected dep-a1, got %v", dep0["id"])
	}
}

// TestIDOR_GetByIDPreventsCrossOrg verifies handleDeployByID returns 403 for other orgs.
func TestIDOR_GetByIDPreventsCrossOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-b1", "site-org-b", "running")

	// Org A user tries to read org B's deployment
	req := requestWithOrg("GET", "/api/deploy/dep-b1", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_GetByIDAllowsSameOrg verifies handleDeployByID returns 200 for same org.
func TestIDOR_GetByIDAllowsSameOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-a1", "site-org-a", "running")

	// Org A user reads their own deployment
	req := requestWithOrg("GET", "/api/deploy/dep-a1", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_DeletePreventsCrossOrg verifies handleDeleteDeployment returns 403 for other orgs.
func TestIDOR_DeletePreventsCrossOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-b1", "site-org-b", "running")

	// Org A user tries to delete org B's deployment
	req := requestWithOrg("DELETE", "/api/deploy/dep-b1", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_DeleteAllowsSameOrg verifies handleDeleteDeployment returns 204 for same org.
func TestIDOR_DeleteAllowsSameOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-a1", "site-org-a", "replaced")

	// Org A user deletes their own deployment
	req := requestWithOrg("DELETE", "/api/deploy/dep-a1", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_LogsPreventsCrossOrg verifies handleDeployLogs returns 403 for other orgs.
func TestIDOR_LogsPreventsCrossOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-b1", "site-org-b", "running")

	// Org A user tries to read logs of org B's deployment
	req := requestWithOrg("GET", "/api/deploy/dep-b1/logs", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_LogsAllowsSameOrg verifies handleDeployLogs returns 200 for same org.
func TestIDOR_LogsAllowsSameOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-a1", "site-org-a", "running")

	// Org A user reads their own deployment logs
	req := requestWithOrg("GET", "/api/deploy/dep-a1/logs", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_RollbackPreventsCrossOrg verifies handleRollback returns 403 for other orgs.
func TestIDOR_RollbackPreventsCrossOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-b1", "site-org-b", "running")

	// Org A user tries to rollback org B's deployment
	req := requestWithOrg("POST", "/api/deploy/dep-b1/rollback", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_RollbackEventsPreventsCrossOrg verifies handleRollbackEvents returns 403 for other orgs.
func TestIDOR_RollbackEventsPreventsCrossOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)

	// Org A user tries to read rollback events for org B's site
	req := requestWithOrg("GET", "/api/deploy/site-org-b/rollback-events", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_NoOrgIDSkipsOwnershipCheck verifies that requests without org_id
// (e.g., site deploy key auth) still work without org filtering.
func TestIDOR_NoOrgIDSkipsOwnershipCheck(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-a1", "site-org-a", "running")

	// Request without org_id in context (site deploy key auth)
	req := httptest.NewRequest("GET", "/api/deploy/dep-a1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should succeed — no org_id means no org filtering
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIDOR_MultipleDeploymentsSameOrg verifies all deployments for same org are returned.
func TestIDOR_MultipleDeploymentsSameOrg(t *testing.T) {
	dep, handler, database, _ := setupDeploy(t)
	_ = dep
	setupIDORTest(t, database)
	insertTestDeployment(t, database, "dep-a1", "site-org-a", "running")
	insertTestDeployment(t, database, "dep-a2", "site-org-a", "replaced")
	insertTestDeployment(t, database, "dep-b1", "site-org-b", "running")

	// Org A user should see both their deployments
	req := requestWithOrg("GET", "/api/deploy", nil, 100)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got: %v", body)
	}

	if len(data) != 2 {
		t.Fatalf("expected 2 deployments for org 100, got %d: %v", len(data), body)
	}
}


