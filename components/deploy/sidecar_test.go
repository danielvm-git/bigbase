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
)

func TestStaticSidecar_BuildAndStart(t *testing.T) {

	dep, handler, database, gitDir := setupDeploy(t)
	defer func() { _ = dep.Stop(nil) }() // clean up apps

	// Create a repo
	repoID := createTestRepo(t, database, "repo-sidecar", gitDir)

	// Since createTestRepo pushes index.html, we need to clone it, add bigbase.toml, commit and push
	repoPath := filepath.Join(gitDir, repoID+".git")
	sourceDir := t.TempDir()
	mustRun(t, "git", "clone", repoPath, sourceDir)
	mustRun(t, "git", "-C", sourceDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", sourceDir, "config", "user.name", "test")

	// Create bigbase.toml for static-sidecar
	tomlContent := `
version = 1
framework = "static-sidecar"
[build]
command = "echo 'sidecar build executed'"
[start]
command = "echo 'sidecar start executed'"
port = 8080
[health_check]
timeout_seconds = 1
interval_seconds = 1
max_retries = 2
`
	if err := os.WriteFile(filepath.Join(sourceDir, "bigbase.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	mustRun(t, "git", "-C", sourceDir, "add", "bigbase.toml")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "add bigbase.toml")
	mustRun(t, "git", "-C", sourceDir, "push", "origin", "main")

	// Trigger deploy
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	deployID := created["id"].(string)

	// Wait for deploy to finish (success or fail)
	deadline := time.Now().Add(10 * time.Second)
	var status string
	for time.Now().Before(deadline) {
		err := database.QueryRowContext(context.Background(), "SELECT status FROM deployments WHERE id = ?", deployID).Scan(&status)
		if err != nil {
			t.Fatal(err)
		}
		if status == "running" || status == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if status != "failed" && status != "running" {
		t.Fatalf("deploy timed out, status: %s", status)
	}

	var buildLog string
	err := database.QueryRowContext(context.Background(), "SELECT build_log FROM deployments WHERE id = ?", deployID).Scan(&buildLog)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains([]byte(buildLog), []byte("sidecar build executed")) {
		t.Errorf("expected build log to contain 'sidecar build executed', got: %s", buildLog)
	}

	// Since start command is `echo`, it will exit immediately, causing health probe to fail
	// or process to die, resulting in status = "failed"
	if status != "failed" {
		t.Fatalf("expected sidecar deploy to fail since it exits immediately, got %s", status)
	}

	var errMsg string
	err = database.QueryRowContext(context.Background(), "SELECT error_message FROM deployments WHERE id = ?", deployID).Scan(&errMsg)
	if err != nil {
		t.Fatal(err)
	}

	if errMsg == "" {
		t.Errorf("expected error message about start failure, got none")
	}
}


