package deploy

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielvm/bigbase/kernel"
)

func (d *Deploy) ensureRelatedEventsSnapshotColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN related_events_snapshot TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return err
}

func (d *Deploy) handleDeployDiagnosis(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if d.diagnosisReader == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no diagnosis available"})
		return
	}
	diag, ok, err := d.diagnosisReader.GetDiagnosis(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no diagnosis available"})
		return
	}
	writeJSON(w, http.StatusOK, diag)
}

func (d *Deploy) handleDeployRelatedEvents(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if d.relatedEventsReader == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}
	rel, ok, err := d.relatedEventsReader.GetRelatedEvents(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (d *Deploy) emitDeployFailed(ctx context.Context, id string) {
	var siteID, repoID, appType, errMsg, buildLog string
	if err := d.db.QueryRowContext(ctx,
		`SELECT COALESCE(site_id,''), COALESCE(repo_id,''), COALESCE(app_type,''), COALESCE(error_message,''), COALESCE(build_log,'')
		 FROM deployments WHERE id = ?`, id).
		Scan(&siteID, &repoID, &appType, &errMsg, &buildLog); err != nil {
		return
	}
	if buildLog == "" {
		if lines := d.getDeployLogs(id); len(lines) > 0 {
			buildLog = strings.Join(lines, "\n")
		}
	}
	_ = d.eventBus.Emit(kernel.Event{
		Name: "deploy.failed",
		Data: map[string]any{
			"deployment_id": id,
			"site_id":       siteID,
			"repo_id":       repoID,
			"app_type":      appType,
			"error_message": errMsg,
			"build_log":     buildLog,
		},
	}, &kernel.Context{})
}
