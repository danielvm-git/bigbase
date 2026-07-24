package cici

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

func (c *CICI) triggerRun(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("id")

	var req struct {
		Event string `json:"event"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Event == "" {
		req.Event = "manual"
	}

	wf, err := c.fetchWorkflow(r.Context(), workflowID)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}

	// Verify the workflow's repo belongs to the caller's org
	if err := c.verifyRepoOwnership(r.Context(), wf.RepoID); err != nil {
		c.logger.Warn("triggerRun rejected: cross-tenant attempt",
			"workflow_id", workflowID, "repo_id", wf.RepoID, "org_id", orgIDFromReq(r))
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return
	}

	runID, err := kernel.GenerateID()
	if err != nil {
		c.logger.Error("generate run id", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := c.db.ExecContext(r.Context(),
		"INSERT INTO cici_runs (id, workflow_id, event, status, started_at) VALUES (?, ?, ?, 'running', ?)",
		runID, workflowID, req.Event, now); err != nil {
		c.logger.Error("insert run", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	go c.executeRun(runID, wf)

	kernel.WriteJSON(w, http.StatusAccepted, PipelineRun{
		ID: runID, WorkflowID: workflowID, Event: req.Event,
		Status: "running", StartedAt: now,
	})
}

func (c *CICI) executeRun(runID string, wf *WorkflowDef) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var wfYAML workflowYAML
	if err := yaml.Unmarshal([]byte(wf.YAML), &wfYAML); err != nil {
		c.finishRun(ctx, runID, "failure")
		return
	}

	status := "success"
	for jobName, jobDef := range wfYAML.Jobs {
		for i, stepDef := range jobDef.Steps {
			if stepDef.Run == "" {
				continue
			}
			stepName := fmt.Sprintf("%s.step-%d", jobName, i)
			if err := c.executeStep(ctx, runID, stepName, stepDef.Run); err != nil {
				c.logger.Error("step failed", "run_id", runID, "step", stepName, "error", err)
				c.appendLog(ctx, runID, stepName, "FAIL: "+err.Error())
				status = "failure"
				break
			}
		}
		if status == "failure" {
			break
		}
	}

	c.finishRun(ctx, runID, status)
}

func (c *CICI) executeStep(ctx context.Context, runID, name, command string) error {
	// Defense-in-depth: validate command against dangerous patterns
	if err := validateCommand(command); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += strings.TrimSpace(stderr.String())
	}

	c.appendLog(ctx, runID, name, output)
	return err
}

func (c *CICI) appendLog(ctx context.Context, runID, step, output string) {
	_, err := c.db.ExecContext(ctx,
		"INSERT INTO cici_logs (run_id, step, output) VALUES (?, ?, ?)",
		runID, step, output)
	if err != nil {
		c.logger.Error("append log", "run_id", runID, "error", err)
	}
}

func (c *CICI) finishRun(ctx context.Context, runID, status string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.ExecContext(ctx,
		"UPDATE cici_runs SET status = ?, finished_at = ? WHERE id = ?",
		status, now, runID)
	if err != nil {
		c.logger.Error("finish run", "run_id", runID, "error", err)
	}
}

func (c *CICI) handleRuns(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	query := "SELECT id, workflow_id, event, status, started_at, finished_at FROM cici_runs"
	args := make([]any, 0)

	// Scope by org_id to prevent cross-tenant data leakage
	if orgID, ok := auth.OrgIDFromContext(r.Context()); ok && orgID > 0 {
		query += " WHERE workflow_id IN (SELECT w.id FROM cici_workflows w JOIN git_repos g ON g.id = w.repo_id WHERE g.owner_id = ?)"
		args = append(args, orgID)
	}

	repoID := r.URL.Query().Get("repo_id")
	if repoID != "" {
		if len(args) > 0 {
			query += " AND workflow_id IN (SELECT id FROM cici_workflows WHERE repo_id = ?)"
		} else {
			query += " WHERE workflow_id IN (SELECT id FROM cici_workflows WHERE repo_id = ?)"
		}
		args = append(args, repoID)
	}
	query += " ORDER BY started_at DESC"

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	query += " LIMIT ?"
	args = append(args, limit)

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	query += " OFFSET ?"
	args = append(args, offset)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.logger.Error("list runs", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	runs := make([]PipelineRun, 0)
	for rows.Next() {
		var r PipelineRun
		if err := rows.Scan(&r.ID, &r.WorkflowID, &r.Event, &r.Status, &r.StartedAt, &r.FinishedAt); err != nil {
			c.logger.Error("scan run", "error", err)
			continue
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		c.logger.Error("iterate runs", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": runs})
}

func (c *CICI) getRunLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	// Verify the run's workflow belongs to the caller's org
	if orgID, ok := auth.OrgIDFromContext(r.Context()); ok && orgID > 0 {
		var runOwnerOrg int64
		err := c.db.QueryRowContext(r.Context(),
			`SELECT COALESCE(g.owner_id, 0)
			 FROM cici_runs r
			 JOIN cici_workflows w ON w.id = r.workflow_id
			 JOIN git_repos g ON g.id = w.repo_id
			 WHERE r.id = ?`, runID).Scan(&runOwnerOrg)
		if err != nil {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
			return
		}
		if runOwnerOrg != 0 && runOwnerOrg != orgID {
			c.logger.Warn("getRunLogs rejected: cross-tenant attempt",
				"run_id", runID, "caller_org", orgID, "run_org", runOwnerOrg)
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		"SELECT id, run_id, step, output FROM cici_logs WHERE run_id = ? ORDER BY id", runID)
	if err != nil {
		c.logger.Error("get logs", "run_id", runID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	logs := make([]LogEntry, 0)
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.RunID, &l.Step, &l.Output); err != nil {
			c.logger.Error("scan log", "error", err)
			continue
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		c.logger.Error("iterate logs", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"logs": logs})
}
