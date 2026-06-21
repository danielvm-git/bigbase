package deploy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type AppType string

const (
	AppNode   AppType = "node"
	AppGo     AppType = "go"
	AppPython AppType = "python"
	AppStatic AppType = "static"
)

type Deployment struct {
	ID               string   `json:"id"`
	RepoID           string   `json:"repo_id"`
	SiteID           string   `json:"site_id"`
	Branch           string   `json:"branch"`
	CommitSHA        string   `json:"commit_sha"`
	Status           string   `json:"status"`
	URL              string   `json:"url"`
	Port             int      `json:"port"`
	AppType          AppType  `json:"app_type"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	PassthroughPaths []string `json:"passthrough_paths,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type runningApp struct {
	cmd     *exec.Cmd
	server  *http.Server
	port    int
	buildID string
	host    string
}

// DeploymentHostRegistry routes public hostnames to local deployment ports.
type DeploymentHostRegistry interface {
	RegisterDeploymentHost(host string, port int, siteID string, passthroughPaths []string, metadata map[string]string) error
	UnregisterDeploymentHost(host string)
}

type Deploy struct {
	db           DBer
	logger       Logger
	buildsDir    string
	gitDir       string
	buildHome    string
	basePort     int
	publicDomain string
	useHTTPS     bool
	hostRouter   DeploymentHostRegistry
	deployLogsMu sync.RWMutex
	deployLogs   map[string][]string
	mu           sync.Mutex
	nextPort     int
	apps         map[string]*runningApp
	unsubscribe  func()
	logHubsMu    sync.RWMutex
	logHubs      map[string]*logHub
}

type Options struct {
	DB           DBer
	Logger       Logger
	BuildsDir    string
	GitDir       string
	BuildHome    string
	BasePort     int
	PublicDomain string
	UseHTTPS     bool
	HostRouter   DeploymentHostRegistry
}

func New(opts Options) *Deploy {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	dir := opts.BuildsDir
	if dir == "" {
		dir = "data/builds"
	}
	absDir, err := filepath.Abs(dir)
	if err == nil {
		dir = absDir
	}
	gitDir := opts.GitDir
	if gitDir == "" {
		gitDir = "data/git"
	}
	absGitDir, err := filepath.Abs(gitDir)
	if err == nil {
		gitDir = absGitDir
	}
	basePort := opts.BasePort
	if basePort == 0 {
		basePort = 10000
	}
	useHTTPS := opts.UseHTTPS
	if strings.TrimSpace(opts.PublicDomain) != "" {
		useHTTPS = true
	}
	return &Deploy{
		db:           opts.DB,
		logger:       logger,
		buildsDir:    dir,
		gitDir:       gitDir,
		buildHome:    strings.TrimSpace(opts.BuildHome),
		basePort:     basePort,
		publicDomain: strings.TrimSpace(opts.PublicDomain),
		useHTTPS:     useHTTPS,
		hostRouter:   opts.HostRouter,
		nextPort:     basePort,
		apps:         make(map[string]*runningApp),
		logHubs:      make(map[string]*logHub),
	}
}

func (d *Deploy) Name() string                  { return "deploy" }
func (d *Deploy) Version() string               { return version }
func (d *Deploy) Dependencies() []string        { return []string{"db", "git"} }
func (d *Deploy) ConfigSchema() json.RawMessage { return nil }
func (d *Deploy) Hooks() []kernel.HookDef       { return nil }

func (d *Deploy) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (d *Deploy) Start(ctx *kernel.Context) error {
	if err := os.MkdirAll(d.buildsDir, 0755); err != nil {
		return fmt.Errorf("create builds dir: %w", err)
	}
	if err := d.db.Migrate(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		site_id TEXT NOT NULL DEFAULT '',
		branch TEXT NOT NULL DEFAULT 'main',
		commit_sha TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		url TEXT DEFAULT '',
		port INTEGER DEFAULT 0,
		app_type TEXT DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		build_log TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate deployments table: %w", err)
	}
	if err := d.ensureErrorMessageColumn(); err != nil {
		return err
	}
	if err := d.ensureBuildLogColumn(); err != nil {
		return err
	}
	if err := d.ensureSiteIDColumn(); err != nil {
		return err
	}
	if err := d.ensurePassthroughPathsColumn(); err != nil {
		return err
	}
	if err := d.ensurePIDColumn(); err != nil {
		return err
	}
	if err := d.db.Migrate(`CREATE TABLE IF NOT EXISTS site_request_logs (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate site_request_logs table: %w", err)
	}
	if err := d.db.Migrate(`CREATE INDEX IF NOT EXISTS idx_srl_site_id ON site_request_logs(site_id, created_at DESC)`); err != nil {
		return fmt.Errorf("migrate site_request_logs index: %w", err)
	}
	d.restoreRunningDeploymentHosts()
	candidates := d.loadResumeCandidates()
	d.logger.Info("deploy component ready", "dir", d.buildsDir, "resume", len(candidates))
	go d.resumeCandidates(candidates)
	return nil
}

type resumeCandidate struct {
	id, repoID, repoName, rawURL, appType, siteID string
	port                                          int
	passthroughPaths                              []string
}

func (d *Deploy) loadResumeCandidates() []resumeCandidate {
	ctx := context.Background()
	rows, err := d.db.QueryContext(ctx,
		`SELECT d.id, d.repo_id, g.name, d.url, d.port, d.app_type, d.site_id, d.passthrough_paths
		 FROM deployments d
		 JOIN git_repos g ON g.id = d.repo_id
		 WHERE d.status = 'running' AND d.port > 0
		 ORDER BY d.created_at DESC`)
	if err != nil {
		d.logger.Warn("load resume candidates", "error", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	seenHost := make(map[string]bool)
	var out []resumeCandidate
	for rows.Next() {
		var c resumeCandidate
		var passthroughJSON string
		if err := rows.Scan(&c.id, &c.repoID, &c.repoName, &c.rawURL, &c.port, &c.appType, &c.siteID, &passthroughJSON); err != nil {
			continue
		}
		c.passthroughPaths = parsePassthroughPaths(passthroughJSON)
		host := HostFromDeploymentURL(c.rawURL)
		if host == "" || host == "localhost" || seenHost[host] {
			continue
		}
		seenHost[host] = true
		out = append(out, c)
	}
	return out
}

func (d *Deploy) resumeCandidates(candidates []resumeCandidate) {
	for _, c := range candidates {
		buildDir := filepath.Join(d.buildsDir, c.id)
		if _, err := os.Stat(buildDir); err != nil {
			continue
		}
		d.mu.Lock()
		_, already := d.apps[c.id]
		d.mu.Unlock()
		if already {
			continue
		}
		appType := AppType(c.appType)
		serveDir := buildDir
		if appType == AppNode {
			if _, err := os.Stat(filepath.Join(buildDir, "dist")); err == nil {
				serveDir = filepath.Join(buildDir, "dist")
				appType = AppStatic
			} else if _, err := os.Stat(filepath.Join(buildDir, "build")); err == nil {
				// SvelteKit adapter-static outputs to build/ by default
				serveDir = filepath.Join(buildDir, "build")
				appType = AppStatic
			} else {
				continue
			}
		}
		if appType == AppStatic {
			if _, err := os.Stat(filepath.Join(buildDir, "dist")); err == nil {
				serveDir = filepath.Join(buildDir, "dist")
			} else if _, err := os.Stat(filepath.Join(buildDir, "build")); err == nil {
				serveDir = filepath.Join(buildDir, "build")
			}
		}
		if appType != AppStatic {
			continue
		}
		deploy := &Deployment{
			ID:      c.id,
			RepoID:  c.repoID,
			SiteID:  c.siteID,
			Port:    c.port,
			URL:     c.rawURL,
			AppType: AppStatic,
		}
		go d.serveStatic(context.Background(), serveDir, deploy, c.repoName)
		d.logger.Info("resumed static deployment", "id", c.id, "port", c.port, "host", HostFromDeploymentURL(c.rawURL))
	}
}

func (d *Deploy) restoreRunningDeploymentHosts() {
	if d.hostRouter == nil {
		return
	}
	ctx := context.Background()
	rows, err := d.db.QueryContext(ctx,
		`SELECT url, port, site_id, commit_sha, passthrough_paths FROM deployments WHERE status = 'running' AND port > 0`)
	if err != nil {
		d.logger.Warn("restore deployment hosts", "error", err)
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var rawURL, siteID, commitSHA, passthroughJSON string
		var port int
		if err := rows.Scan(&rawURL, &port, &siteID, &commitSHA, &passthroughJSON); err != nil {
			continue
		}
		host := HostFromDeploymentURL(rawURL)
		if host == "" {
			continue
		}
		passthroughPaths := parsePassthroughPaths(passthroughJSON)
		metadata := map[string]string{}
		if commitSHA != "" {
			metadata["version"] = commitSHA
		}
		if err := d.hostRouter.RegisterDeploymentHost(host, port, siteID, passthroughPaths, metadata); err != nil {
			d.logger.Warn("restore deployment host", "host", host, "error", err)
		}
	}
}

func (d *Deploy) Stop(ctx *kernel.Context) error {
	if d.unsubscribe != nil {
		d.unsubscribe()
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	for id, app := range d.apps {
		if app.host != "" && d.hostRouter != nil {
			d.hostRouter.UnregisterDeploymentHost(app.host)
		}
		// Shutdown static HTTP servers
		if app.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = app.server.Shutdown(shutdownCtx)
			cancel()
		}
		// Kill process-based deployments
		if app.cmd != nil && app.cmd.Process != nil {
			_ = app.cmd.Process.Kill()
		}
		delete(d.apps, id)
	}
	return nil
}

func (d *Deploy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/deploy", d.handleDeploy)
	mux.HandleFunc("/api/deploy/", d.handleDeployByID)
	mux.HandleFunc("/api/samples", d.handleSamples)
	mux.HandleFunc("/api/samples/", d.handleSamples)
	mux.HandleFunc("/api/deploy/stats", d.handleDeployStats)
	return mux
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func pickPort(base int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000))
	return base + int(n.Int64())
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// marshalPassthroughPaths converts a string slice to a JSON array string for DB storage.
func marshalPassthroughPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	data, _ := json.Marshal(paths)
	return string(data)
}

// parsePassthroughPaths parses a JSON array string from the DB into a string slice.
func parsePassthroughPaths(raw string) []string {
	if raw == "" {
		return nil
	}
	var paths []string
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil
	}
	return paths
}

func (d *Deploy) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		RepoID           string   `json:"repo_id"`
		Branch           string   `json:"branch"`
		SiteName         string   `json:"site_name"`
		SiteID           string   `json:"site_id"`
		PassthroughPaths []string `json:"passthrough_paths"`
		AppType          string   `json:"app_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.RepoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id is required"})
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	deploy, err := d.Trigger(r.Context(), req.RepoID, req.Branch, req.SiteName, req.SiteID, req.PassthroughPaths, req.AppType)
	if err != nil {
		switch err.Error() {
		case "repo not found":
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		default:
			d.logger.Error("create deployment", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, deploy)
}

// Trigger starts a deployment for a git repo without going through HTTP.
func (d *Deploy) Trigger(ctx context.Context, repoID, branch, siteName, siteID string, passthroughPaths []string, appType string) (*Deployment, error) {
	if repoID == "" {
		return nil, fmt.Errorf("repo_id is required")
	}
	if branch == "" {
		branch = "main"
	}

	var repoName string
	err := d.db.QueryRowContext(ctx, "SELECT name FROM git_repos WHERE id = ?", repoID).Scan(&repoName)
	if err != nil {
		return nil, fmt.Errorf("repo not found")
	}

	if siteName == "" {
		siteName = repoName
	}

	// Stop any previous running deployment for the same site or repo.
	// By unregistering the old host before the new deployment starts,
	// finalizeDeploymentURL will successfully register the new port.
	d.stopPreviousDeployments(ctx, siteID, repoID)

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	buildDir := filepath.Join(d.buildsDir, id)
	port := pickPort(d.basePort)

	passthroughJSON := marshalPassthroughPaths(passthroughPaths)
	deploy := &Deployment{
		ID:               id,
		RepoID:           repoID,
		SiteID:           siteID,
		Branch:           branch,
		Status:           "pending",
		Port:             port,
		AppType:          AppType(appType),
		PassthroughPaths: passthroughPaths,
		URL:              deploymentURL(d.publicDomain, d.useHTTPS, siteName, port),
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	if _, err := d.db.ExecContext(ctx,
		"INSERT INTO deployments (id, repo_id, site_id, branch, status, port, url, passthrough_paths, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, repoID, siteID, branch, deploy.Status, deploy.Port, deploy.URL, passthroughJSON, deploy.CreatedAt); err != nil {
		return nil, err
	}

	d.initDeployLogs(id)
	d.appendDeployLog(id, fmt.Sprintf("→ Deployment started (branch: %s)", branch))

	// After finalizeDeploymentURL runs, the proxy will route to the new port.
	// Any old deployment stopped above is already unregistered and cleaned up.
	go d.runDeployment(deploy, buildDir, siteName)
	return deploy, nil
}

// stopDeployment stops a running deployment: kills the process, shuts down the
// static server, unregisters the proxy host, and optionally updates DB status.
func (d *Deploy) stopDeployment(id, newStatus string) {
	d.mu.Lock()
	app, hasApp := d.apps[id]
	if hasApp {
		delete(d.apps, id)
	}
	d.mu.Unlock()

	if hasApp {
		if app.cmd != nil && app.cmd.Process != nil {
			_ = app.cmd.Process.Kill()
		}
		if app.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = app.server.Shutdown(shutdownCtx)
			cancel()
		}
		if d.hostRouter != nil && app.host != "" {
			d.hostRouter.UnregisterDeploymentHost(app.host)
		}
	} else {
		// Orphaned process recovery: the deployment was started in a previous
		// BigBase session (or survived a crash). Read the PID from DB and kill.
		var pid int
		err := d.db.QueryRowContext(context.Background(),
			"SELECT pid FROM deployments WHERE id = ?", id).Scan(&pid)
		if err == nil && pid > 0 {
			if p, err := os.FindProcess(pid); err == nil {
				if err := p.Kill(); err == nil {
					d.logger.Info("killed orphaned process", "id", id, "pid", pid)
				}
				_, _ = p.Wait() // reap zombie so Signal(0) reflects dead state
			}
		}
	}

	if newStatus != "" {
		_, _ = d.db.ExecContext(context.Background(),
			"UPDATE deployments SET status = ? WHERE id = ?", newStatus, id)
	}
}

// stopPreviousDeployments finds and stops any running deployments for the same
// site (by site_id) or same repo (by repo_id), marking them as "replaced".
// IDs are collected first, then stopped, to avoid nesting DB queries on a
// single-connection SQLite pool.
func (d *Deploy) stopPreviousDeployments(ctx context.Context, siteID, repoID string) {
	var query string
	var args []any
	if siteID != "" {
		query = "SELECT id FROM deployments WHERE site_id = ? AND status = 'running'"
		args = []any{siteID}
	} else {
		query = "SELECT id FROM deployments WHERE repo_id = ? AND status = 'running'"
		args = []any{repoID}
	}
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		d.logger.Warn("query previous deployments", "error", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	_ = rows.Close()

	for _, id := range ids {
		d.stopDeployment(id, "replaced")
		d.logger.Info("stopped previous deployment", "replaced", id)
	}
}

func (d *Deploy) runDeployment(deploy *Deployment, buildDir, repoName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Eagerly create the logHub so all log lines from the very start are captured.
	d.getOrCreateHub(deploy.ID)

	d.updateStatus(deploy.ID, "building")
	d.appendDeployLog(deploy.ID, "→ Status: building")

	repoPath := filepath.Join(d.gitDir, deploy.RepoID+".git")
	if _, err := os.Stat(repoPath); err != nil {
		d.logger.Error("repo not found on disk", "id", deploy.RepoID, "path", repoPath)
		d.appendDeployLog(deploy.ID, "✗ Repository not found on disk")
		d.updateStatus(deploy.ID, "failed")
		return
	}

	if err := os.MkdirAll(buildDir, 0755); err != nil {
		d.logger.Error("create build dir", "error", err)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ Create build directory failed: %v", err))
		d.updateStatus(deploy.ID, "failed")
		return
	}

	d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Cloning repository (branch: %s)", deploy.Branch))
	if err := d.cloneAndCheckout(ctx, deploy.ID, repoPath, buildDir, deploy.Branch); err != nil {
		d.logger.Error("clone repo", "error", err)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ Clone failed: %v", err))
		d.updateStatus(deploy.ID, "failed")
		return
	}
	d.appendDeployLog(deploy.ID, "→ Clone complete")

	commitSHA, _ := d.getCommitSHA(buildDir)
	deploy.CommitSHA = commitSHA
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET commit_sha = ? WHERE id = ?", commitSHA, deploy.ID)
	if commitSHA != "" {
		short := commitSHA
		if len(short) > 7 {
			short = short[:7]
		}
		d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Commit: %s", short))
	}

	appType := deploy.AppType
	if appType == "" {
		appType = DetectAppType(buildDir)
	}
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET app_type = ? WHERE id = ?", string(appType), deploy.ID)
	d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Detected app type: %s", appType))

	if appType == AppStatic {
		d.appendDeployLog(deploy.ID, "→ Serving static files")
		d.updateStatus(deploy.ID, "running")
		d.finalizeDeploymentURL(deploy, repoName)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Deployed at %s", deploy.URL))
		go d.serveStatic(context.Background(), buildDir, deploy, repoName)
		return
	}

	if err := d.buildApp(ctx, deploy.ID, buildDir, appType); err != nil {
		d.logger.Error("build app", "type", appType, "error", err)
		d.failDeployment(deploy.ID, err)
		return
	}

	serveDir := buildDir
	if appType == AppNode {
		if _, err := os.Stat(filepath.Join(buildDir, "dist")); err == nil {
			serveDir = filepath.Join(buildDir, "dist")
			appType = AppStatic
			deploy.AppType = AppStatic
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET app_type = ? WHERE id = ?", string(AppStatic), deploy.ID)
		} else if _, err := os.Stat(filepath.Join(buildDir, "build")); err == nil {
			// SvelteKit adapter-static outputs to build/ by default
			serveDir = filepath.Join(buildDir, "build")
			appType = AppStatic
			deploy.AppType = AppStatic
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET app_type = ? WHERE id = ?", string(AppStatic), deploy.ID)
		}
	}

	d.updateStatus(deploy.ID, "running")
	d.finalizeDeploymentURL(deploy, repoName)
	d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Deployed at %s", deploy.URL))

	if appType == AppStatic {
		d.appendDeployLog(deploy.ID, "→ Serving static files")
		go d.serveStatic(context.Background(), serveDir, deploy, repoName)
	} else {
		d.appendDeployLog(deploy.ID, "→ Starting application")
		go d.startApp(context.Background(), serveDir, deploy, appType, repoName)
	}
}

func (d *Deploy) cloneAndCheckout(ctx context.Context, deployID, repoPath, buildDir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", repoPath, ".")
	cmd.Dir = buildDir
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	if branch != "main" {
		checkout := exec.CommandContext(ctx, "git", "checkout", branch)
		checkout.Dir = buildDir
		if err := checkout.Run(); err != nil {
			return fmt.Errorf("git checkout %s: %w", branch, err)
		}
	}

	return nil
}

func (d *Deploy) getCommitSHA(buildDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = buildDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *Deploy) buildApp(ctx context.Context, deployID string, buildDir string, appType AppType) error {
	switch appType {
	case AppNode:
		if err := d.runBuildCommand(ctx, deployID, buildDir, "npm", "install"); err != nil {
			return fmt.Errorf("npm install: %w", err)
		}
		if err := ValidateNodeBuildScript(buildDir); err != nil {
			d.appendDeployLog(deployID, "✗ "+err.Error())
			return err
		}
		if err := d.runBuildCommand(ctx, deployID, buildDir, "npm", "run", "build"); err != nil {
			return fmt.Errorf("npm run build: %w", err)
		}
		return nil
	case AppGo:
		return d.runBuildCommand(ctx, deployID, buildDir, "go", "build", "-o", "app", ".")
	case AppPython:
		return d.runBuildCommand(ctx, deployID, buildDir, "pip", "install", "--break-system-packages", "-r", "requirements.txt")
	}
	return nil
}

func FormatBuildCommand(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func (d *Deploy) runBuildCommand(ctx context.Context, deployID, dir, name string, args ...string) error {
	label := FormatBuildCommand(name, args...)
	d.appendDeployLog(deployID, "→ Running: "+label)

	var stderr, stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = d.buildCmdEnv()
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail != "" {
			d.appendDeployLogBlock(deployID, detail)
			return fmt.Errorf("%w: %s", err, detail)
		}
		d.appendDeployLog(deployID, fmt.Sprintf("✗ Command failed: %v", err))
		return err
	}
	d.appendDeployLog(deployID, "→ "+label+" complete")
	return nil
}

func (d *Deploy) startApp(ctx context.Context, buildDir string, deploy *Deployment, appType AppType, repoName string) {
	var cmd *exec.Cmd

	switch appType {
	case AppNode:
		startCmd := GetStartCommand(buildDir)
		parts := strings.Fields(startCmd)
		if len(parts) == 0 {
			parts = []string{"node", "index.js"}
		}
		args := append([]string{"exec", "--"}, parts...)
		cmd = exec.CommandContext(ctx, "npm", args...)
		cmd.Dir = buildDir
	case AppGo:
		cmd = exec.CommandContext(ctx, filepath.Join(buildDir, "app"))
		cmd.Dir = buildDir
	case AppPython:
		pythonBin := "python3"
		if _, err := exec.LookPath(pythonBin); err != nil {
			pythonBin = "python"
		}
		cmd = exec.CommandContext(ctx, pythonBin, "app.py")
		cmd.Dir = buildDir
	}

	if cmd == nil {
		d.logger.Error("no start command for", "type", appType)
		d.updateStatus(deploy.ID, "failed")
		return
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", deploy.Port))

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	host := deploymentHost(d.publicDomain, repoName)
	d.mu.Lock()
	d.apps[deploy.ID] = &runningApp{cmd: cmd, port: deploy.Port, buildID: deploy.ID, host: host}
	d.mu.Unlock()

	if err := cmd.Start(); err != nil {
		d.logger.Error("start app", "id", deploy.ID, "error", err)
		d.updateStatus(deploy.ID, "failed")
		return
	}

	// Persist PID to DB so orphaned processes can be killed after restart
	if pid := cmd.Process.Pid; pid > 0 {
		_, _ = d.db.ExecContext(context.Background(),
			"UPDATE deployments SET pid = ? WHERE id = ?", pid, deploy.ID)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			d.appendDeployLog(deploy.ID, "[runtime] "+scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			d.appendDeployLog(deploy.ID, "[runtime] "+scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		d.logger.Error("app exited", "id", deploy.ID, "error", err)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ App exited: %v", err))
		d.updateStatus(deploy.ID, "failed")
		if host != "" && d.hostRouter != nil {
			d.hostRouter.UnregisterDeploymentHost(host)
		}
	}
}

func (d *Deploy) serveStatic(ctx context.Context, buildDir string, deploy *Deployment, repoName string) {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(buildDir)))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", deploy.Port),
		Handler: mux,
	}

	host := deploymentHost(d.publicDomain, repoName)
	d.mu.Lock()
	d.apps[deploy.ID] = &runningApp{port: deploy.Port, buildID: deploy.ID, server: server, host: host}
	d.mu.Unlock()

	d.logger.Info("serving static site", "id", deploy.ID, "port", deploy.Port, "url", deploy.URL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		d.logger.Error("static server error", "id", deploy.ID, "error", err)
	}
}

func (d *Deploy) updateStatus(id, status string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET status = ? WHERE id = ?", status, id)

	if status == "running" || status == "failed" {
		lines := d.getDeployLogs(id)
		if len(lines) > 0 {
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET build_log = ? WHERE id = ?", strings.Join(lines, "\n"), id)
		}
	}
}

func (d *Deploy) failDeployment(id string, buildErr error) {
	msg := buildErr.Error()
	const maxLen = 2000
	if len(msg) > maxLen {
		msg = msg[:maxLen]
	}
	d.appendDeployLog(id, "✗ Deploy failed: "+msg)
	// Close the log stream so WebSocket subscribers know the deployment failed.
	d.closeLogStream(id)

	// Use a more direct update here since we have the message
	d.mu.Lock()
	defer d.mu.Unlock()
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET status = ?, error_message = ? WHERE id = ?", "failed", msg, id)

	lines := d.getDeployLogs(id)
	if len(lines) > 0 {
		_, _ = d.db.ExecContext(context.Background(),
			"UPDATE deployments SET build_log = ? WHERE id = ?", strings.Join(lines, "\n"), id)
	}
}

func (d *Deploy) ensureSiteIDColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN site_id TEXT NOT NULL DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add site_id column: %w", err)
}

func (d *Deploy) ensureErrorMessageColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN error_message TEXT NOT NULL DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add error_message column: %w", err)
}

func (d *Deploy) ensureBuildLogColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN build_log TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add build_log column: %w", err)
}

func (d *Deploy) ensurePassthroughPathsColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN passthrough_paths TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add passthrough_paths column: %w", err)
}

func (d *Deploy) ensurePIDColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN pid INTEGER NOT NULL DEFAULT 0`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add pid column: %w", err)
}

func (d *Deploy) finalizeDeploymentURL(deploy *Deployment, repoName string) {
	url := deploymentURL(d.publicDomain, d.useHTTPS, repoName, deploy.Port)
	host := deploymentHost(d.publicDomain, repoName)
	deploy.URL = url

	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET url = ?, port = ? WHERE id = ?",
		url, deploy.Port, deploy.ID)

	if d.hostRouter != nil && host != "" {
		metadata := d.buildMetadata(deploy)
		if err := d.hostRouter.RegisterDeploymentHost(host, deploy.Port, deploy.SiteID, deploy.PassthroughPaths, metadata); err != nil {
			d.logger.Warn("register deployment host", "host", host, "error", err)
		}
	}
}

// buildMetadata constructs the __BIGBASE_METADATA__ map for a deployment.
func (d *Deploy) buildMetadata(deploy *Deployment) map[string]string {
	m := map[string]string{
		"deployedAt": time.Now().UTC().Format(time.RFC3339),
	}
	if deploy.CommitSHA != "" {
		m["version"] = deploy.CommitSHA
	}
	return m
}

func DetectAppType(buildDir string) AppType {
	if fileExists(filepath.Join(buildDir, "package.json")) {
		return AppNode
	}
	if fileExists(filepath.Join(buildDir, "go.mod")) {
		return AppGo
	}
	// Python: only if a runnable entry point exists at root. requirements.txt
	// alone is insufficient — many Node/static repos ship it alongside a Python
	// tool without an actual web-server entry point.
	if fileExists(filepath.Join(buildDir, "app.py")) ||
		fileExists(filepath.Join(buildDir, "main.py")) {
		return AppPython
	}
	if fileExists(filepath.Join(buildDir, "index.html")) {
		return AppStatic
	}
	return AppStatic
}

func GetStartCommand(buildDir string) string {
	data, err := os.ReadFile(filepath.Join(buildDir, "package.json"))
	if err != nil {
		return "node index.js"
	}

	var pkg struct {
		Scripts struct {
			Start string `json:"start"`
		} `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "node index.js"
	}

	if pkg.Scripts.Start != "" {
		return pkg.Scripts.Start
	}
	return "node index.js"
}

// ValidateNodeBuildScript checks that a package.json file exists at dir and
// contains a "build" script. Returns a clear error message if not.
func ValidateNodeBuildScript(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return fmt.Errorf("cannot read package.json: %w", err)
	}
	var pkg struct {
		Scripts struct {
			Build string `json:"build"`
		} `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("invalid package.json: %w", err)
	}
	if pkg.Scripts.Build == "" {
		return fmt.Errorf("no build script found in package.json: add a \"build\" entry to \"scripts\" or set app_type to static")
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (d *Deploy) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := d.db.QueryContext(r.Context(),
		"SELECT id, repo_id, site_id, COALESCE(branch,'main'), COALESCE(commit_sha,''), COALESCE(status,'pending'), COALESCE(url,''), COALESCE(port,0), COALESCE(app_type,''), COALESCE(error_message,''), COALESCE(passthrough_paths,''), created_at FROM deployments ORDER BY created_at DESC")
	if err != nil {
		d.logger.Error("list deployments", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	deployments := make([]Deployment, 0)
	for rows.Next() {
		var dep Deployment
		var passthroughJSON string
		if err := rows.Scan(&dep.ID, &dep.RepoID, &dep.SiteID, &dep.Branch, &dep.CommitSHA, &dep.Status, &dep.URL, &dep.Port, &dep.AppType, &dep.ErrorMessage, &passthroughJSON, &dep.CreatedAt); err != nil {
			d.logger.Error("scan deployment", "error", err)
			continue
		}
		dep.PassthroughPaths = parsePassthroughPaths(passthroughJSON)
		deployments = append(deployments, dep)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": deployments})
}

func (d *Deploy) handleDeploy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		d.HandleCreate(w, r)
	case "GET":
		d.HandleList(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (d *Deploy) handleDeployByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/deploy/")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	// Check for logs sub-path: /api/deploy/:id/logs/stream (WebSocket) or /api/deploy/:id/logs
	if strings.HasSuffix(path, "/logs/stream") {
		id := strings.TrimSuffix(path, "/logs/stream")
		d.handleLogsStream(w, r, id)
		return
	}
	if strings.HasSuffix(path, "/logs") {
		id := strings.TrimSuffix(path, "/logs")
		d.handleDeployLogs(w, r, id)
		return
	}

	id := path
	switch r.Method {
	case http.MethodDelete:
		d.handleDeleteDeployment(w, r, id)
	case http.MethodGet:
		var dep Deployment
		var appType, passthroughJSON string
		err := d.db.QueryRowContext(r.Context(),
			"SELECT id, repo_id, site_id, branch, commit_sha, status, url, port, app_type, COALESCE(error_message,''), COALESCE(passthrough_paths,''), created_at FROM deployments WHERE id = ?", id).
			Scan(&dep.ID, &dep.RepoID, &dep.SiteID, &dep.Branch, &dep.CommitSHA, &dep.Status, &dep.URL, &dep.Port, &appType, &dep.ErrorMessage, &passthroughJSON, &dep.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
			return
		}
		dep.AppType = AppType(appType)
		dep.PassthroughPaths = parsePassthroughPaths(passthroughJSON)
		writeJSON(w, http.StatusOK, dep)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (d *Deploy) handleDeleteDeployment(w http.ResponseWriter, r *http.Request, id string) {
	var status, url string
	err := d.db.QueryRowContext(r.Context(),
		"SELECT status, COALESCE(url,'') FROM deployments WHERE id = ?", id).
		Scan(&status, &url)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}
	if status == "pending" || status == "building" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "deployment is in progress — wait or trigger a new deploy"})
		return
	}

	d.stopDeployment(id, "replaced")

	_ = os.RemoveAll(filepath.Join(d.buildsDir, id))
	d.deleteDeployLogs(id)
	_, _ = d.db.ExecContext(r.Context(), "DELETE FROM deployments WHERE id = ?", id)

	w.WriteHeader(http.StatusNoContent)
}

func (d *Deploy) handleDeployLogs(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Verify deployment exists and load status + error message + build_log
	var status, errMsg, buildLog string
	err := d.db.QueryRowContext(r.Context(),
		"SELECT status, COALESCE(error_message,''), COALESCE(build_log,'') FROM deployments WHERE id = ?", id).
		Scan(&status, &errMsg, &buildLog)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}

	lines := d.getDeployLogs(id)
	if len(lines) == 0 && buildLog != "" {
		lines = strings.Split(buildLog, "\n")
	}

	payload := map[string]any{
		"deployment_id": id,
		"status":        status,
		"lines":         lines,
		"log_available": len(lines) > 0,
	}
	if errMsg != "" {
		payload["error_message"] = errMsg
	}
	writeJSON(w, http.StatusOK, payload)
}

// handleDeployStats returns aggregate deployment statistics: total, running,
// failed counts, and a rolling failure rate over the last 24 hours.
func (d *Deploy) handleDeployStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx := r.Context()
	stats := map[string]any{}

	var total int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments").Scan(&total); err == nil {
		stats["total"] = total
	}

	var running int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments WHERE status = 'running'").Scan(&running); err == nil {
		stats["running"] = running
	}

	var totalFailed int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments WHERE status = 'failed'").Scan(&totalFailed); err == nil {
		stats["total_failed"] = totalFailed
	}

	var recentFailed int
	if err := d.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployments WHERE status = 'failed' AND created_at > datetime('now', '-1 day')").
		Scan(&recentFailed); err == nil {
		stats["failed_24h"] = recentFailed
	}

	var recentTotal int
	if err := d.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployments WHERE created_at > datetime('now', '-1 day')").
		Scan(&recentTotal); err == nil {
		stats["total_24h"] = recentTotal
		if recentTotal > 0 {
			stats["failure_rate_24h"] = fmt.Sprintf("%.1f%%", float64(recentFailed)/float64(recentTotal)*100)
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

// DeleteSiteDeployments terminates all running apps and removes all deployment
// artifacts for a given site. It matches by site_id first, then falls back to
// empty-site_id + repo_id for legacy deployments. Unlike the HTTP handler, this
// does not reject active deployments — the caller (sites.deleteSite) has already
// decided the site is being deleted.
func (d *Deploy) DeleteSiteDeployments(ctx context.Context, siteID, repoID string) error {
	rows, err := d.db.QueryContext(ctx,
		`SELECT id, status, COALESCE(url,'') FROM deployments
		 WHERE site_id = ? OR (site_id = '' AND repo_id = ?)`,
		siteID, repoID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	type depRow struct{ id, status, url string }
	var deps []depRow
	for rows.Next() {
		var r depRow
		if err := rows.Scan(&r.id, &r.status, &r.url); err != nil {
			continue
		}
		deps = append(deps, r)
	}

	for _, dep := range deps {
		d.mu.Lock()
		app, hasApp := d.apps[dep.id]
		if hasApp {
			delete(d.apps, dep.id)
		}
		d.mu.Unlock()

		if hasApp {
			if app.cmd != nil && app.cmd.Process != nil {
				_ = app.cmd.Process.Kill()
			}
			if app.server != nil {
				_ = app.server.Close()
			}
			if d.hostRouter != nil {
				d.hostRouter.UnregisterDeploymentHost(app.host)
			}
		}

		_ = os.RemoveAll(filepath.Join(d.buildsDir, dep.id))
		d.deleteDeployLogs(dep.id)
	}

	_, err = d.db.ExecContext(ctx,
		`DELETE FROM deployments WHERE site_id = ? OR (site_id = '' AND repo_id = ?)`,
		siteID, repoID)
	return err
}

var _ kernel.Component = (*Deploy)(nil)
