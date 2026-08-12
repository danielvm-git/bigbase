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
	"syscall"
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
	port, err := pickPort(d.basePort)
	if err != nil {
		return nil, fmt.Errorf("allocate port: %w", err)
	}

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
			killProcessGroup(app.cmd.Process.Pid)
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
		// BigBase session (or survived a crash). Read the PID from DB and kill
		// the entire process tree.
		var pid int
		err := d.db.QueryRowContext(context.Background(),
			"SELECT pid FROM deployments WHERE id = ?", id).Scan(&pid)
		if err == nil && pid > 0 {
			killProcessGroup(pid)
			d.logger.Info("killed orphaned process group", "id", id, "pid", pid)
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

	repoID := filepath.Clean(deploy.RepoID)
	if strings.Contains(repoID, "..") {
		d.logger.Error("invalid repoID", "id", deploy.RepoID)
		d.appendDeployLog(deploy.ID, "✗ Invalid repository ID")
		d.updateStatus(deploy.ID, "failed")
		return
	}
	repoPath := filepath.Join(d.gitDir, repoID+".git")
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

	// Honor sites.root_path (monorepos / pnpm workspaces) for detection + build.
	appRoot := buildDir
	if deploy.SiteID != "" {
		var rootPath string
		_ = d.db.QueryRowContext(context.Background(),
			"SELECT root_path FROM sites WHERE id = ?", deploy.SiteID).Scan(&rootPath)
		appRoot = ResolveAppRoot(buildDir, rootPath)
		if appRoot != buildDir {
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ App root: %s", rootPath))
		}
	}

	// Load deployment manifest if present in the app root.
	manifestPathStr := "bigbase.yaml"
	if deploy.ManifestPath != "" {
		manifestPathStr = deploy.ManifestPath
	}
	manifest, loadErr := LoadManifestPath(appRoot, deploy.ManifestPath)
	if loadErr != nil {
		d.appendDeployLog(deploy.ID, fmt.Sprintf("⚠ Invalid %s: %v — falling back to auto-detection", manifestPathStr, loadErr))
	}

	var appType AppType
	outDirHint := ""
	switch {
	case deploy.AppType != "":
		resolved, outDir, resolveErr := ResolveDeployAppType(appRoot, deploy.AppType)
		if resolveErr != nil {
			d.logger.Error("resolve app type", "error", resolveErr)
			d.appendDeployLog(deploy.ID, "✗ "+resolveErr.Error())
			d.failDeployment(deploy.ID, resolveErr)
			return
		}
		appType = resolved
		outDirHint = outDir
	case manifest != nil:
		appType = manifestToAppType(manifest)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Manifest: framework=%s", manifest.Framework))
	default:
		resolved, outDir, resolveErr := ResolveDeployAppType(appRoot, "")
		if resolveErr != nil {
			d.logger.Error("resolve app type", "error", resolveErr)
			d.appendDeployLog(deploy.ID, "✗ "+resolveErr.Error())
			d.failDeployment(deploy.ID, resolveErr)
			return
		}
		appType = resolved
		outDirHint = outDir
	}
	_, _ = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET app_type = ? WHERE id = ?", string(appType), deploy.ID)
	d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Detected app type: %s", appType))

	// Propagate manifest [security] csp to deploy_defaults so the proxy picks
	// it up at RegisterDeploymentHost without needing to re-parse the manifest.
	if manifest != nil && manifest.Security.CSP != "" {
		d.applyManifestCSP(deploy.SiteID, manifest.Security.CSP)
	}

	// Pure static (index.html present, no package.json build) can serve immediately.
	// Framework-static (Astro/SvelteKit) and other package.json apps still need install+build
	// even when the host model is static — otherwise we serve a source directory listing.
	if appType == AppStatic {
		needsNodeBuild := fileExists(filepath.Join(appRoot, "package.json")) &&
			!fileExists(filepath.Join(appRoot, "index.html"))
		if !needsNodeBuild {
			serveDir, serveErr := ResolvePureStaticServeDir(appRoot, outDirHint)
			if serveErr != nil {
				d.logger.Error("static output missing", "error", serveErr)
				d.appendDeployLog(deploy.ID, "✗ "+serveErr.Error())
				d.failDeployment(deploy.ID, serveErr)
				return
			}
			if serveDir != appRoot {
				rel, relErr := filepath.Rel(appRoot, serveDir)
				if relErr == nil {
					d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Static serve dir: %s", rel))
				}
			}
			d.appendDeployLog(deploy.ID, "→ Serving static files")
			d.updateStatus(deploy.ID, "running")
			d.finalizeDeploymentURL(deploy, repoName)
			d.persistPipelineTimeline(deploy.ID, timeline)
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Deployed at %s", deploy.URL))
			_ = d.RegisterCustomDomainHosts(context.Background(), deploy.SiteID, deploy.Port)
			d.drainOldDeployments(deploy.SiteID)
			go d.serveStatic(context.Background(), serveDir, deploy, repoName)
			return
		}
		d.appendDeployLog(deploy.ID, "→ Static host model requires Node build (package.json without index.html)")
		appType = AppNode
	}

	timeline.BuildStart = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)
	if err := d.buildApp(ctx, deploy.ID, deploy.SiteID, deploy.RepoID, deploy.Branch, appRoot, appType, manifest); err != nil {
		d.logger.Error("build app", "type", appType, "error", err)
		d.persistPipelineTimeline(deploy.ID, timeline)
		d.failDeployment(deploy.ID, err)
		return
	}
	timeline.BuildEnd = timelineNow()
	d.persistPipelineTimeline(deploy.ID, timeline)

	serveDir := appRoot
	if manifest != nil && manifest.Build.Output != "" {
		outputDir := filepath.Join(appRoot, manifest.Build.Output)
		if _, err := os.Stat(outputDir); err == nil {
			serveDir = outputDir
			appType = AppStatic
			deploy.AppType = AppStatic
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET app_type = ? WHERE id = ?", string(AppStatic), deploy.ID)
			d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Serving manifest output: %s", manifest.Build.Output))
		}
	} else if appType == AppNode {
		// Promote to static only when the outDir has a browser entrypoint.
		// adapter-node emits build/ without index.html — keep process AppNode.
		if staticDir, ok := FindStaticServeDirAfterNodeBuild(appRoot, outDirHint); ok {
			serveDir = staticDir
			appType = AppStatic
			deploy.AppType = AppStatic
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET app_type = ? WHERE id = ?", string(AppStatic), deploy.ID)
			if rel, relErr := filepath.Rel(appRoot, staticDir); relErr == nil {
				d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Static serve dir: %s", rel))
			}
		}
	}

	if appType == AppStatic {
		tried := []string{
			filepath.Join(appRoot, "dist"),
			filepath.Join(appRoot, "build"),
			filepath.Join(appRoot, "public"),
			appRoot,
		}
		if outDirHint != "" {
			tried = append([]string{filepath.Join(appRoot, outDirHint)}, tried...)
		}
		if err := RequireStaticIndex(serveDir, tried...); err != nil {
			d.logger.Error("static output missing after build", "error", err, "serveDir", serveDir)
			d.appendDeployLog(deploy.ID, "✗ "+err.Error())
			d.failDeployment(deploy.ID, err)
			return
		}
		if serveDir != appRoot {
			rel, relErr := filepath.Rel(appRoot, serveDir)
			if relErr == nil {
				d.appendDeployLog(deploy.ID, fmt.Sprintf("→ Static serve dir: %s", rel))
			}
		}
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
	cmd := exec.CommandContext(ctx, "git", "clone", "--", repoPath, ".")
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
	// Resolve the build environment through EnvResolver (issue #41): the
	// single owner of precedence (platform defaults → manifest → project →
	// site) and of which keys are secrets (for redacting captured build
	// output). Manifest [env] is a build-scope layer below secrets.
	resolved, err := d.envResolver.ResolveOptions(ctx, siteID, ScopeBuild, ResolveOptions{
		ManifestEnv: manifestEnv(manifest),
	})
	if err != nil {
		return fmt.Errorf("resolve build-time env vars: %w", err)
	}

	// Use manifest build command if present, falling back to auto-detection.
	if manifest != nil && manifest.Build.Command != "" {
		if err := ensureManifestPackageManager(manifest.Build.Command); err != nil {
			return err
		}
		parts := strings.Fields(manifest.Build.Command)
		if len(parts) > 0 {
			return d.runBuildCommand(ctx, deployID, buildDir, resolved, parts[0], parts[1:]...)
		}
	}

	switch appType {
	case AppNode:
		pm := DetectNodePackageManager(buildDir)
		if err := ensureNodePackageManager(pm); err != nil {
			d.appendDeployLog(deployID, "✗ "+err.Error())
			return err
		}
		if err := d.nodeInstall(ctx, deployID, siteID, repoID, branch, buildDir); err != nil {
			return err
		}
		if err := ValidateNodeBuildScript(buildDir); err != nil {
			d.appendDeployLog(deployID, "✗ "+err.Error())
			return err
		}
		name, args := NodeBuildCommand(pm)
		if err := d.runBuildCommand(ctx, deployID, buildDir, resolved, name, args...); err != nil {
			return fmt.Errorf("%s run build: %w", pm, err)
		}
		return nil
	case AppGo:
		if _, err := exec.LookPath("go"); err != nil {
			return codedErr("tool_missing", "go not found on PATH",
				"Install Go on the deploy host (matching go.mod) before deploying AppType=go.")
		}
		return d.runBuildCommand(ctx, deployID, buildDir, resolved, "go", "build", "-o", "app", ".")
	case AppPHP:
		if _, err := exec.LookPath("composer"); err != nil {
			return codedErr("tool_missing", "composer not found on PATH",
				"Install Composer (and PHP) on the deploy host before deploying AppType=php.")
		}
		if _, err := exec.LookPath("php"); err != nil {
			return codedErr("tool_missing", "php not found on PATH",
				"Install PHP on the deploy host before deploying AppType=php.")
		}
		return d.runBuildCommand(ctx, deployID, buildDir, resolved, "composer", "install", "--no-dev", "--optimize-autoloader")
	case AppPython:
		if _, err := exec.LookPath("python3"); err != nil {
			if _, err2 := exec.LookPath("python"); err2 != nil {
				return codedErr("tool_missing", "python3 not found on PATH",
					"Install Python 3 on the deploy host before deploying AppType=python.")
			}
		}
		if HasPyProjectTOML(buildDir) {
			pp := ParsePyProjectTOML(buildDir)
			if pp != nil {
				if deps := pp.SystemDeps(); len(deps) > 0 {
					args := append([]string{"install", "-y"}, deps...)
					if err := d.runBuildCommand(ctx, deployID, buildDir, resolved, "apt-get", "update"); err != nil {
						d.appendDeployLog(deployID, "⚠ apt-get update failed — continuing without system deps")
					} else {
						if err := d.runBuildCommand(ctx, deployID, buildDir, resolved, "apt-get", args...); err != nil {
							d.appendDeployLog(deployID, "⚠ apt-get install failed — continuing")
						}
					}
				}
			}
			if _, lookErr := exec.LookPath("uv"); lookErr == nil {
				return d.runBuildCommand(ctx, deployID, buildDir, resolved, "uv", "sync", "--frozen")
			}
			d.appendDeployLog(deployID, "→ uv not found, falling back to pip install")
			return d.runBuildCommand(ctx, deployID, buildDir, resolved, "pip", "install", "--break-system-packages", ".")
		}
		return d.runBuildCommand(ctx, deployID, buildDir, resolved, "pip", "install", "--break-system-packages", "-r", "requirements.txt")
	}
	return nil
}
func (d *Deploy) nodeInstall(ctx context.Context, deployID, siteID, repoID, branch, buildDir string) error {
	pm := DetectNodePackageManager(buildDir)
	if err := ensureNodePackageManager(pm); err != nil {
		return err
	}
	key, keyErr := CacheKey(buildDir, repoID, branch)
	if keyErr == nil && d.restoreNodeModules(deployID, key, buildDir) {
		return nil
	}
	name, args := NodeInstallCommand(pm)
	if err := d.runBuildCommand(ctx, deployID, buildDir, nil, name, args...); err != nil {
		return fmt.Errorf("%s install: %w", pm, err)
	}
	if keyErr == nil {
		d.saveNodeModules(deployID, siteID, repoID, branch, buildDir, key)
	}
	return nil
}
func (d *Deploy) restoreNodeModules(deployID, key, buildDir string) bool {
	hit, err := d.cache.Restore(key, buildDir)
	if err != nil {
		d.appendDeployLog(deployID, fmt.Sprintf("⚠ Cache restore failed: %v — running install", err))
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
func (d *Deploy) runBuildCommand(ctx context.Context, deployID, dir string, resolved *ResolvedEnv, name string, args ...string) error {
	label := FormatBuildCommand(name, args...)
	d.appendDeployLog(deployID, "→ Running: "+label)

	var stderr, stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	// resolved.Environ is the merged build environment from EnvResolver
	// (platform defaults overlaid with site env vars). ResolvedEnv is also
	// the redaction context for any captured tool output (issue #41): a build
	// tool that echoes its own environment must not leak a secret value into
	// the deploy log.
	var extraEnv []string
	if resolved != nil {
		extraEnv = resolved.Environ
	}
	cmd.Env = append(d.buildCmdEnv(), extraEnv...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if resolved != nil {
			detail = RedactLogText(detail, resolved)
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
		if err := ensureManifestPackageManager(manifest.Start.Command); err != nil {
			d.logger.Error("node package manager", "error", err)
			d.updateStatus(deploy.ID, "failed")
			return
		}
		parts := strings.Fields(manifest.Start.Command)
		if len(parts) > 0 {
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
			cmd.Dir = buildDir
		}
	} else {
		switch appType {
		case AppNode:
			pm := DetectNodePackageManager(buildDir)
			if err := ensureNodePackageManager(pm); err != nil {
				d.logger.Error("node package manager", "error", err)
				d.updateStatus(deploy.ID, "failed")
				return
			}
			name, args := NodeStartCommand(buildDir)
			cmd = exec.CommandContext(ctx, name, args...)
			cmd.Dir = buildDir
		case AppGo:
			cmd = exec.CommandContext(ctx, filepath.Join(buildDir, "app"))
			cmd.Dir = buildDir
		case AppPython:
			cmd = pythonStartCommand(ctx, buildDir, deploy.Port, manifest)
		case AppPHP:
			docRoot := buildDir
			if st, err := os.Stat(filepath.Join(buildDir, "public")); err == nil && st.IsDir() {
				docRoot = filepath.Join(buildDir, "public")
			}
			cmd = exec.CommandContext(ctx, "php", "-S", fmt.Sprintf("0.0.0.0:%d", deploy.Port), "-t", docRoot)
			cmd.Dir = buildDir
		}
	}

	if cmd == nil {
		d.logger.Error("no start command for", "type", appType)
		d.updateStatus(deploy.ID, "failed")
		return
	}

	// Put the child in its own process group so Stop can kill the entire tree
	// (npm exec → node, python → uvicorn, etc.) without leaking orphans.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Reserved runtime values are owned by the platform and always win: the
	// allocated port, the writable runtime directory, and the native DB
	// connection. They are applied as the final resolution layer so neither a
	// manifest value nor a secret can hijack them.
	reserved := []string{fmt.Sprintf("PORT=%d", deploy.Port)}

	// Create and inject writable persistent directory for runtime data.
	writableDir := filepath.Join(d.buildsDir, "..", "writable", deploy.ID)
	if err := os.MkdirAll(writableDir, 0755); err != nil {
		d.logger.Warn("create writable dir", "deployID", deploy.ID)
	} else {
		reserved = append(reserved, "WRITABLE_DIR="+writableDir)
	}

	// Inject native DB connection (DB_PATH for SQLite, DATABASE_URL for Postgres).
	reserved = append(reserved, d.nativeDBEnv()...)

	// Resolve the complete runtime environment through the single resolver:
	// platform baseline → manifest config → Project secrets → Site
	// compatibility values → reserved runtime values. Manifest env must NOT be
	// appended after resolution — that would let repo config override secrets.
	resolvedRuntime, err := d.envResolver.ResolveOptions(ctx, deploy.SiteID, ScopeRuntime, ResolveOptions{
		ManifestEnv: manifestEnv(manifest),
		Reserved:    reserved,
	})
	if err != nil {
		d.logger.Error("resolve runtime environment failed", "deployment_id", deploy.ID, "error", err)
		_, _ = d.db.ExecContext(context.Background(), "UPDATE deployments SET error_message = ? WHERE id = ?", err.Error(), deploy.ID)
		d.updateStatus(deploy.ID, "failed")
		return
	}
	// resolvedRuntime.Environ already contains the platform baseline; replacing
	// cmd.Env wholesale (instead of appending over os.Environ) keeps precedence
	// deterministic and removes duplicate entries.
	cmd.Env = resolvedRuntime.Environ

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	host := deploymentHost(d.publicDomain, repoName)

	// Start the child BEFORE publishing it in d.apps: Stop/drain only ever see
	// apps whose Process is already populated, closing the cmd.Start vs
	// cmd.Process.Pid read/write race under -race.
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

	d.mu.Lock()
	d.apps[deploy.ID] = &runningApp{cmd: cmd, port: deploy.Port, buildID: deploy.ID, host: host}
	d.mu.Unlock()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if resolvedRuntime != nil {
				line = RedactLogText(line, resolvedRuntime)
			}
			d.appendDeployLog(deploy.ID, "[runtime] "+line)
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if resolvedRuntime != nil {
				line = RedactLogText(line, resolvedRuntime)
			}
			d.appendDeployLog(deploy.ID, "[runtime] "+line)
		}
	}()

	if err := cmd.Wait(); err != nil {
		d.logger.Error("app exited", "id", deploy.ID, "error", err)
		// Intentional stops remove the app from d.apps (Stop/drain/rollback)
		// or record a terminal DB status. In both cases the stop path owns the
		// final status; do not race it with "failed".
		if d.hasTerminalStopStatus(deploy.ID) || !d.hasRegisteredApp(deploy.ID) {
			return
		}
		_, _ = d.db.ExecContext(context.Background(), "UPDATE deployments SET error_message = ? WHERE id = ?", err.Error(), deploy.ID)
		d.appendDeployLog(deploy.ID, fmt.Sprintf("✗ App exited: %v", err))
		d.updateStatus(deploy.ID, "failed")
		// Do NOT unregister the host here — the host may be shared with another
		// running deployment for the same site. Host cleanup is handled by
		// stopDeployment and drainDeployment.
	}
}

// hasTerminalStopStatus reports whether the deployment was already recorded as
// intentionally stopped or replaced by a stop/drain/rollback path.
func (d *Deploy) hasTerminalStopStatus(id string) bool {
	var status string
	if err := d.db.QueryRowContext(context.Background(),
		"SELECT status FROM deployments WHERE id = ?", id).Scan(&status); err != nil {
		return false
	}
	return status == "stopped" || status == "replaced"
}

// hasRegisteredApp reports whether the deployment is still in the running-app
// registry. Stop, drain, and rollback remove it before/after killing the
// process, which is the intentional-stop signal for the Wait handler.
func (d *Deploy) hasRegisteredApp(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.apps[id]
	return ok
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

// applyManifestCSP writes a manifest-declared CSP to deploy_defaults.csp_policy
// so it persists across proxy restarts and is picked up by RegisterDeploymentHost.
func (d *Deploy) applyManifestCSP(siteID, csp string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var ddStr string
	_ = d.db.QueryRowContext(ctx, "SELECT deploy_defaults FROM sites WHERE id = ?", siteID).Scan(&ddStr)

	var dd SiteDefaults
	if ddStr != "" && ddStr != "{}" {
		_ = json.Unmarshal([]byte(ddStr), &dd)
	}
	dd.CSPPolicy = csp

	ddBytes, err := json.Marshal(dd)
	if err != nil {
		return
	}
	_, _ = d.db.ExecContext(ctx, "UPDATE sites SET deploy_defaults = ? WHERE id = ?", string(ddBytes), siteID)
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
