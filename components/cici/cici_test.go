package cici_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/cici"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupCICI(t *testing.T) *cici.CICI {
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
	t.Cleanup(func() { _ = k.Stop() })
	return c
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

func saveWorkflow(t *testing.T, h http.Handler, repoID, name, yaml string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"repo_id": repoID, "name": name, "yaml": yaml,
	})
	req := httptest.NewRequest("PUT", "/api/cici/"+repoID+"/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
	c := setupCICI(t)
	h := c.Handler()

	workflowYAML := `name: Test
on: [push]
jobs:
  test:
    runs-on: self-hosted
    steps:
      - run: echo "hello"
`
	saveWorkflow(t, h, "repo1", "Test", workflowYAML)

	getReq := httptest.NewRequest("GET", "/api/cici/repo1/workflows", nil)
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
	c := setupCICI(t)
	h := c.Handler()

	wfID := saveWorkflow(t, h, "r", "CI", `name: CI
on: [push]
jobs:
  build:
    steps:
      - run: echo "ok"
`)
	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/r/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
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
	c := setupCICI(t)
	h := c.Handler()

	wfID := saveWorkflow(t, h, "r", "CI", `name: CI
on: [push]
jobs:
  build:
    steps:
      - run: echo "hello world"
`)
	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/r/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	var run map[string]any
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	runID, _ := run["id"].(string)

	pollForStatus(t, h, runID, "success")

	logReq := httptest.NewRequest("GET", "/api/cici/runs/"+runID+"/logs", nil)
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
	c := setupCICI(t)
	h := c.Handler()

	body, _ := json.Marshal(map[string]string{"repo_id": "r", "name": "", "yaml": "name: Test\non: [push]\njobs:\n  t:\n    steps:\n      - run: echo\n"})
	req := httptest.NewRequest("PUT", "/api/cici/r/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICISaveWorkflowInvalidYAML(t *testing.T) {
	c := setupCICI(t)
	h := c.Handler()

	body, _ := json.Marshal(map[string]string{"repo_id": "r", "name": "Bad", "yaml": "<<<<"})
	req := httptest.NewRequest("PUT", "/api/cici/r/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICITriggerRunNotFound(t *testing.T) {
	c := setupCICI(t)
	h := c.Handler()

	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	req := httptest.NewRequest("POST", "/api/cici/r/workflows/nonexistent/run", bytes.NewReader(runBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICIRunsMethodNotAllowed(t *testing.T) {
	c := setupCICI(t)
	h := c.Handler()

	req := httptest.NewRequest("DELETE", "/api/cici/runs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICIWorkflowsMethodNotAllowed(t *testing.T) {
	c := setupCICI(t)
	h := c.Handler()

	req := httptest.NewRequest("DELETE", "/api/cici/r/workflows", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICIEmptyRepoWorkflows(t *testing.T) {
	c := setupCICI(t)
	h := c.Handler()

	req := httptest.NewRequest("GET", "/api/cici/empty/workflows", nil)
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
