package deploy

import (
	"context"
	"encoding/json"
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

	// Fetch previous successful deployment for the same site
	// Exclude current, failed, pending, and building deployments.
	type prevRow struct {
		id, status, appType, url, passthroughJSON, commitSHA string
		port                                                 int
	}
	var prev prevRow
	err = d.db.QueryRowContext(ctx,
		`SELECT id, status, port, app_type, COALESCE(url,''), COALESCE(passthrough_paths,''), COALESCE(commit_sha,'')
		 FROM deployments
		 WHERE site_id = ? AND id != ? AND status NOT IN ('failed', 'pending', 'building')
		 ORDER BY created_at DESC LIMIT 1`,
		currentSiteID, currentID).Scan(&prev.id, &prev.status, &prev.port, &prev.appType, &prev.url, &prev.passthroughJSON, &prev.commitSHA)
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

	// Update the old deployment's status to running (it was in "replaced" state)
	// Use updateStatus which calls TransitionState — this works because
	// TransitionState has lenient mode for unknown statuses like "replaced".
	d.updateStatus(prev.id, string(StateRunning))
	d.updateStatusHistory(ctx, prev.id, prev.status, string(StateRunning))
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
		if _, err := os.Stat(filepath.Join(buildDir, "dist")); err == nil {
			serveDir = filepath.Join(buildDir, "dist")
		} else if _, err := os.Stat(filepath.Join(buildDir, "build")); err == nil {
			serveDir = filepath.Join(buildDir, "build")
		}
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
	}

	if newStatus != "" {
		_ = d.TransitionState(ctx, id, newStatus)
	}
}

// updateStatusHistory appends a transition record to the deployment's
// status_history JSON without going through TransitionState validation.
// Used for rollback where the status change crosses states not in the
// state machine (e.g., "replaced" → "running").
func (d *Deploy) updateStatusHistory(ctx context.Context, id, from, to string) {
	var current string
	if err := d.db.QueryRowContext(ctx,
		"SELECT status_history FROM deployments WHERE id = ?", id).Scan(&current); err != nil {
		return
	}
	if current == "" {
		current = "[]"
	}
	var history []StatusTransition
	if err := json.Unmarshal([]byte(current), &history); err != nil {
		history = nil
	}
	history = append(history, StatusTransition{
		From:      from,
		To:        to,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	data, err := json.Marshal(history)
	if err != nil {
		return
	}
	_, _ = d.db.ExecContext(ctx,
		"UPDATE deployments SET status_history = ? WHERE id = ?", string(data), id)
}

// handleRollbackEvents returns all rollback events for a site.
// GET /api/deploy/{siteID}/rollback-events
func (d *Deploy) handleRollbackEvents(w http.ResponseWriter, r *http.Request, siteID string) {
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
