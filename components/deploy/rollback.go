package deploy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// RollbackEvent records a single rollback operation.
type RollbackEvent struct {
	ID             string `json:"id"`
	SiteID         string `json:"site_id"`
	RolledBackFrom string `json:"rolled_back_from"`
	RolledBackTo   string `json:"rolled_back_to"`
	CreatedAt      string `json:"created_at"`
}

// ensureRollbackEventsTable creates the rollback_events table if it doesn't exist.
func (d *Deploy) ensureRollbackEventsTable() error {
	return d.db.Migrate(`CREATE TABLE IF NOT EXISTS rollback_events (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		rolled_back_from TEXT NOT NULL,
		rolled_back_to TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
}

// Rollback rolls back the deployment identified by currentID to the most recent
// previous successful deployment for the same site. It stops the current app,
// restores the old deployment's build artifacts, updates proxy routing, and
// records a RollbackEvent.
func (d *Deploy) Rollback(ctx context.Context, currentID string) (*RollbackEvent, error) {
	// Fetch current deployment
	var currentSiteID, currentStatus string
	err := d.db.QueryRowContext(ctx,
		"SELECT site_id, status FROM deployments WHERE id = ?", currentID).
		Scan(&currentSiteID, &currentStatus)
	if err != nil {
		return nil, ErrDeploymentNotFound
	}
	if currentSiteID == "" {
		return nil, fmt.Errorf("deployment has no site_id — cannot determine rollback target")
	}
	if currentStatus != string(StateRunning) {
		return nil, fmt.Errorf("can only rollback a running deployment (current status: %s)", currentStatus)
	}

	// Fetch the most recent successful deployment for the same site.
	// "Successful" excludes failed builds and every in-flight state
	// (pending, building, deploying, draining) — the exclusion list comes from
	// the state machine so it cannot drift (#304).
	type prevRow struct {
		id, status, appType, url, passthroughJSON, commitSHA string
		port                                                 int
	}
	excluded := d.sm.InFlightStatuses()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(excluded)), ",")
	query := `SELECT id, status, port, app_type, COALESCE(url,''), COALESCE(passthrough_paths,''), COALESCE(commit_sha,'')
		 FROM deployments
		 WHERE site_id = ? AND id != ? AND status != ? AND status NOT IN (` + placeholders + `)
		 ORDER BY created_at DESC LIMIT 1`
	args := make([]any, 0, 3+len(excluded))
	args = append(args, currentSiteID, currentID, string(StateFailed))
	for _, status := range excluded {
		args = append(args, status)
	}
	var prev prevRow
	err = d.db.QueryRowContext(ctx, query, args...).Scan(&prev.id, &prev.status, &prev.port, &prev.appType, &prev.url, &prev.passthroughJSON, &prev.commitSHA)
	if err != nil {
		return nil, fmt.Errorf("no previous successful deployment found for site %s", currentSiteID)
	}

	// Verify build artifacts still exist on disk
	buildDir := filepath.Join(d.buildsDir, prev.id)
	if _, err := os.Stat(buildDir); err != nil {
		return nil, fmt.Errorf("previous deployment build artifacts not found at %s", buildDir)
	}

	// Stop the current deployment and mark it as rolled_back via TransitionState
	d.stopDeploymentWithTransition(ctx, currentID, string(StateRolledBack))
	d.appendDeployLog(currentID, fmt.Sprintf("→ Rolled back — replacing with deployment %s", prev.id[:min(8, len(prev.id))]))

	// Restore the old deployment through the state machine. A drained
	// deployment is stopped; stopped → running is the documented rollback
	// transition (see StateMachine), so no lenient/unknown-state path is used.
	d.updateStatus(prev.id, string(StateRunning))
	d.appendDeployLog(prev.id, "→ Rolled back — deployment restored as live")

	// Re-register the host with the old deployment's port
	passthroughPaths := parsePassthroughPaths(prev.passthroughJSON)
	host := HostFromDeploymentURL(prev.url)
	if host != "" && d.hostRouter != nil {
		metadata := map[string]string{
			"deployedAt": time.Now().UTC().Format(time.RFC3339),
			"rolledBack": "true",
		}
		if prev.commitSHA != "" {
			metadata["version"] = prev.commitSHA
		}
		_ = d.hostRouter.RegisterDeploymentHost(host, prev.port, currentSiteID, passthroughPaths, metadata)
	}

	// Create the old deployment struct and start serving
	oldDeploy := &Deployment{
		ID:               prev.id,
		SiteID:           currentSiteID,
		Port:             prev.port,
		AppType:          AppType(prev.appType),
		URL:              prev.url,
		PassthroughPaths: passthroughPaths,
	}

	// Start serving the old build artifacts (static or process app)
	serveDir := buildDir
	if oldDeploy.AppType == AppStatic {
		resolved, serveErr := ResolvePureStaticServeDir(buildDir, "dist")
		if serveErr != nil {
			resolved, serveErr = ResolvePureStaticServeDir(buildDir, "build")
		}
		if serveErr != nil {
			d.logger.Error("rollback static output missing", "id", prev.id, "error", serveErr)
			return nil, serveErr
		}
		serveDir = resolved
		go d.serveStatic(context.Background(), serveDir, oldDeploy, extractRepoName(host))
	} else {
		go d.startApp(context.Background(), serveDir, oldDeploy, oldDeploy.AppType, extractRepoName(host), nil)
	}

	// Record rollback event
	eventID, err := kernel.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("generate rollback event id: %w", err)
	}
	event := &RollbackEvent{
		ID:             eventID,
		SiteID:         currentSiteID,
		RolledBackFrom: prev.id,
		RolledBackTo:   currentID,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	_, err = d.db.ExecContext(ctx,
		"INSERT INTO rollback_events (id, site_id, rolled_back_from, rolled_back_to, created_at) VALUES (?, ?, ?, ?, ?)",
		event.ID, event.SiteID, event.RolledBackFrom, event.RolledBackTo, event.CreatedAt)
	if err != nil {
		d.logger.Warn("failed to persist rollback event", "error", err)
	}

	return event, nil
}

func (d *Deploy) handleRollback(w http.ResponseWriter, r *http.Request, id string) {
	if !d.verifyDeploymentOwnership(w, r, id) {
		return
	}
	if r.Method != http.MethodPost {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	event, err := d.Rollback(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDeploymentNotFound) {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		} else {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "an error occurred"})
		}
		return
	}

	kernel.WriteJSON(w, http.StatusOK, event)
}

// stopDeploymentWithTransition stops a deployment and uses TransitionState
// for the status change so the state machine validates it.
func (d *Deploy) stopDeploymentWithTransition(ctx context.Context, id, newStatus string) {
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
	}

	if newStatus != "" {
		_ = d.TransitionState(ctx, id, newStatus)
	}
}

// handleRollbackEvents returns all rollback events for a site.
// GET /api/deploy/{siteID}/rollback-events
func (d *Deploy) handleRollbackEvents(w http.ResponseWriter, r *http.Request, siteID string) {
	if !d.verifySiteOwnership(w, r, siteID) {
		return
	}
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rows, err := d.db.QueryContext(r.Context(),
		`SELECT id, site_id, rolled_back_from, rolled_back_to, created_at
		 FROM rollback_events
		 WHERE site_id = ?
		 ORDER BY created_at DESC`, siteID)
	if err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "query rollback events"})
		return
	}
	defer func() { _ = rows.Close() }()

	events := make([]RollbackEvent, 0)
	for rows.Next() {
		var ev RollbackEvent
		if err := rows.Scan(&ev.ID, &ev.SiteID, &ev.RolledBackFrom, &ev.RolledBackTo, &ev.CreatedAt); err != nil {
			continue
		}
		events = append(events, ev)
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": events})
}

// extractRepoName extracts the subdomain part from a hostname.
// e.g., "myapp.example.com" → "myapp"
func extractRepoName(host string) string {
	if host == "" {
		return ""
	}
	parts := strings.SplitN(host, ".", 2)
	return parts[0]
}
