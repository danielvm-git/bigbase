package deploy

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

func (d *Deploy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/deploy", d.handleDeploy)
	mux.HandleFunc("/api/deploy/", d.handleDeployByID)
	mux.HandleFunc("/api/samples", d.handleSamples)
	mux.HandleFunc("/api/samples/", d.handleSamples)
	mux.HandleFunc("/api/deploy/stats", d.handleDeployStats)
	d.registerCacheRoutes(mux)
	return mux
}
func (d *Deploy) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		RepoID           string   `json:"repo_id"`
		Branch           string   `json:"branch"`
		SiteName         string   `json:"site_name"`
		SiteID           string   `json:"site_id"`
		PassthroughPaths []string `json:"passthrough_paths"`
		AppType          string   `json:"app_type"`
		ManifestPath     string   `json:"manifest_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.RepoID == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id is required"})
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	if siteID, ok := kernel.SiteIDFromContext(r.Context()); ok && siteID != "" {
		// Site deploy key auth — verify the key matches the requested site.
		if req.SiteID != "" && req.SiteID != siteID {
			d.logger.Warn("deploy rejected: site key mismatch", "ctx_site_id", siteID, "req_site_id", req.SiteID)
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "site key not authorized for this site"})
			return
		}
		req.SiteID = siteID
	} else if req.SiteID != "" {
		// JWT/API-key auth — verify the site belongs to the caller's org.
		if orgID, ok := auth.OrgIDFromContext(r.Context()); ok && orgID > 0 {
			var siteOrgID int64
			err := d.db.QueryRowContext(r.Context(),
				"SELECT COALESCE(org_id, 0) FROM sites WHERE id = ?", req.SiteID).Scan(&siteOrgID)
			if err != nil {
				d.logger.Warn("deploy rejected: site not found", "site_id", req.SiteID, "error", err)
				kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "site not found or access denied"})
				return
			}
			if siteOrgID != 0 && siteOrgID != orgID {
				d.logger.Warn("deploy rejected: cross-tenant attempt",
					"caller_org", orgID, "site_org", siteOrgID, "site_id", req.SiteID)
				kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "site not found or access denied"})
				return
			}
		}
	}

	deploy, err := d.Trigger(r.Context(), req.RepoID, req.Branch, req.SiteName, req.SiteID, req.PassthroughPaths, req.AppType, req.ManifestPath)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "repo not found"})
		} else {
			d.logger.Error("create deployment", "error", err)
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		return
	}
	kernel.WriteJSON(w, http.StatusCreated, deploy)
}

// verifyDeploymentOwnership checks that the deployment's site belongs to the
// caller's org. Returns true if access is allowed (same org, no org_id in
// context, or deployment has no site). Returns false and writes 403 if denied.
func (d *Deploy) verifyDeploymentOwnership(w http.ResponseWriter, r *http.Request, deploymentID string) bool {
	orgID, ok := auth.OrgIDFromContext(r.Context())
	if !ok || orgID == 0 {
		return true // no org_id in context (site deploy key) — allow
	}

	var siteID string
	err := d.db.QueryRowContext(r.Context(),
		"SELECT COALESCE(site_id, '') FROM deployments WHERE id = ?", deploymentID).
		Scan(&siteID)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return false
	}
	if siteID == "" {
		return true // no site — can't verify ownership
	}

	var siteOrgID int64
	err = d.db.QueryRowContext(r.Context(),
		"SELECT COALESCE(org_id, 0) FROM sites WHERE id = ?", siteID).Scan(&siteOrgID)
	if err != nil {
		return true // site not found — allow (site may not have org_id)
	}
	if siteOrgID != 0 && siteOrgID != orgID {
		d.logger.Warn("access denied: cross-org deployment access",
			"caller_org", orgID, "site_org", siteOrgID, "deployment_id", deploymentID)
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return false
	}
	return true
}

// verifySiteOwnership checks that the given site belongs to the caller's org.
// Returns true if access is allowed. Returns false and writes 403 if denied.
func (d *Deploy) verifySiteOwnership(w http.ResponseWriter, r *http.Request, siteID string) bool {
	orgID, ok := auth.OrgIDFromContext(r.Context())
	if !ok || orgID == 0 {
		return true
	}

	var siteOrgID int64
	err := d.db.QueryRowContext(r.Context(),
		"SELECT COALESCE(org_id, 0) FROM sites WHERE id = ?", siteID).Scan(&siteOrgID)
	if err != nil {
		return true
	}
	if siteOrgID != 0 && siteOrgID != orgID {
		d.logger.Warn("access denied: cross-org site access",
			"caller_org", orgID, "site_org", siteOrgID, "site_id", siteID)
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return false
	}
	return true
}

func (d *Deploy) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Scope by caller's org_id if present (JWT/API-key auth).
	var rows *sql.Rows
	var err error
	if orgID, ok := auth.OrgIDFromContext(r.Context()); ok && orgID > 0 {
		rows, err = d.db.QueryContext(r.Context(),
			`SELECT d.id, d.repo_id, d.site_id, COALESCE(d.branch,'main'), COALESCE(d.commit_sha,''),
				COALESCE(d.status,'pending'), COALESCE(d.url,''), COALESCE(d.port,0), COALESCE(d.app_type,''),
				COALESCE(d.error_message,''), COALESCE(d.passthrough_paths,''), COALESCE(d.manifest_path,''),
				COALESCE(d.health_summary,''), COALESCE(d.pipeline_timeline,''), d.created_at
			 FROM deployments d
			 LEFT JOIN sites s ON d.site_id = s.id
			 WHERE s.org_id = ? OR d.site_id = '' OR d.site_id IS NULL
			 ORDER BY d.created_at DESC`, orgID)
	} else {
		rows, err = d.db.QueryContext(r.Context(),
			"SELECT id, repo_id, site_id, COALESCE(branch,'main'), COALESCE(commit_sha,''), COALESCE(status,'pending'), COALESCE(url,''), COALESCE(port,0), COALESCE(app_type,''), COALESCE(error_message,''), COALESCE(passthrough_paths,''), COALESCE(manifest_path,''), COALESCE(health_summary,''), COALESCE(pipeline_timeline,''), created_at FROM deployments ORDER BY created_at DESC")
	}
	if err != nil {
		d.logger.Error("list deployments", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	deployments := make([]Deployment, 0)
	for rows.Next() {
		var dep Deployment
		var passthroughJSON, timelineJSON string
		if err := rows.Scan(&dep.ID, &dep.RepoID, &dep.SiteID, &dep.Branch, &dep.CommitSHA, &dep.Status, &dep.URL, &dep.Port, &dep.AppType, &dep.ErrorMessage, &passthroughJSON, &dep.ManifestPath, &dep.HealthSummary, &timelineJSON, &dep.CreatedAt); err != nil {
			d.logger.Error("scan deployment", "error", err)
			continue
		}
		dep.PassthroughPaths = parsePassthroughPaths(passthroughJSON)
		dep.PipelineTimeline = parsePipelineTimeline(timelineJSON)
		deployments = append(deployments, dep)
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": deployments})
}
func (d *Deploy) handleDeploy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		d.HandleCreate(w, r)
	case "GET":
		d.HandleList(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
func (d *Deploy) handleDeployByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/deploy/")
	if path == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	// Check for sub-paths: /api/deploy/:id/logs/stream, /api/deploy/:id/logs, /api/deploy/:id/rollback
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
	if strings.HasSuffix(path, "/rollback-events") {
		id := strings.TrimSuffix(path, "/rollback-events")
		d.handleRollbackEvents(w, r, id)
		return
	}
	if strings.HasSuffix(path, "/rollback") {
		id := strings.TrimSuffix(path, "/rollback")
		d.handleRollback(w, r, id)
		return
	}
	if strings.HasSuffix(path, "/diagnosis") {
		id := strings.TrimSuffix(path, "/diagnosis")
		d.handleDeployDiagnosis(w, r, id)
		return
	}
	if strings.HasSuffix(path, "/related-events") {
		id := strings.TrimSuffix(path, "/related-events")
		d.handleDeployRelatedEvents(w, r, id)
		return
	}

	id := path
	switch r.Method {
	case http.MethodDelete:
		d.handleDeleteDeployment(w, r, id)
	case http.MethodGet:
		if !d.verifyDeploymentOwnership(w, r, id) {
			return
		}
		var dep Deployment
		var appType, passthroughJSON, timelineJSON string
		err := d.db.QueryRowContext(r.Context(),
			"SELECT id, repo_id, site_id, branch, commit_sha, status, url, port, app_type, COALESCE(error_message,''), COALESCE(passthrough_paths,''), COALESCE(manifest_path,''), COALESCE(health_summary,''), COALESCE(pipeline_timeline,''), created_at FROM deployments WHERE id = ?", id).
			Scan(&dep.ID, &dep.RepoID, &dep.SiteID, &dep.Branch, &dep.CommitSHA, &dep.Status, &dep.URL, &dep.Port, &appType, &dep.ErrorMessage, &passthroughJSON, &dep.ManifestPath, &dep.HealthSummary, &timelineJSON, &dep.CreatedAt)
		if err != nil {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
			return
		}
		dep.AppType = AppType(appType)
		dep.PassthroughPaths = parsePassthroughPaths(passthroughJSON)
		dep.PipelineTimeline = parsePipelineTimeline(timelineJSON)
		kernel.WriteJSON(w, http.StatusOK, dep)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
func (d *Deploy) handleDeleteDeployment(w http.ResponseWriter, r *http.Request, id string) {
	if !d.verifyDeploymentOwnership(w, r, id) {
		return
	}
	cleanID := filepath.Clean(id)
	if strings.Contains(cleanID, "..") || strings.Contains(cleanID, "/") {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var status, url string
	err := d.db.QueryRowContext(r.Context(),
		"SELECT status, COALESCE(url,'') FROM deployments WHERE id = ?", id).
		Scan(&status, &url)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}
	if status == "pending" || status == "building" {
		kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "deployment is in progress — wait or trigger a new deploy"})
		return
	}

	d.stopDeployment(id, "replaced")

	_ = os.RemoveAll(filepath.Join(d.buildsDir, cleanID))
	d.deleteDeployLogs(id)
	_, _ = d.db.ExecContext(r.Context(), "DELETE FROM deployments WHERE id = ?", id)

	w.WriteHeader(http.StatusNoContent)
}
func (d *Deploy) handleDeployLogs(w http.ResponseWriter, r *http.Request, id string) {
	if !d.verifyDeploymentOwnership(w, r, id) {
		return
	}
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Verify deployment exists and load status + error message + build_log
	var status, errMsg, buildLog string
	err := d.db.QueryRowContext(r.Context(),
		"SELECT status, COALESCE(error_message,''), COALESCE(build_log,'') FROM deployments WHERE id = ?", id).
		Scan(&status, &errMsg, &buildLog)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
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
	kernel.WriteJSON(w, http.StatusOK, payload)
}
func (d *Deploy) handleDeployStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx := r.Context()
	stats := map[string]any{}

	// Scope stats by caller's org_id to prevent cross-tenant info disclosure.
	// Deployments are linked to sites via site_id; sites have org_id.
	orgID, hasOrg := auth.OrgIDFromContext(ctx)
	var orgFilter string
	var orgArgs []any
	if hasOrg && orgID > 0 {
		orgFilter = " AND d.site_id IN (SELECT id FROM sites WHERE org_id = ?)"
		orgArgs = append(orgArgs, orgID)
	}

	var total int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments d WHERE 1=1"+orgFilter, orgArgs...).Scan(&total); err == nil {
		stats["total"] = total
	}

	var running int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments d WHERE status = 'running'"+orgFilter, orgArgs...).Scan(&running); err == nil {
		stats["running"] = running
	}

	var totalFailed int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments d WHERE status = 'failed'"+orgFilter, orgArgs...).Scan(&totalFailed); err == nil {
		stats["total_failed"] = totalFailed
	}

	var recentFailed int
	if err := d.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployments d WHERE status = 'failed' AND created_at > datetime('now', '-1 day')"+orgFilter,
		orgArgs...).Scan(&recentFailed); err == nil {
		stats["failed_24h"] = recentFailed
	}

	var recentTotal int
	if err := d.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM deployments d WHERE created_at > datetime('now', '-1 day')"+orgFilter,
		orgArgs...).Scan(&recentTotal); err == nil {
		stats["total_24h"] = recentTotal
		if recentTotal > 0 {
			stats["failure_rate_24h"] = fmt.Sprintf("%.1f%%", float64(recentFailed)/float64(recentTotal)*100)
		}
	}

	kernel.WriteJSON(w, http.StatusOK, stats)
}
