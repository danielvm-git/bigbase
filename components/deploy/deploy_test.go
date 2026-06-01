package deploy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupDeploy(t *testing.T) (*deploy.Deploy, http.Handler, *db.DB, string) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	gitComp := newGitStub(gitDir)
	buildsDir := t.TempDir()

	dep := deploy.New(deploy.Options{
		DB:        d,
		Logger:    logger,
		BuildsDir: buildsDir,
		GitDir:    gitDir,
	})
	k.Register(d)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return dep, dep.Handler(), d, gitDir
}

type gitStub struct {
	dir string
}

func newGitStub(dir string) *gitStub {
	return &gitStub{dir: dir}
}

func (g *gitStub) Name() string                                  { return "git" }
func (g *gitStub) Version() string                               { return "0.1.0" }
func (g *gitStub) Dependencies() []string                        { return []string{"db"} }
func (g *gitStub) ConfigSchema() json.RawMessage                 { return nil }
func (g *gitStub) Hooks() []kernel.HookDef                       { return nil }
func (g *gitStub) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (g *gitStub) Start(ctx *kernel.Context) error               { return os.MkdirAll(g.dir, 0755) }
func (g *gitStub) Stop(ctx *kernel.Context) error                { return nil }

func createTestRepo(t *testing.T, database *db.DB, repoID, gitDir string) string {
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
		repoID, "test-repo", 0, 1, "main", "test repo")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	repoPath := filepath.Join(gitDir, repoID+".git")
	mustRun(t, "git", "init", "--bare", "-b", "main", repoPath)
	// CI runners may mark temp dirs as dubious ownership (git 2.35+).
	mustRun(t, "git", "config", "--global", "--add", "safe.directory", repoPath)

	sourceDir := t.TempDir()
	mustRun(t, "git", "init", "-b", "main", sourceDir)
	mustRun(t, "git", "config", "--global", "--add", "safe.directory", sourceDir)
	mustRun(t, "git", "-C", sourceDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", sourceDir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("<h1>Hello</h1>"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	mustRun(t, "git", "-C", sourceDir, "add", ".")
	mustRun(t, "git", "-C", sourceDir, "commit", "-m", "initial commit")
	mustRun(t, "git", "-C", sourceDir, "remote", "add", "origin", repoPath)
	mustRun(t, "git", "-C", sourceDir, "push", "-u", "origin", "main")

	return repoID
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	if err := exec.Command(name, args...).Run(); err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
}

func TestDeployImplementsComponent(t *testing.T) {
	var _ kernel.Component = (*deploy.Deploy)(nil)
}

func TestDeployName(t *testing.T) {
	d := &deploy.Deploy{}
	if got := d.Name(); got != "deploy" {
		t.Fatalf("expected Name()='deploy', got '%s'", got)
	}
}

func TestDeployVersion(t *testing.T) {
	d := &deploy.Deploy{}
	if got := d.Version(); got == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestDeployDependencies(t *testing.T) {
	d := &deploy.Deploy{}
	deps := d.Dependencies()
	if len(deps) != 2 || deps[0] != "db" || deps[1] != "git" {
		t.Fatalf("expected dependencies [db git], got %v", deps)
	}
}

func TestDeployHooks(t *testing.T) {
	d := &deploy.Deploy{}
	if got := d.Hooks(); len(got) != 0 {
		t.Fatalf("expected no hooks, got %v", got)
	}
}

func TestDeployCreateMissingRepoID(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"branch": "main"})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeployCreateRepoNotFound(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": "nonexistent"})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeployCreateInvalidJSON(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	req := httptest.NewRequest("POST", "/api/deploy", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeployCreateSuccess(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-1", gitDir)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var dep map[string]any
	if err := json.NewDecoder(w.Body).Decode(&dep); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if id, ok := dep["id"]; !ok || id == "" {
		t.Fatalf("expected non-empty id, got: %v", dep)
	}
	if status, ok := dep["status"]; !ok || status != "pending" {
		t.Fatalf("expected status 'pending', got: %v", dep)
	}
	if p, ok := dep["port"]; !ok || p == float64(0) {
		t.Fatalf("expected non-zero port, got: %v", dep)
	}
}

func TestDeployCreateWrongMethod(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	req := httptest.NewRequest("PUT", "/api/deploy", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestDeployListEmpty(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	req := httptest.NewRequest("GET", "/api/deploy", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := body["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("expected empty list, got %d items", len(data))
	}
}

func TestDeployListAfterCreate(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-2", gitDir)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d: %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/api/deploy", nil)
	listW := httptest.NewRecorder()
	handler.ServeHTTP(listW, listReq)

	var body map[string]any
	_ = json.NewDecoder(listW.Body).Decode(&body)
	data, _ := body["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 deployment, got %d: %v", len(data), body)
	}
}

func TestDeployGetByID(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-3", gitDir)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	depID := created["id"].(string)

	getReq := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var got map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&got)
	if got["id"] != depID {
		t.Fatalf("expected id '%s', got '%s'", depID, got["id"])
	}
}

func TestDeployGetByIDNotFound(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	req := httptest.NewRequest("GET", "/api/deploy/nonexistent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeployGetByIDWrongMethod(t *testing.T) {
	_, handler, _, _ := setupDeploy(t)

	req := httptest.NewRequest("DELETE", "/api/deploy/some-id", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestDetectAppTypeNode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.DetectAppType(dir); got != "node" {
		t.Fatalf("expected 'node', got '%s'", got)
	}
}

func TestDetectAppTypeGo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.DetectAppType(dir); got != "go" {
		t.Fatalf("expected 'go', got '%s'", got)
	}
}

func TestDetectAppTypePython(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.DetectAppType(dir); got != "python" {
		t.Fatalf("expected 'python', got '%s'", got)
	}
}

func TestDetectAppTypeStatic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Hi</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.DetectAppType(dir); got != "static" {
		t.Fatalf("expected 'static', got '%s'", got)
	}
}

func TestDetectAppTypeUnknownFallbackToStatic(t *testing.T) {
	dir := t.TempDir()
	if got := deploy.DetectAppType(dir); got != "static" {
		t.Fatalf("expected 'static' fallback, got '%s'", got)
	}
}

func TestGetStartCommandWithStartScript(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"start":"node server.js"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.GetStartCommand(dir); got != "node" {
		t.Fatalf("expected 'node', got '%s'", got)
	}
}

func TestGetStartCommandNoStartScript(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.GetStartCommand(dir); got != "node index.js" {
		t.Fatalf("expected 'node index.js', got '%s'", got)
	}
}

func TestGetStartCommandNoPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if got := deploy.GetStartCommand(dir); got != "node index.js" {
		t.Fatalf("expected 'node index.js', got '%s'", got)
	}
}

func TestGetStartCommandMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`not json`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.GetStartCommand(dir); got != "node index.js" {
		t.Fatalf("expected 'node index.js' fallback, got '%s'", got)
	}
}

func TestUpdateURLUpdatesPort(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "repo-url-test", gitDir)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	depID := created["id"].(string)

	// Wait briefly for async deployment to begin processing
	time.Sleep(500 * time.Millisecond)

	// List and verify deployment has a URL
	listReq := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
	listW := httptest.NewRecorder()
	handler.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}

	var got map[string]any
	_ = json.NewDecoder(listW.Body).Decode(&got)
	if url, ok := got["url"]; !ok || url == "" {
		t.Fatalf("expected non-empty url after deployment, got: %v", got)
	}
	if port, ok := got["port"]; !ok || port == float64(0) {
		t.Fatalf("expected non-zero port, got: %v", got)
	}
}

func TestRunDeploymentRepoNotFoundOnDisk(t *testing.T) {
	// Use setupDeploy which creates the kernel + db + git_stub
	// Create a repo in the DB but without a corresponding bare git repo on disk
	_, handler, database, gitDir := setupDeploy(t)

	// Manually insert a repo row - no bare repo created
	_, err := database.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS git_repos (
			id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, owner_id INTEGER NOT NULL,
			private INTEGER NOT NULL DEFAULT 1, default_branch TEXT NOT NULL DEFAULT 'main',
			description TEXT DEFAULT '', created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	if err != nil {
		t.Fatalf("create git_repos table: %v", err)
	}

	// Insert but DON'T call createTestRepo (which creates the bare git repo)
	_, err = database.ExecContext(context.Background(),
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		"repo-no-clone", "no-clone-repo", 0, 1, "main", "")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	_ = gitDir

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": "repo-no-clone"})
	req := httptest.NewRequest("POST", "/api/deploy", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	depID := created["id"].(string)

	// Wait for async runDeployment to try cloning and fail
	time.Sleep(1 * time.Second)

	getReq := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var got map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&got)
	if got["status"] != "failed" {
		t.Fatalf("expected status 'failed' for repo without git dir, got '%s'", got["status"])
	}
}

func TestRunDeploymentStaticSite(t *testing.T) {
	dep, handler, database, gitDir := setupDeploy(t)
	_ = dep

	// Create a repo with an index.html (detected as static)
	repoID := createTestRepo(t, database, "repo-static-test", gitDir)

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
	_ = json.NewDecoder(w.Body).Decode(&created)
	depID := created["id"].(string)

	// Wait for async deploy (clone + detect static + serve)
	time.Sleep(2 * time.Second)

	getReq := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var got map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&got)

	if got["status"] != "running" {
		t.Fatalf("expected status 'running', got '%s'", got["status"])
	}
	if appType, ok := got["app_type"]; !ok || appType != "static" {
		t.Fatalf("expected app_type 'static', got '%v'", appType)
	}
	if url, ok := got["url"]; !ok || url == "" {
		t.Fatalf("expected non-empty url, got: %v", got)
	}
}
