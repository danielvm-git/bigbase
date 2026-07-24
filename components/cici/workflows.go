package cici

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/danielvm/bigbase/kernel"
)

type saveWorkflowReq struct {
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

func (c *CICI) saveWorkflow(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo")

	// Verify repo ownership before allowing workflow creation
	if err := c.verifyRepoOwnership(r.Context(), repoID); err != nil {
		if err == ErrRepoNotFound {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "repo not found"})
			return
		}
		c.logger.Warn("saveWorkflow rejected: cross-tenant attempt",
			"repo_id", repoID, "org_id", orgIDFromReq(r))
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return
	}

	var req saveWorkflowReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.YAML == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name and yaml are required"})
		return
	}

	var wf workflowYAML
	if err := yaml.Unmarshal([]byte(req.YAML), &wf); err != nil {
		c.logger.Warn("invalid workflow yaml", "repo", repoID, "error", err)
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workflow YAML format"})
		return
	}

	id, err := kernel.GenerateID()
	if err != nil {
		c.logger.Error("generate id", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := c.db.ExecContext(r.Context(),
		"INSERT INTO cici_workflows (id, repo_id, name, yaml, created_at) VALUES (?, ?, ?, ?, ?)",
		id, repoID, req.Name, req.YAML, now); err != nil {
		c.logger.Error("insert workflow", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, WorkflowDef{
		ID: id, RepoID: repoID, Name: req.Name, YAML: req.YAML,
	})
}

func (c *CICI) listWorkflows(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo")

	// Verify repo ownership before listing workflows
	if err := c.verifyRepoOwnership(r.Context(), repoID); err != nil {
		if err == ErrRepoNotFound {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "repo not found"})
			return
		}
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		"SELECT id, repo_id, name, yaml FROM cici_workflows WHERE repo_id = ? ORDER BY name", repoID)
	if err != nil {
		c.logger.Error("list workflows", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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
	if err := rows.Err(); err != nil {
		c.logger.Error("iterate workflows", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": workflows})
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
