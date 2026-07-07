package deploy_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

func TestNativeDBEnv_sqlite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "test.db")
	got := deploy.NativeDBEnv("sqlite", dbFile)
	if len(got) != 1 {
		t.Fatalf("expected 1 env var, got %v", got)
	}
	if !strings.HasPrefix(got[0], "DB_PATH=") {
		t.Fatalf("expected DB_PATH prefix, got %q", got[0])
	}
	abs, err := filepath.Abs(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "DB_PATH="+abs {
		t.Fatalf("expected absolute DB_PATH, got %q want %q", got[0], "DB_PATH="+abs)
	}
}

func TestNativeDBEnv_postgres(t *testing.T) {
	t.Parallel()
	dsn := "postgres://user:pass@localhost:5432/mydb"
	got := deploy.NativeDBEnv("postgres", dsn)
	if len(got) != 1 || got[0] != "DATABASE_URL="+dsn {
		t.Fatalf("got %v, want DATABASE_URL=%s", got, dsn)
	}
}

func TestNativeDBEnv_emptyDSN(t *testing.T) {
	t.Parallel()
	if got := deploy.NativeDBEnv("sqlite", ""); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestNativeDBEnv_memorySQLite(t *testing.T) {
	t.Parallel()
	got := deploy.NativeDBEnv("", ":memory:")
	if len(got) != 1 || got[0] != "DB_PATH=:memory:" {
		t.Fatalf("got %v", got)
	}
}

func TestRuntimeInjectsNativeDBEnv_sqlite(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}

	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	defer func() { _ = database.Stop(&kernel.Context{}) }()

	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "platform.db")

	dep := deploy.New(deploy.Options{
		DB:        database,
		Logger:    logger,
		BuildsDir: buildsDir,
		GitDir:    gitDir,
		BasePort:  13000,
		DBDriver:  "sqlite",
		DBDSN:     dbPath,
	})
	_ = dep.Start(&kernel.Context{})
	defer func() { _ = dep.Stop(&kernel.Context{}) }()

	repoID := createNativeDBRuntimeRepo(t, database, gitDir)
	deployment, err := dep.Trigger(context.Background(), repoID, "main", "native-db-app", "site-native-db", nil, "", "")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	waitForDeploymentRunning(t, database, deployment.ID, 30*time.Second)

	var body string
	for i := 0; i < 10; i++ {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", deployment.Port))
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		body = strings.TrimSpace(string(b))
		if body != "" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if body != absPath {
		t.Fatalf("expected app to echo DB_PATH %q, got %q", absPath, body)
	}
}

func createNativeDBRuntimeRepo(t *testing.T, database *db.DB, gitDir string) string {
	t.Helper()
	repoID := "repo-native-db"
	_, _ = database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	_, err := database.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		repoID, "native-db-repo", 0, 1, "main", "")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	repoPath := filepath.Join(gitDir, repoID+".git")
	mustRun(t, "git", "init", "--bare", "-b", "main", repoPath)
	mustRun(t, "git", "config", "--global", "--add", "safe.directory", repoPath)

	sourceDir := t.TempDir()
	mustRun(t, "git", "init", "-b", "main", sourceDir)
	mustRun(t, "git", "config", "--global", "--add", "safe.directory", sourceDir)
	mustRun(t, "git", "-C", sourceDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", sourceDir, "config", "user.name", "test")

	server := `
const http = require('http');
const msg = process.env.DB_PATH || process.env.DATABASE_URL || 'NOT_INJECTED';
const port = parseInt(process.env.PORT || '3000');
http.createServer((req, res) => { res.end(msg); }).listen(port);
`
	if err := os.WriteFile(filepath.Join(sourceDir, "index.js"), []byte(server), 0644); err != nil {
		t.Fatalf("write index.js: %v", err)
	}
	pkg := `{"name":"native-db-test","scripts":{"build":"true","start":"node index.js"}}`
	if err := os.WriteFile(filepath.Join(sourceDir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "initial")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")
	return repoID
}

func waitForDeploymentRunning(t *testing.T, database *db.DB, deployID string, timeout time.Duration) {
	t.Helper()
	start := time.Now()
	for time.Since(start) < timeout {
		var status string
		_ = database.QueryRowContext(context.Background(),
			"SELECT status FROM deployments WHERE id = ?", deployID).Scan(&status)
		if status == "running" {
			time.Sleep(1 * time.Second)
			return
		}
		if status == "failed" {
			t.Fatal("deployment failed during native DB env injection test")
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("deployment %s did not reach running within %s", deployID, timeout)
}
