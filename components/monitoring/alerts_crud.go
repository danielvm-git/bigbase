package monitoring

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielvm/bigbase/kernel"
)

// handleAlertByID handles GET/DELETE for a single alert by ID.
func (m *Monitoring) handleAlertByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/monitoring/alerts/")
	if id == "" || strings.Contains(id, "/") {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		m.handleAlertGet(w, r, id)
	case http.MethodDelete:
		m.handleAlertDelete(w, r, id)
	case http.MethodPut:
		m.handleAlertUpdate(w, r, id)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (m *Monitoring) handleAlertGet(w http.ResponseWriter, r *http.Request, id string) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var a Alert
	var enabled int
	err := m.db.QueryRowContext(r.Context(),
		"SELECT id, name, metric, threshold, operator, enabled, duration_seconds FROM monitoring_alerts WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.Metric, &a.Threshold, &a.Operator, &enabled, &a.DurationSeconds)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
		return
	}
	a.Enabled = enabled == 1
	kernel.WriteJSON(w, http.StatusOK, a)
}

func (m *Monitoring) handleAlertDelete(w http.ResponseWriter, r *http.Request, id string) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db not configured"})
		return
	}
	_, err := m.db.ExecContext(r.Context(),
		"DELETE FROM monitoring_alerts WHERE id = ?", id)
	if err != nil {
		m.logger.Error("delete alert", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]string{"id": id, "deleted": "true"})
}

func (m *Monitoring) handleAlertUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db not configured"})
		return
	}
	var body struct {
		Name      string  `json:"name"`
		Metric    string  `json:"metric"`
		Threshold float64 `json:"threshold"`
		Operator  string  `json:"operator"`
		Enabled   bool    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Name == "" || body.Metric == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name and metric are required"})
		return
	}
	enabled := 0
	if body.Enabled {
		enabled = 1
	}
	_, err := m.db.ExecContext(r.Context(),
		"UPDATE monitoring_alerts SET name=?, metric=?, threshold=?, operator=?, enabled=? WHERE id=?",
		body.Name, body.Metric, body.Threshold, body.Operator, enabled, id)
	if err != nil {
		m.logger.Error("update alert", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]string{"id": id})
}
