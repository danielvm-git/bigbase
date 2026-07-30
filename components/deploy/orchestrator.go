package deploy

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
)

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
			// Promote to static only when an output dir genuinely contains a browser
			// entrypoint (index.html). SvelteKit adapter-node (and other SSR adapters)
			// emit build/ with a Node server entry (build/index.js) and no index.html —
			// those must be resumed as AppNode process apps, not FileServer'd as static
			// (issue #181). Mirrors the deploy-time check in engine.go.
			if staticDir, ok := FindStaticServeDirAfterNodeBuild(buildDir, ""); ok {
				serveDir = staticDir
				appType = AppStatic
			}
			// Node SSR apps (build/ without index.html, or no dist/build) fall through to
			// the process resume below.
		}
		if appType == AppStatic {
			resolved, serveErr := ResolvePureStaticServeDir(buildDir, "dist")
			if serveErr != nil {
				resolved, serveErr = ResolvePureStaticServeDir(buildDir, "build")
			}
			if serveErr != nil {
				// Do not resume a checkout FileServer listing — leave host down until redeploy.
				d.logger.Error("skip resume: static output missing", "id", c.id, "error", serveErr)
				_, _ = d.db.ExecContext(context.Background(),
					"UPDATE deployments SET status = ?, error_message = ? WHERE id = ?",
					"failed", serveErr.Error(), c.id)
				continue
			}
			serveDir = resolved
			host := deploymentHost(d.publicDomain, c.repoName)
			spec := Spec{
				DeployID: c.id,
				Host:     host,
				Port:     c.port,
				Dir:      serveDir,
				AppType:  AppStatic,
				Env:      d.buildCmdEnv(),
			}
			d.mu.Lock()
			d.apps[c.id] = &runningApp{port: c.port, buildID: c.id, host: host}
			d.mu.Unlock()
			d.logger.Info("resumed static deployment via supervisor", "id", c.id, "port", c.port, "host", host)
			go d.supervisor.Run(context.Background(), spec)
			continue
		}
		// Resume process-based apps (Python, Go, Node SSR).
		// These are killed when BigBase stops; restart them from the existing build directory.
		// Supervisor wiring for process apps is done in e53s05 (after crash-loop
		// behavior is validated against the existing TestResumeCandidatesAttemptsProcessApps contract).
		deploy := &Deployment{
			ID:      c.id,
			RepoID:  c.repoID,
			SiteID:  c.siteID,
			Port:    c.port,
			URL:     c.rawURL,
			AppType: appType,
		}
		manifest, _ := LoadManifest(serveDir) // best-effort: load if present
		go d.startApp(context.Background(), serveDir, deploy, appType, c.repoName, manifest)
		d.logger.Info("resumed process deployment", "id", c.id, "appType", string(appType), "port", c.port, "host", HostFromDeploymentURL(c.rawURL))
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
func (d *Deploy) collectPreviousDeployments(ctx context.Context, siteID, repoID string) {
	if siteID == "" && repoID == "" {
		return
	}
	var query string
	var args []any
	if siteID != "" {
		query = "SELECT id FROM deployments WHERE site_id = ? AND status IN ('running')"
		args = []any{siteID}
	} else {
		query = "SELECT id FROM deployments WHERE repo_id = ? AND status IN ('running')"
		args = []any{repoID}
	}
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		d.logger.Warn("collect previous deployments", "error", err)
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

	if len(ids) == 0 {
		return
	}

	d.oldDeploymentsMu.Lock()
	d.oldDeployments[siteID] = append(d.oldDeployments[siteID], ids...)
	d.oldDeploymentsMu.Unlock()
	d.logger.Info("collected previous deployments for drain", "site_id", siteID, "count", len(ids))
}
func (d *Deploy) drainOldDeployments(siteID string) {
	d.oldDeploymentsMu.Lock()
	ids := d.oldDeployments[siteID]
	delete(d.oldDeployments, siteID)
	d.oldDeploymentsMu.Unlock()

	for _, id := range ids {
		d.logger.Info("draining old deployment", "id", id)
		go d.drainDeployment(id)
	}
}
func (d *Deploy) drainDeployment(id string) {
	d.mu.Lock()
	app, hasApp := d.apps[id]
	d.mu.Unlock()

	if !hasApp {
		d.logger.Warn("drain: no running app found", "id", id)
		// Try to stop via DB (orphaned process handling)
		d.stopDeployment(id, "stopped")
		return
	}

	// Transition to draining
	ctx := context.Background()
	if err := d.TransitionState(ctx, id, "draining"); err != nil {
		d.logger.Warn("drain: transition to draining failed", "id", id, "error", err)
	}

	// Signal the Supervisor so any respawn loop knows this is intentional.
	if d.supervisor != nil {
		d.supervisor.Stop(id)
	}

	// Create a context with the drain timeout
	drainCtx, cancel := context.WithTimeout(ctx, d.DrainTimeout)
	defer cancel()

	drained := make(chan struct{}, 1)

	if app.server != nil {
		// Static server: use Go http.Server.Shutdown which handles graceful drain
		go func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(ctx, d.DrainTimeout)
			defer shutdownCancel()
			if err := app.server.Shutdown(shutdownCtx); err != nil {
				d.logger.Warn("drain: static server shutdown timed out", "id", id, "error", err)
			}
			drained <- struct{}{}
		}()
	} else if app.cmd != nil && app.cmd.Process != nil {
		// Process app: send SIGTERM, wait for graceful exit
		d.logger.Info("drain: sending SIGTERM to process", "id", id, "pid", app.cmd.Process.Pid)
		if err := app.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			d.logger.Warn("drain: SIGTERM failed, falling back to kill", "id", id, "error", err)
			_ = app.cmd.Process.Kill()
			drained <- struct{}{}
		} else {
			go func() {
				_, waitErr := app.cmd.Process.Wait()
				if waitErr != nil {
					d.logger.Debug("drain: process wait done", "id", id, "error", waitErr)
				}
				drained <- struct{}{}
			}()
		}
	} else {
		// No server or cmd — mark as drained immediately
		drained <- struct{}{}
	}

	// Wait for drain or timeout
	select {
	case <-drained:
		d.logger.Info("drain: completed gracefully", "id", id)
	case <-drainCtx.Done():
		d.logger.Warn("drain: timeout — force killing", "id", id, "timeout", d.DrainTimeout)
		// Force kill the entire process tree, not just the main process.
		// Process.Kill() only sends SIGKILL to the parent — child processes
		// (Python workers, Telegram polling threads) survive as orphans.
		if app.cmd != nil && app.cmd.Process != nil {
			killProcessGroup(app.cmd.Process.Pid)
		}
		if app.server != nil {
			_ = app.server.Close()
		}
	}

	// Do NOT unregister the host — the new deployment already registered on the
	// same host via finalizeDeploymentURL. Unregistering here would delete the
	// new deployment's host mapping, breaking traffic routing.

	// Remove from apps map and transition to stopped
	d.mu.Lock()
	delete(d.apps, id)
	d.mu.Unlock()

	_ = d.TransitionState(ctx, id, string(StateStopped))
	d.logger.Info("drain: deployment stopped", "id", id)
}
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
