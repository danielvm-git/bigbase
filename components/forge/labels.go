package forge

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (f *Forge) createLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID string `json:"repo_id"`
		Name   string `json:"name"`
		Color  string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RepoID == "" || req.Name == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id and name are required"})
		return
	}
	if req.Color == "" {
		req.Color = "#cccccc"
	}

	id, err := kernel.GenerateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if _, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_labels (id, repo_id, name, color) VALUES (?, ?, ?, ?)",
		id, req.RepoID, req.Name, req.Color); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "label already exists"})
			return
		}
		f.logger.Error("insert label", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusCreated, Label{ID: id, RepoID: req.RepoID, Name: req.Name, Color: req.Color})
}

func (f *Forge) listLabels(w http.ResponseWriter, r *http.Request) {
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := f.db.QueryContext(ctx,
		"SELECT id, repo_id, name, color FROM forge_labels WHERE repo_id = ? ORDER BY name", repoID)
	if err != nil {
		f.logger.Error("list labels", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	labels := make([]Label, 0)
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.RepoID, &l.Name, &l.Color); err != nil {
			f.logger.Error("scan label", "error", err)
			continue
		}
		labels = append(labels, l)
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": labels})
}
