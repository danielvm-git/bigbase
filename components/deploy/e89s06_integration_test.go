package deploy_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// deployStack is the full e89 composition used by the s06 integration tests:
// db + projects + secrets + deploy, with the legacy sites/site_env_vars
// tables available for dual-read scenarios.
type deployStack struct {
	database  *db.DB
	s         *secrets.Secrets
	p         *projects.Projects
	gitDir    string
	buildsDir string
	dep       *deploy.Deploy
	handler   http.Handler
}

func newDeployStack(t *testing.T, publicDomain string) *deployStack {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	if err := d.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = d.Stop(&kernel.Context{}) })

	p := projects.New(projects.Options{DB: d, Logger: logger})
	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("projects start: %v", err)
	}
	raw := base64.StdEncoding.EncodeToString(bytesRepeat42())
	rootKey, err := secrets.ParseRootKey(raw)
	if err != nil {
		t.Fatalf("parse root key: %v", err)
	}
	s, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: rootKey})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	if err := s.Start(&kernel.Context{}); err != nil {
		t.Fatalf("secrets start: %v", err)
	}

	// Legacy tables after component Start (projects.Start tolerates their
	// absence); the sites table carries the project_id attachment column.
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL DEFAULT '',
		org_id INTEGER NOT NULL DEFAULT 0,
		project_id TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("create sites table: %v", err)
	}
	migrateEnvVarsTable(t, d)

	gitDir := t.TempDir()
	buildsDir := t.TempDir()
	dep := deploy.New(deploy.Options{
		DB:           d,
		Logger:       logger,
		BuildsDir:    buildsDir,
		GitDir:       gitDir,
		Secrets:      s,
		PublicDomain: publicDomain,
		BasePort:     13000,
	})
	if err := dep.Start(&kernel.Context{}); err != nil {
		t.Fatalf("deploy start: %v", err)
	}
	t.Cleanup(func() { _ = dep.Stop(&kernel.Context{}) })

	return &deployStack{database: d, s: s, p: p, gitDir: gitDir, buildsDir: buildsDir, dep: dep, handler: dep.Handler()}
}

func bytesRepeat42() []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = 0x42
	}
	return out
}

// createSiteRow inserts a site row into the legacy sites table.
func createSiteRow(t *testing.T, database *db.DB, siteID string, orgID int64) {
	t.Helper()
	if _, err := database.ExecContext(context.Background(),
		`INSERT OR IGNORE INTO sites (id, name, git_repo_id, org_id) VALUES (?, ?, '', ?)`,
		siteID, siteID, orgID); err != nil {
		t.Fatalf("insert site %s: %v", siteID, err)
	}
}

// attachProjectSecret attaches one compatibility project + production
// environment to the site and stores a native Project secret in it.
func attachProjectSecret(t *testing.T, st *deployStack, siteID string, orgID int64, key, value string) {
	t.Helper()
	ctx := auth.WithOrgID(context.Background(), orgID)
	createSiteRow(t, st.database, siteID, orgID)
	projectID, err := st.p.EnsureSiteProject(ctx, siteID, siteID, orgID)
	if err != nil {
		t.Fatalf("attach site %s: %v", siteID, err)
	}
	var envID string
	if err := st.database.QueryRowContext(ctx,
		`SELECT id FROM project_environments WHERE project_id = ? AND slug = 'production'`, projectID).
		Scan(&envID); err != nil {
		t.Fatalf("find production environment: %v", err)
	}
	folder, err := st.s.EnsureFolder(ctx, projectID, envID, "default")
	if err != nil {
		t.Fatalf("ensure default folder: %v", err)
	}
	if _, err := st.s.CreateSecret(ctx, projectID, envID, folder.ID, key, value); err != nil {
		t.Fatalf("create project secret %s: %v", key, err)
	}
}

// createEnvEchoRepo creates a Node app whose HTTP response echoes a fixed set
// of env keys as JSON. When printEnv is true it also prints the echoed map to
// stdout at startup (the realistic tool-echo leak vector for redaction tests).
func createEnvEchoRepo(t *testing.T, database *db.DB, gitDir, repoID string, printEnv bool) string {
	t.Helper()
	if _, err := database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`); err != nil {
		t.Fatalf("create git_repos table: %v", err)
	}
	if _, err := database.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		repoID, repoID, 0, 1, "main", ""); err != nil {
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

	echoKeys := `['DB_URL','API_TOKEN','BUILD_ONLY','RUNTIME_ONLY','LEGACY_ONLY','PROJECT_ONLY']`
	server := `const http = require('http');
const port = parseInt(process.env.PORT || '3000');
const keys = ` + echoKeys + `;
const out = {};
for (const k of keys) out[k] = process.env[k] || null;
`
	if printEnv {
		server += `console.log('ENV_ECHO ' + JSON.stringify({DB_URL: process.env.DB_URL, API_TOKEN: process.env.API_TOKEN}));
`
	}
	server += `http.createServer((req, res) => { res.setHeader('Content-Type','application/json'); res.end(JSON.stringify(out)); }).listen(port);
`
	if err := os.WriteFile(filepath.Join(sourceDir, "index.js"), []byte(server), 0644); err != nil {
		t.Fatalf("write index.js: %v", err)
	}
	pkg := `{"name":"env-echo","private":true,"scripts":{"build":"true","start":"node index.js"}}`
	if err := os.WriteFile(filepath.Join(sourceDir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "initial")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")
	return repoID
}

// httpGet polls the app on port until it returns a non-empty body.
func httpGet(t *testing.T, port int, timeout time.Duration) string {
	t.Helper()
	url := fmt.Sprintf("http://localhost:%d/", port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			b, rerr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if rerr == nil && len(b) > 0 {
				return strings.TrimSpace(string(b))
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return ""
}

// deploymentLogs returns the captured deploy log lines for a deployment.
func deploymentLogs(t *testing.T, handler http.Handler, depID string) []string {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/deploy/"+depID+"/logs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logs: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	raw, _ := result["lines"].([]any)
	var lines []string
	for _, l := range raw {
		if s, ok := l.(string); ok {
			lines = append(lines, s)
		}
	}
	return lines
}

func TestRuntimeProjectSecretInjected(t *testing.T) {
	// SC-e89s06-P0-01: Site compatibility overrides Project on collision, and
	// the project-only secret still reaches the runtime child process.
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}
	st := newDeployStack(t, "test.example.com")
	repoID := createEnvEchoRepo(t, st.database, st.gitDir, "repo-rt-project", false)
	attachProjectSecret(t, st, "site-rt-project", 1, "DB_URL", "project-db-value")
	attachProjectSecret(t, st, "site-rt-project", 1, "API_TOKEN", "project-token-1")
	// Legacy Site compatibility value for the same key (dual-read).
	insertEnvVar(t, st.database, "site-rt-project", "DB_URL", "site-db-value", true, true)

	deployment, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-project-app", "site-rt-project", nil, "", "")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deployment.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deployment.ID, "running")

	body := httpGet(t, deployment.Port, 30*time.Second)
	if body == "" {
		t.Fatal("app never responded")
	}
	var echo map[string]any
	if err := json.Unmarshal([]byte(body), &echo); err != nil {
		t.Fatalf("decode echo %q: %v", body, err)
	}
	if echo["DB_URL"] != "site-db-value" {
		t.Fatalf("site value must override project value at runtime, got %v", echo["DB_URL"])
	}
	if echo["API_TOKEN"] != "project-token-1" {
		t.Fatalf("project secret missing at runtime, got %v", echo["API_TOKEN"])
	}
}

func TestRuntimeBuildOnlyExcluded(t *testing.T) {
	// SC-e89s06-P0-02: build-only legacy values are excluded from the runtime
	// environment while native Project values reach the child process.
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}
	st := newDeployStack(t, "test.example.com")
	repoID := createEnvEchoRepo(t, st.database, st.gitDir, "repo-rt-scope", false)
	attachProjectSecret(t, st, "site-rt-scope", 1, "PROJECT_ONLY", "native-value")
	insertEnvVar(t, st.database, "site-rt-scope", "BUILD_ONLY", "build-secret", true, false)
	insertEnvVar(t, st.database, "site-rt-scope", "RUNTIME_ONLY", "runtime-secret", false, true)

	deployment, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-scope-app", "site-rt-scope", nil, "", "")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deployment.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deployment.ID, "running")

	body := httpGet(t, deployment.Port, 30*time.Second)
	var echo map[string]any
	if err := json.Unmarshal([]byte(body), &echo); err != nil {
		t.Fatalf("decode echo %q: %v", body, err)
	}
	if v, ok := echo["BUILD_ONLY"]; ok && v != nil {
		t.Fatalf("build-only secret leaked into runtime: %v", echo["BUILD_ONLY"])
	}
	if echo["RUNTIME_ONLY"] != "runtime-secret" {
		t.Fatalf("runtime-only secret missing: %v", echo["RUNTIME_ONLY"])
	}
	if echo["PROJECT_ONLY"] != "native-value" {
		t.Fatalf("project secret missing at runtime: %v", echo["PROJECT_ONLY"])
	}
}

func TestRuntimeLogsRedactResolvedValues(t *testing.T) {
	// SC-e89s06-P0-03: a child process that echoes its own environment must not
	// leak resolved Project or Site values into the captured deploy log.
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}
	st := newDeployStack(t, "test.example.com")
	repoID := createEnvEchoRepo(t, st.database, st.gitDir, "repo-rt-redact", true)
	attachProjectSecret(t, st, "site-rt-redact", 1, "DB_URL", "redact-project-db-9f3a")
	attachProjectSecret(t, st, "site-rt-redact", 1, "API_TOKEN", "redact-project-token")
	insertEnvVar(t, st.database, "site-rt-redact", "API_TOKEN", "redact-site-token-77", false, true)

	deployment, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-redact-app", "site-rt-redact", nil, "", "")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deployment.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deployment.ID, "running")

	// Give the runtime log scanner a moment to capture the startup echo.
	deadline := time.Now().Add(10 * time.Second)
	var joined string
	for time.Now().Before(deadline) {
		joined = strings.Join(deploymentLogs(t, st.handler, deployment.ID), "\n")
		if strings.Contains(joined, "ENV_ECHO") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(joined, "ENV_ECHO") {
		t.Fatalf("expected runtime env echo in logs, got:\n%s", joined)
	}
	for _, secret := range []string{"redact-project-db-9f3a", "redact-project-token", "redact-site-token-77"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("plaintext %q leaked into deploy logs:\n%s", secret, joined)
		}
	}
}

func TestRuntimeLegacyOnlyDualRead(t *testing.T) {
	// SC-e89s06-P0-04: a site with only legacy values stays deployable through
	// the compatibility layer during native-first dual-read.
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}
	st := newDeployStack(t, "test.example.com")
	repoID := createEnvEchoRepo(t, st.database, st.gitDir, "repo-rt-legacy", false)
	// No project attachment — legacy-only site.
	createSiteRow(t, st.database, "site-rt-legacy", 1)
	insertEnvVar(t, st.database, "site-rt-legacy", "LEGACY_ONLY", "legacy-deployable", false, true)

	deployment, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-legacy-app", "site-rt-legacy", nil, "", "")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deployment.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deployment.ID, "running")

	body := httpGet(t, deployment.Port, 30*time.Second)
	var echo map[string]any
	if err := json.Unmarshal([]byte(body), &echo); err != nil {
		t.Fatalf("decode echo %q: %v", body, err)
	}
	if echo["LEGACY_ONLY"] != "legacy-deployable" {
		t.Fatalf("legacy value missing at runtime: %v", echo["LEGACY_ONLY"])
	}
}

func TestResumePreservesResolvedEnv(t *testing.T) {
	// SC-e89s06-P1-06: after a BigBase restart, the resumed child process
	// receives the same resolved environment (scope, precedence, redaction).
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}
	st := newDeployStack(t, "test.example.com")
	repoID := createEnvEchoRepo(t, st.database, st.gitDir, "repo-rt-resume", false)
	attachProjectSecret(t, st, "site-rt-resume", 1, "DB_URL", "resume-db-value")

	deployment, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-resume-app", "site-rt-resume", nil, "", "")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deployment.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deployment.ID, "running")

	body := httpGet(t, deployment.Port, 30*time.Second)
	if !strings.Contains(body, "resume-db-value") {
		t.Fatalf("pre-restart echo missing secret: %s", body)
	}

	// Simulate a restart: Stop kills the child; the DB keeps the running row.
	if err := st.dep.Stop(&kernel.Context{}); err != nil {
		t.Fatalf("stop deploy: %v", err)
	}

	restarted := deploy.New(deploy.Options{
		DB:           st.database,
		Logger:       testLogger{},
		BuildsDir:    st.buildsDir,
		GitDir:       st.gitDir,
		Secrets:      st.s,
		PublicDomain: "test.example.com",
		BasePort:     13000,
	})
	if err := restarted.Start(&kernel.Context{}); err != nil {
		t.Fatalf("restart deploy: %v", err)
	}
	t.Cleanup(func() { _ = restarted.Stop(&kernel.Context{}) })

	body = httpGet(t, deployment.Port, 60*time.Second)
	if !strings.Contains(body, "resume-db-value") {
		t.Fatalf("resumed child missing resolved secret: %s", body)
	}
}

func TestRollbackPreservesResolvedEnv(t *testing.T) {
	// SC-e89s06-P1-06: rollback restarts the previous build through the same
	// resolver, so the rolled-back child keeps the resolved Project values.
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH")
	}
	st := newDeployStack(t, "test.example.com")
	repoID := createEnvEchoRepo(t, st.database, st.gitDir, "repo-rt-rollback", false)
	attachProjectSecret(t, st, "site-rt-rollback", 1, "DB_URL", "rollback-db-value")

	deploy1, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-rollback-app", "site-rt-rollback", nil, "", "")
	if err != nil {
		t.Fatalf("trigger 1: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deploy1.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deploy1.ID, "running")

	deploy2, err := st.dep.Trigger(context.Background(), repoID, "main", "rt-rollback-app", "site-rt-rollback", nil, "", "")
	if err != nil {
		t.Fatalf("trigger 2: %v", err)
	}
	waitForDeploymentTerminal(t, st.handler, deploy2.ID, 90*time.Second)
	verifyDeployStatus(t, st.handler, deploy2.ID, "running")
	waitForDeployStatus(t, st.handler, deploy1.ID, "stopped", 30*time.Second)

	rollbackReq := httptest.NewRequest("POST", "/api/deploy/"+deploy2.ID+"/rollback", nil)
	rollbackW := httptest.NewRecorder()
	st.handler.ServeHTTP(rollbackW, rollbackReq)
	if rollbackW.Code != http.StatusOK {
		t.Fatalf("rollback expected 200, got %d: %s", rollbackW.Code, rollbackW.Body.String())
	}
	verifyDeployStatus(t, st.handler, deploy1.ID, "running")

	body := httpGet(t, deploy1.Port, 60*time.Second)
	if !strings.Contains(body, "rollback-db-value") {
		t.Fatalf("rolled-back child missing resolved secret: %s", body)
	}
}
