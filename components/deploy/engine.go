package deploy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (d *Deploy) Trigger(ctx context.Context, repoID, branch, siteName, siteID string, passthroughPaths []string, appType string, manifestPath string) (*Deployment, error) {
	if repoID == "" {
		return nil, fmt.Errorf("repo_id is required")
	}
	if branch == "" {
		branch = "main"
	}

	var repoName string
	err := d.db.QueryRowContext(ctx, "SELECT name FROM git_repos WHERE id = ?", repoID).Scan(&repoName)
	if err != nil {
		return nil, ErrRepoNotFound
	}

	if siteName == "" {
		siteName = repoName
	}

	// Collect previous running deployments for zero-downtime drain.
	// Old deployments are NOT stopped here — they continue serving existing
	// connections until the new deployment passes health check, at which point
	// drainOldDeployments() signals them to drain gracefully.
	d.collectPreviousDeployments(ctx, siteID, repoID)

	id, err := kernel.GenerateID()
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
		ManifestPath:     manifestPath,
		URL:              deploymentURL(d.publicDomain, d.useHTTPS, siteName, port),
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	if _, err := d.db.ExecContext(ctx,
		"INSERT INTO deployments (id, repo_id, site_id, branch, status, port, url, passthrough_paths, manifest_path, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, repoID, siteID, branch, deploy.Status, deploy.Port, deploy.URL, passthroughJSON, deploy.ManifestPath, deploy.CreatedAt); err != nil {
		return nil, err
	}

	d.initDeployLogs(id)
	d.appendDeployLog(id, fmt.Sprintf("→ Deployment started (branch: %s)", branch))

	// After finalizeDeploymentURL runs, the proxy will route to the new port.
	// Old deployments are drained after health check passes — zero-downtime.
	go d.runDeployment(deploy, buildDir, siteName)
	return deploy, nil
}
func (d *Deploy) stopDeployment(id, newStatus string) {
	// Signal the Supervisor so any respawn loop knows this is intentional.
	if d.supervisor != nil {
		d.supervisor.Stop(id)
	}

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
func (d *Deploy) runDeployment(deploy *Deployment, buildDir, repoName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	timeline := &PipelineTimeline{}

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
	timeline.CloneStart = timelineNow()
	if err := d.cloneAndCheckout(ctx, deploy.ID, repoPath, buildDir, deploy.Branch); err != nil {
		d.logger.Error("clone repo", "error", err)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ Clone failed: %v", err))
		timeline.CloneEnd = timelineNow()
		d.persistPipelineTimeline(deploy.ID, timeline)
		d.updateStatus(deploy.ID, "failed")
		return
	}
	timeline.CloneEnd = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)
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

	// Load deployment manifest if present in the repo root.
	manifestPathStr := "bigbase.yaml"
	if deploy.ManifestPath != "" {
		manifestPathStr = deploy.ManifestPath
	}
	manifest, loadErr := LoadManifestPath(buildDir, deploy.ManifestPath)
	if loadErr != nil {
		d.appendDeployLog(deploy.ID, fmt.Sprintf("⚠ Invalid %s: %v — falling back to auto-detection", manifestPathStr, loadErr))
	}

	appType := deploy.AppType
	if appType == "" {
		if manifest != nil {
			appType = manifestToAppType(manifest)
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Manifest: framework=%s", manifest.Framework))
		} else {
			appType = DetectAppType(buildDir)
		}
	}
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET app_type = ? WHERE id = ?", string(appType), deploy.ID)
	d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Detected app type: %s", appType))

	if appType == AppStatic {
		d.appendDeployLog(deploy.ID, "→ Serving static files")
		d.updateStatus(deploy.ID, "running")
		d.finalizeDeploymentURL(deploy, repoName)
		d.persistPipelineTimeline(deploy.ID, timeline)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Deployed at %s", deploy.URL))
		_ = d.RegisterCustomDomainHosts(context.Background(), deploy.SiteID, deploy.Port)
		// Drain old deployments now that new host is registered
		d.drainOldDeployments(deploy.SiteID)
		go d.serveStatic(context.Background(), buildDir, deploy, repoName)
		return
	}

	timeline.BuildStart = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)
	if err := d.buildApp(ctx, deploy.ID, deploy.SiteID, deploy.RepoID, deploy.Branch, buildDir, appType, manifest); err != nil {
		d.logger.Error("build app", "type", appType, "error", err)
		d.persistPipelineTimeline(deploy.ID, timeline)
		d.failDeployment(deploy.ID, err)
		return
	}
	timeline.BuildEnd = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)

	serveDir := buildDir
	if manifest != nil && manifest.Build.Output != "" {
		outputDir := filepath.Join(buildDir, manifest.Build.Output)
		if _, err := os.Stat(outputDir); err == nil {
			serveDir = outputDir
			appType = AppStatic
			deploy.AppType = AppStatic
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET app_type = ? WHERE id = ?", string(AppStatic), deploy.ID)
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Serving manifest output: %s", manifest.Build.Output))
		}
	} else if appType == AppNode {
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

	if appType == AppStatic {
		d.updateStatus(deploy.ID, "running")
		d.finalizeDeploymentURL(deploy, repoName)
		d.persistPipelineTimeline(deploy.ID, timeline)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Deployed at %s", deploy.URL))
		d.appendDeployLog(deploy.ID, "→ Serving static files")
		_ = d.RegisterCustomDomainHosts(context.Background(), deploy.SiteID, deploy.Port)
		d.drainOldDeployments(deploy.SiteID)
		go d.serveStatic(context.Background(), serveDir, deploy, repoName)
		return
	}

	// Process app: transition to deploying, start the app, then health-probe
	// before marking running and registering the proxy host.
	timeline.StartStart = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)
	_ = d.TransitionState(ctx, deploy.ID, "deploying")
	d.appendDeployLog(deploy.ID, "→ Status: deploying — starting application")
	go d.startApp(context.Background(), serveDir, deploy, appType, repoName, manifest)
	timeline.StartEnd = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)

	timeline.HealthStart = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)
	result := d.runHealthCheck(ctx, deploy, manifest)
	timeline.HealthEnd = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)
	if result.OK {
		_ = d.TransitionState(ctx, deploy.ID, "running")
		d.finalizeDeploymentURL(deploy, repoName)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Deployed at %s", deploy.URL))
		_ = d.RegisterCustomDomainHosts(context.Background(), deploy.SiteID, deploy.Port)
		// Drain old deployments now that new host is registered
		d.drainOldDeployments(deploy.SiteID)
	} else {
		d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ Health check failed after %d attempts: %s",
			result.Attempts, result.FirstFailureReason))
		_ = d.TransitionState(context.Background(), deploy.ID, "failed")
		_, _ = d.db.ExecContext(context.Background(),
			"UPDATE deployments SET error_message = ? WHERE id = ?",
			fmt.Sprintf("Health check failed: %s", result.FirstFailureReason), deploy.ID)
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
func (d *Deploy) buildApp(ctx context.Context, deployID, siteID, repoID, branch, buildDir string, appType AppType, manifest *Manifest) error {
	siteEnv, err := d.FetchSiteEnvVars(ctx, siteID, true)
	if err != nil {
		return fmt.Errorf("fetch build-time env vars: %w", err)
	}

	// Use manifest build command if present, falling back to auto-detection.
	if manifest != nil && manifest.Build.Command != "" {
		parts := strings.Fields(manifest.Build.Command)
		if len(parts) > 0 {
			return d.runBuildCommand(ctx, deployID, buildDir, siteEnv, parts[0], parts[1:]...)
		}
	}

	switch appType {
	case AppNode:
		if err := d.nodeInstall(ctx, deployID, siteID, repoID, branch, buildDir); err != nil {
			return err
		}
		if err := ValidateNodeBuildScript(buildDir); err != nil {
			d.appendDeployLog(deployID, "✗ "+err.Error())
			return err
		}
		if err := d.runBuildCommand(ctx, deployID, buildDir, siteEnv, "npm", "run", "build"); err != nil {
			return fmt.Errorf("npm run build: %w", err)
		}
		return nil
	case AppGo:
		return d.runBuildCommand(ctx, deployID, buildDir, siteEnv, "go", "build", "-o", "app", ".")
	case AppPython:
		if HasPyProjectTOML(buildDir) {
			pp := ParsePyProjectTOML(buildDir)
			if pp != nil {
				if deps := pp.SystemDeps(); len(deps) > 0 {
					args := append([]string{"install", "-y"}, deps...)
					if err := d.runBuildCommand(ctx, deployID, buildDir, siteEnv, "apt-get", "update"); err != nil {
						d.appendDeployLog(deployID, "⚠ apt-get update failed — continuing without system deps")
					} else {
						if err := d.runBuildCommand(ctx, deployID, buildDir, siteEnv, "apt-get", args...); err != nil {
							d.appendDeployLog(deployID, "⚠ apt-get install failed — continuing")
						}
					}
				}
			}
			return d.runBuildCommand(ctx, deployID, buildDir, siteEnv, "uv", "sync", "--frozen")
		}
		return d.runBuildCommand(ctx, deployID, buildDir, siteEnv, "pip", "install", "--break-system-packages", "-r", "requirements.txt")
	}
	return nil
}
func (d *Deploy) nodeInstall(ctx context.Context, deployID, siteID, repoID, branch, buildDir string) error {
	key, keyErr := CacheKey(buildDir, repoID, branch)
	if keyErr == nil && d.restoreNodeModules(deployID, key, buildDir) {
		return nil
	}
	if err := d.runBuildCommand(ctx, deployID, buildDir, nil, "npm", "install"); err != nil {
		return fmt.Errorf("npm install: %w", err)
	}
	if keyErr == nil {
		d.saveNodeModules(deployID, siteID, repoID, branch, buildDir, key)
	}
	return nil
}
func (d *Deploy) restoreNodeModules(deployID, key, buildDir string) bool {
	hit, err := d.cache.Restore(key, buildDir)
	if err != nil {
		d.appendDeployLog(deployID, fmt.Sprintf("⚠ Cache restore failed: %v — running npm install", err))
		return false
	}
	if hit {
		d.appendDeployLog(deployID, "→ Cache hit: restored node_modules")
	}
	return hit
}
func (d *Deploy) saveNodeModules(deployID, siteID, repoID, branch, buildDir, key string) {
	if err := d.cache.Save(key, buildDir, siteID, repoID, branch, []string{"node_modules"}); err != nil {
		d.appendDeployLog(deployID, fmt.Sprintf("⚠ Cache save failed: %v", err))
		return
	}
	d.appendDeployLog(deployID, "→ Cache saved: node_modules")
	_ = d.cache.Evict()
}
func (d *Deploy) runBuildCommand(ctx context.Context, deployID, dir string, extraEnv []string, name string, args ...string) error {
	label := FormatBuildCommand(name, args...)
	d.appendDeployLog(deployID, "→ Running: "+label)

	var stderr, stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(d.buildCmdEnv(), extraEnv...)
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
func (d *Deploy) startApp(ctx context.Context, buildDir string, deploy *Deployment, appType AppType, repoName string, manifest *Manifest) {
	var cmd *exec.Cmd

	// Use manifest start command if present, falling back to auto-detection.
	if manifest != nil && manifest.Start.Command != "" {
		parts := strings.Fields(manifest.Start.Command)
		if len(parts) > 0 {
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
			cmd.Dir = buildDir
		}
	} else {
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
			cmd = pythonStartCommand(ctx, buildDir)
		}
	}

	if cmd == nil {
		d.logger.Error("no start command for", "type", appType)
		d.updateStatus(deploy.ID, "failed")
		return
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", deploy.Port))

	// Create and inject writable persistent directory for runtime data.
	writableDir := filepath.Join(d.buildsDir, "..", "writable", deploy.ID)
	if err := os.MkdirAll(writableDir, 0755); err != nil {
		d.logger.Warn("create writable dir", "deployID", deploy.ID, "error", err)
	} else {
		cmd.Env = append(cmd.Env, "WRITABLE_DIR="+writableDir)
	}

	// Inject native DB connection (DB_PATH for SQLite, DATABASE_URL for Postgres).
	if native := d.nativeDBEnv(); len(native) > 0 {
		cmd.Env = append(cmd.Env, native...)
	}

	// Inject site env vars (runtime) into the running process. A fetch failure
	// must not be silent: the app would boot without its secrets (F09).
	if siteEnv, err := d.FetchSiteEnvVars(ctx, deploy.SiteID, false); err != nil {
		d.logger.Warn("fetch runtime env vars — app starting without site env", "site", deploy.SiteID, "error", err)
	} else {
		cmd.Env = append(cmd.Env, siteEnv...)
	}

	// Inject manifest environment variables into the running process.
	if manifest != nil {
		for k, v := range manifest.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

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
		// Do NOT unregister the host here — the host may be shared with another
		// running deployment for the same site. Host cleanup is handled by
		// stopDeployment and drainDeployment.
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
func (d *Deploy) runHealthCheck(ctx context.Context, deploy *Deployment, manifest *Manifest) HealthResult {
	cfg := ManifestHealthCheck{}.WithDefaults()
	if manifest != nil {
		cfg = manifest.HealthCheck.WithDefaults()
	}
	baseURL := fmt.Sprintf("http://localhost:%d", deploy.Port)
	result := probeHealth(ctx, http.DefaultClient, baseURL, cfg, &wallClock{})

	// Log each probe attempt
	for i, probe := range result.Probes {
		attempt := i + 1
		if probe.Err != "" {
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Health [%d/%d]: GET %s → ERROR %s (%dms)",
				attempt, cfg.MaxRetries, cfg.Path, probe.Err, probe.DurationMS))
		} else {
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Health [%d/%d]: GET %s → %d (%dms)",
				attempt, cfg.MaxRetries, cfg.Path, probe.Status, probe.DurationMS))
		}
	}

	// On final failure, append the failure response body if available
	if !result.OK && len(result.Probes) > 0 {
		lastProbe := result.Probes[len(result.Probes)-1]
		if lastProbe.Err != "" {
			d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ Health check failed: %s", result.FirstFailureReason))
		}
	}

	// Persist health_summary JSON
	summaryJSON, _ := json.Marshal(HealthSummary{
		ProbeCount:         result.Attempts,
		AvgResponseTimeMs:  result.AvgResponseTimeMS,
		FirstFailureReason: result.FirstFailureReason,
	})
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET health_summary = ? WHERE id = ?",
		string(summaryJSON), deploy.ID)

	return result
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

	// Use TransitionState for validated status change
	_ = d.TransitionState(context.Background(), id, "failed")

	// Update error_message (separate from TransitionState to keep it focused on status)
	d.mu.Lock()
	defer d.mu.Unlock()
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET error_message = ? WHERE id = ?", msg, id)
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
func (d *Deploy) buildMetadata(deploy *Deployment) map[string]string {
	m := map[string]string{
		"deployedAt": time.Now().UTC().Format(time.RFC3339),
	}
	if deploy.CommitSHA != "" {
		m["version"] = deploy.CommitSHA
	}
	return m
}
