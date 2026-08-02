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

// migrateEnvVarsTable creates site_env_vars in a test DB (normally done by sites.Start).
func migrateEnvVarsTable(t *testing.T, database *db.DB) {
	t.Helper()
	_, err := database.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS site_env_vars (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value_encrypted TEXT NOT NULL DEFAULT '',
		is_build_time INTEGER NOT NULL DEFAULT 0,
		is_runtime INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(site_id, key)
	)`)
	if err != nil {
		t.Fatalf("migrate site_env_vars: %v", err)
	}
}

func insertEnvVar(t *testing.T, database *db.DB, siteID, key, value string, buildTime, runtime bool) {
	t.Helper()
	bt := 0
	if buildTime {
		bt = 1
	}
	rt := 0
	if runtime {
		rt = 1
	}
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO site_env_vars (id, site_id, key, value_encrypted, is_build_time, is_runtime)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("%s-%s", siteID, key), siteID, key, value, bt, rt)
	if err != nil {
		t.Fatalf("insert env var %s: %v", key, err)
	}
}

func TestEnvVarInjection(t *testing.T) {
	t.Run("FetchSiteEnvVars_returns_build_time_vars", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		dep := deploy.New(deploy.Options{DB: database, Logger: logger})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		migrateEnvVarsTable(t, database)
		insertEnvVar(t, database, "site-1", "BUILD_VAR", "build-value", true, false)
		insertEnvVar(t, database, "site-1", "RUNTIME_VAR", "runtime-value", false, true)
		insertEnvVar(t, database, "site-1", "BOTH_VAR", "both-value", true, true)

		vars, err := dep.FetchSiteEnvVars(context.Background(), "site-1", true)
		if err != nil {
			t.Fatalf("FetchSiteEnvVars: %v", err)
		}
		if len(vars) != 2 {
			t.Fatalf("expected 2 build-time vars, got %d: %v", len(vars), vars)
		}
		found := make(map[string]bool)
		for _, v := range vars {
			found[v] = true
		}
		if !found["BUILD_VAR=build-value"] {
			t.Fatalf("expected BUILD_VAR=build-value, got %v", vars)
		}
		if !found["BOTH_VAR=both-value"] {
			t.Fatalf("expected BOTH_VAR=both-value, got %v", vars)
		}
	})

	t.Run("FetchSiteEnvVars_returns_runtime_vars", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		dep := deploy.New(deploy.Options{DB: database, Logger: logger})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		migrateEnvVarsTable(t, database)
		insertEnvVar(t, database, "site-2", "BUILD_ONLY", "b", true, false)
		insertEnvVar(t, database, "site-2", "RUNTIME_ONLY", "r", false, true)

		vars, err := dep.FetchSiteEnvVars(context.Background(), "site-2", false)
		if err != nil {
			t.Fatalf("FetchSiteEnvVars: %v", err)
		}
		if len(vars) != 1 {
			t.Fatalf("expected 1 runtime var, got %d: %v", len(vars), vars)
		}
		if vars[0] != "RUNTIME_ONLY=r" {
			t.Fatalf("expected RUNTIME_ONLY=r, got %v", vars[0])
		}
	})

	t.Run("FetchSiteEnvVars_empty_site_returns_empty", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		dep := deploy.New(deploy.Options{DB: database, Logger: logger})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		migrateEnvVarsTable(t, database)

		vars, err := dep.FetchSiteEnvVars(context.Background(), "no-such-site", true)
		if err != nil {
			t.Fatalf("FetchSiteEnvVars: %v", err)
		}
		if len(vars) != 0 {
			t.Fatalf("expected 0 vars for unknown site, got %v", vars)
		}
	})

	t.Run("FetchSiteEnvVars_with_encryption_key", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		// 32-byte key = 64 hex chars
		encKey := strings.Repeat("b2", 32)
		dep := deploy.New(deploy.Options{DB: database, Logger: logger, EnvEncryptionKey: encKey})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		migrateEnvVarsTable(t, database)

		// Store ciphertext manually: just store "raw-value" (no encryption) for this unit test,
		// since FetchSiteEnvVars decrypts using the key. We verify the decryption path works
		// by storing an actual encrypted value via EncryptForTest helper.
		encrypted, err := dep.EncryptEnvValue("secret-value")
		if err != nil {
			t.Fatalf("EncryptEnvValue: %v", err)
		}
		_, err = database.ExecContext(context.Background(),
			`INSERT INTO site_env_vars (id, site_id, key, value_encrypted, is_build_time, is_runtime)
			 VALUES ('ev-1', 'site-enc', 'SECRET_KEY', ?, 1, 0)`, encrypted)
		if err != nil {
			t.Fatalf("insert encrypted env var: %v", err)
		}

		vars, err := dep.FetchSiteEnvVars(context.Background(), "site-enc", true)
		if err != nil {
			t.Fatalf("FetchSiteEnvVars: %v", err)
		}
		if len(vars) != 1 || vars[0] != "SECRET_KEY=secret-value" {
			t.Fatalf("expected SECRET_KEY=secret-value, got %v", vars)
		}
	})

	t.Run("build_injects_env_var_via_shell_script", func(t *testing.T) {
		if _, err := exec.LookPath("npm"); err != nil {
			t.Skip("npm not in PATH")
		}

		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		gitDir := t.TempDir()
		buildsDir := t.TempDir()
		dep := deploy.New(deploy.Options{
			DB:        database,
			Logger:    logger,
			BuildsDir: buildsDir,
			GitDir:    gitDir,
		})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		// Set up a Node app whose build script writes BUILD_INJECTED to a file.
		repoID := createEnvVarBuildRepo(t, database, gitDir)
		migrateEnvVarsTable(t, database)
		insertEnvVar(t, database, "site-build-inj", "BUILD_INJECTED", "injected-build-value", true, false)

		deployment, err := dep.Trigger(context.Background(), repoID, "main", "env-test-app", "site-build-inj", nil, "", "")
		if err != nil {
			t.Fatalf("trigger: %v", err)
		}

		// Wait for build to complete (success or failure).
		waitForDeploymentTerminal(t, dep.Handler(), deployment.ID, 60*time.Second)

		// The build script writes BUILD_INJECTED to injected.txt in the build dir.
		injectedFile := filepath.Join(buildsDir, deployment.ID, "injected.txt")
		data, err := os.ReadFile(injectedFile)
		if err != nil {
			t.Fatalf("read injected.txt: %v — env var was not injected during build", err)
		}
		got := strings.TrimSpace(string(data))
		if got != "injected-build-value" {
			t.Fatalf("expected injected.txt to contain 'injected-build-value', got %q", got)
		}
	})

	t.Run("runtime_injects_env_var_into_process", func(t *testing.T) {
		if _, err := exec.LookPath("npm"); err != nil {
			t.Skip("npm not in PATH")
		}

		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		gitDir := t.TempDir()
		buildsDir := t.TempDir()
		dep := deploy.New(deploy.Options{
			DB:        database,
			Logger:    logger,
			BuildsDir: buildsDir,
			GitDir:    gitDir,
			BasePort:  12900,
		})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		repoID := createEnvVarRuntimeRepo(t, database, gitDir)
		migrateEnvVarsTable(t, database)
		insertEnvVar(t, database, "site-rt-inj", "RUNTIME_GREETING", "hello-from-env", false, true)

		deployment, err := dep.Trigger(context.Background(), repoID, "main", "env-rt-app", "site-rt-inj", nil, "", "")
		if err != nil {
			t.Fatalf("trigger: %v", err)
		}

		// Wait for running.
		start := time.Now()
		for time.Since(start) < 30*time.Second {
			var status string
			_ = database.QueryRowContext(context.Background(),
				"SELECT status FROM deployments WHERE id = ?", deployment.ID).Scan(&status)
			if status == "running" {
				break
			}
			if status == "failed" {
				t.Fatal("deployment failed during runtime injection test")
			}
			time.Sleep(20 * time.Millisecond)
		}
		time.Sleep(50 * time.Millisecond) // allow server to bind

		// Hit the app and verify it echoes the injected env var.
		var body string
		for i := 0; i < 10; i++ {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", deployment.Port))
			if err != nil {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			body = strings.TrimSpace(string(b))
			if body != "" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if body != "hello-from-env" {
			t.Fatalf("expected response 'hello-from-env', got %q", body)
		}
	})
}

// createEnvVarBuildRepo creates a Node.js repo whose build script writes BUILD_INJECTED to injected.txt.
func createEnvVarBuildRepo(t *testing.T, database *db.DB, gitDir string) string {
	t.Helper()
	repoID := "repo-env-build"
	_, _ = database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	_, err := database.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		repoID, "env-build-repo", 0, 1, "main", "")
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

	// Build script writes BUILD_INJECTED to injected.txt; static output goes to dist/
	pkg := `{
		"name":"env-build-test","private":true,
		"scripts":{"build":"node -e \"const fs=require('fs');fs.writeFileSync('injected.txt',process.env.BUILD_INJECTED||'NOT_INJECTED');fs.mkdirSync('dist',{recursive:true});fs.writeFileSync('dist/index.html','<h1>done</h1>')\""}
	}`
	if err := os.WriteFile(filepath.Join(sourceDir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "initial")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")
	return repoID
}

// createEnvVarRuntimeRepo creates a Node.js app that responds with its RUNTIME_GREETING env var.
func createEnvVarRuntimeRepo(t *testing.T, database *db.DB, gitDir string) string {
	t.Helper()
	repoID := "repo-env-runtime"
	_, _ = database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	_, err := database.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		repoID, "env-runtime-repo", 0, 1, "main", "")
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

	// Node.js HTTP server that responds with RUNTIME_GREETING
	server := `
const http = require('http');
const msg = process.env.RUNTIME_GREETING || 'NOT_INJECTED';
const port = parseInt(process.env.PORT || '3000');
http.createServer((req, res) => { res.end(msg); }).listen(port);
`
	if err := os.WriteFile(filepath.Join(sourceDir, "index.js"), []byte(server), 0644); err != nil {
		t.Fatalf("write index.js: %v", err)
	}
	pkg := `{"name":"env-runtime-test","scripts":{"build":"true","start":"node index.js"}}`
	if err := os.WriteFile(filepath.Join(sourceDir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "initial")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")
	return repoID
}
