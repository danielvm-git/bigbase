package forge

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func (f *Forge) createIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID      string   `json:"repo_id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Labels      []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RepoID == "" || req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id and title are required"})
		return
	}
	if req.Status == "" {
		req.Status = "open"
	}

	id, err := generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_issues (id, repo_id, title, description, status, labels, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.RepoID, req.Title, req.Description, req.Status, string(labelsJSON), now, now); err != nil {
		f.logger.Error("insert issue", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, Issue{
		ID: id, RepoID: req.RepoID, Title: req.Title,
		Description: req.Description, Status: req.Status,
		Labels: req.Labels, CreatedAt: now, UpdatedAt: now,
	})
}

func (f *Forge) listIssues(w http.ResponseWriter, r *http.Request) {
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id query parameter required"})
		return
	}

	issues, err := f.fetchIssues(r.Context(), repoID, "")
	if err != nil {
		f.logger.Error("list issues", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": issues})
}

func (f *Forge) fetchIssues(ctx context.Context, repoID, status string) ([]Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = f.db.QueryContext(ctx,
			"SELECT id, repo_id, title, description, status, labels, created_at, updated_at FROM forge_issues WHERE repo_id = ? AND status = ? ORDER BY updated_at DESC",
			repoID, status)
	} else {
		rows, err = f.db.QueryContext(ctx,
			"SELECT id, repo_id, title, description, status, labels, created_at, updated_at FROM forge_issues WHERE repo_id = ? ORDER BY created_at DESC",
			repoID)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	issues := make([]Issue, 0)
	for rows.Next() {
		var i Issue
		var labelsStr string
		if err := rows.Scan(&i.ID, &i.RepoID, &i.Title, &i.Description, &i.Status, &labelsStr, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &i.Labels)
		issues = append(issues, i)
	}
	return issues, rows.Err()
}

func (f *Forge) getIssue(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var i Issue
	var labelsStr string
	err := f.db.QueryRowContext(ctx,
		"SELECT id, repo_id, title, description, status, labels, created_at, updated_at FROM forge_issues WHERE id = ?", id).
		Scan(&i.ID, &i.RepoID, &i.Title, &i.Description, &i.Status, &labelsStr, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}
	_ = json.Unmarshal([]byte(labelsStr), &i.Labels)
	writeJSON(w, http.StatusOK, i)
}

func (f *Forge) updateIssue(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Labels      []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	validStatuses := map[string]bool{"open": true, "in_progress": true, "review": true, "closed": true}
	if req.Status != "" && !validStatuses[req.Status] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := f.db.ExecContext(r.Context(),
		"UPDATE forge_issues SET title=COALESCE(NULLIF(?,''),title), description=COALESCE(NULLIF(?,''),description), status=COALESCE(NULLIF(?,''),status), labels=COALESCE(NULLIF(?,''),labels), updated_at=? WHERE id=?",
		req.Title, req.Description, req.Status, string(labelsJSON), now, id)
	if err != nil {
		f.logger.Error("update issue", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (f *Forge) handleComments(w http.ResponseWriter, r *http.Request, issueID string) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	id, err := generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_comments (id, issue_id, content, created_at) VALUES (?, ?, ?, ?)",
		id, issueID, req.Content, now); err != nil {
		f.logger.Error("insert comment", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, Comment{
		ID: id, IssueID: issueID, Content: req.Content, CreatedAt: now,
	})
}
