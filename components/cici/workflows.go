package cici

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (c *CICI) handleRepoWorkflows(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/cici/")
	parts := strings.SplitN(path, "/", 4)
	if len(parts) < 2 || parts[0] == "" || parts[1] != "workflows" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	repoID := parts[0]

	if r.Method == "GET" {
		c.listWorkflows(w, r, repoID)
		return
	}
	if r.Method == "PUT" {
		c.saveWorkflow(w, r, repoID)
		return
	}
	if len(parts) >= 4 && parts[3] == "run" && r.Method == "POST" {
		c.triggerRun(w, r, repoID, parts[2])
		return
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

type saveWorkflowReq struct {
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

func (c *CICI) saveWorkflow(w http.ResponseWriter, r *http.Request, repoID string) {
	req, ok := decodeSaveWorkflow(w, r)
	if !ok {
		return
	}

	var wf workflowYAML
	if err := yaml.Unmarshal([]byte(req.YAML), &wf); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid yaml: " + err.Error()})
		return
	}

	id, err := generateID()
	if err != nil {
		c.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := c.db.ExecContext(r.Context(),
		"INSERT INTO cici_workflows (id, repo_id, name, yaml, created_at) VALUES (?, ?, ?, ?, ?)",
		id, repoID, req.Name, req.YAML, now); err != nil {
		c.logger.Error("insert workflow", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, WorkflowDef{
		ID: id, RepoID: repoID, Name: req.Name, YAML: req.YAML,
	})
}

func decodeSaveWorkflow(w http.ResponseWriter, r *http.Request) (*saveWorkflowReq, bool) {
	var req saveWorkflowReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return nil, false
	}
	if req.Name == "" || req.YAML == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and yaml are required"})
		return nil, false
	}
	return &req, true
}

func (c *CICI) listWorkflows(w http.ResponseWriter, r *http.Request, repoID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		"SELECT id, repo_id, name, yaml FROM cici_workflows WHERE repo_id = ? ORDER BY name", repoID)
	if err != nil {
		c.logger.Error("list workflows", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	workflows := make([]WorkflowDef, 0)
	for rows.Next() {
		var w WorkflowDef
		if err := rows.Scan(&w.ID, &w.RepoID, &w.Name, &w.YAML); err != nil {
			c.logger.Error("scan workflow", "error", err)
			continue
		}
		workflows = append(workflows, w)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": workflows})
}

func (c *CICI) fetchWorkflow(ctx context.Context, id string) (*WorkflowDef, error) {
	var w WorkflowDef
	err := c.db.QueryRowContext(ctx,
		"SELECT id, repo_id, name, yaml FROM cici_workflows WHERE id = ?", id).
		Scan(&w.ID, &w.RepoID, &w.Name, &w.YAML)
	if err != nil {
		return nil, fmt.Errorf("fetch workflow: %w", err)
	}
	return &w, nil
}
