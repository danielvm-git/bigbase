package deploy_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
)

func TestHealthCheckIntegrationPass(t *testing.T) {
	dep, handler, database, gitDir := setupDeploy(t)
	_ = handler

	// Create a repo with a Go HTTP server that returns 200 on /healthz
	repoID := createHealthRepo(t, database, "repo-health-pass", gitDir,
		`package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
`, true)

	ctx := context.Background()
	deployment, err := dep.Trigger(ctx, repoID, "main", "health-site-pass", "site-health-pass", nil, "go", "")
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	waitForDeploymentTerminal(t, handler, deployment.ID, 30*time.Second)

	req := httptest.NewRequest("GET", "/api/deploy/"+deployment.ID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "running" {
		t.Fatalf("expected status 'running', got '%s'", result["status"])
	}
	if url, ok := result["url"]; !ok || url == "" {
		t.Fatalf("expected non-empty url, got: %v", result)
	}
}

func TestHealthCheckIntegrationFail(t *testing.T) {
	dep, handler, database, gitDir := setupDeploy(t)

	// Repo with a Go server that immediately exits (health check fails)
	repoID := createHealthRepo(t, database, "repo-health-fail", gitDir,
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

	ctx := context.Background()
	deployment, err := dep.Trigger(ctx, repoID, "main", "health-site-fail", "site-health-fail", nil, "go", "")
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	waitForDeploymentTerminal(t, handler, deployment.ID, 90*time.Second)

	req := httptest.NewRequest("GET", "/api/deploy/"+deployment.ID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "failed" {
		t.Fatalf("expected status 'failed', got '%s'", result["status"])
	}

	if errMsg, ok := result["error_message"].(string); ok && errMsg != "" {
		if !strings.Contains(errMsg, "Health check") && !strings.Contains(errMsg, "health") {
			t.Logf("error_message = %q, expected health-related message", errMsg)
		}
	}
}

func TestHealthCheckIntegrationDefaults(t *testing.T) {
	// Verify a deployment WITHOUT health_check in manifest uses defaults (path=/)
	dep, handler, database, gitDir := setupDeploy(t)

	repoID := createHealthRepo(t, database, "repo-health-def", gitDir,
		`package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
`, false) // no health_check in manifest

	ctx := context.Background()
	deployment, err := dep.Trigger(ctx, repoID, "main", "health-site-def", "site-health-def", nil, "go", "")
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	waitForDeploymentTerminal(t, handler, deployment.ID, 30*time.Second)

	req := httptest.NewRequest("GET", "/api/deploy/"+deployment.ID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "running" {
		t.Fatalf("expected status 'running', got '%s'", result["status"])
	}
}

func TestHealthCheckReporting(t *testing.T) {
	dep, handler, database, gitDir := setupDeploy(t)

	// Repo with a server that responds on /healthz
	repoID := createHealthRepo(t, database, "repo-health-rpt", gitDir,
		`package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
`, true)

	ctx := context.Background()
	deployment, err := dep.Trigger(ctx, repoID, "main", "health-site-rpt", "site-health-rpt", nil, "go", "")
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	waitForDeploymentTerminal(t, handler, deployment.ID, 30*time.Second)

	// Check that the deploy log contains health probe lines
	req := httptest.NewRequest("GET", "/api/deploy/"+deployment.ID+"/logs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var logResult map[string]any
	_ = json.NewDecoder(w.Body).Decode(&logResult)

	lines, ok := logResult["lines"].([]any)
	if !ok {
		t.Fatal("expected lines array in logs response")
	}

	foundHealthLine := false
	for _, line := range lines {
		lineStr, ok := line.(string)
		if ok && strings.Contains(lineStr, "Health") {
			foundHealthLine = true
			break
		}
	}
	if !foundHealthLine {
		t.Errorf("expected health probe log line, got lines: %v", lines)
	}

	// Check that health_summary is persisted
	var healthSummary string
	err = database.QueryRowContext(ctx,
		"SELECT COALESCE(health_summary,'') FROM deployments WHERE id = ?", deployment.ID).Scan(&healthSummary)
	if err != nil {
		t.Fatalf("query health_summary: %v", err)
	}
	if healthSummary == "" {
		t.Fatal("expected non-empty health_summary in DB")
	}
	if !strings.Contains(healthSummary, "probe_count") {
		t.Errorf("health_summary should contain probe_count: %s", healthSummary)
	}
}

func TestHealthSummaryInStatusAPI(t *testing.T) {
	dep, handler, database, gitDir := setupDeploy(t)

	// Create a healthy app — health_summary should be present in the status response
	repoID := createHealthRepo(t, database, "repo-health-api", gitDir,
		`package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
`, true)

	ctx := context.Background()
	deployment, err := dep.Trigger(ctx, repoID, "main", "health-site-api", "site-health-api", nil, "go", "")
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	waitForDeploymentTerminal(t, handler, deployment.ID, 30*time.Second)

	// GET the deployment detail
	req := httptest.NewRequest("GET", "/api/deploy/"+deployment.ID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	_ = json.NewDecoder(w.Body).Decode(&result)

	hs, ok := result["health_summary"]
	if !ok {
		t.Fatal("expected health_summary in deployment status response")
	}
	hsStr, ok := hs.(string)
	if !ok {
		t.Fatalf("health_summary type = %T, want string, got: %v", hs, hs)
	}
	if hsStr == "" {
		t.Fatal("expected non-empty health_summary string")
	}
	if !strings.Contains(hsStr, "probe_count") {
		t.Errorf("health_summary should contain probe_count, got: %s", hsStr)
	}
}

// createHealthRepo creates a git repo with bigbase.yaml + Go server source.
// If withHealthCheck is true, includes a health_check section in the manifest.
func createHealthRepo(t *testing.T, database *db.DB, repoID, gitDir, serverSource string, withHealthCheck bool) string {
	t.Helper()

	_, err := database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	if err != nil {
		t.Fatalf("create git_repos table: %v", err)
	}

	_, err = database.ExecContext(context.Background(),
		"INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		repoID, "process-test-"+repoID, 0, 1, "main", "process test repo")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	repoPath := filepath.Join(gitDir, repoID+".git")
	mustRun(t, "git", "init", "--bare", "-b", "main", repoPath)
	mustRun(t, "git", "-C", repoPath, "config", "--local", "--add", "safe.directory", "'*'")

	sourceDir := t.TempDir()
	mustRun(t, "git", "init", "-b", "main", sourceDir)
	mustRun(t, "git", "-C", sourceDir, "config", "--local", "--add", "safe.directory", "'*'")
	mustRun(t, "git", "-C", sourceDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", sourceDir, "config", "user.name", "test")

	healthSection := ""
	if withHealthCheck {
		healthSection = `
health_check:
  path: /healthz
  expected_status: 200
  expected_body_contains: ok
  timeout_seconds: 1
  interval_seconds: 1
  max_retries: 2
`
	}

	manifest := fmt.Sprintf(`version: 1
framework: go
build:
  command: go build -o server server.go
start:
  command: ./server
  port: 8080%s`, healthSection)

	if err := os.WriteFile(filepath.Join(sourceDir, "bigbase.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write bigbase.yaml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(sourceDir, "server.go"), []byte(serverSource), 0644); err != nil {
		t.Fatalf("write server.go: %v", err)
	}

	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "initial commit")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")

	return repoID
}
