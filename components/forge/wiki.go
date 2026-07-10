package forge

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (f *Forge) getWiki(w http.ResponseWriter, r *http.Request, title string) {
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var page WikiPage
	err := f.db.QueryRowContext(ctx,
		"SELECT title, repo_id, content, updated_at FROM forge_wiki WHERE title = ? AND repo_id = ?", title, repoID).
		Scan(&page.Title, &page.RepoID, &page.Content, &page.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (f *Forge) saveWiki(w http.ResponseWriter, r *http.Request, title string) {
	var req struct {
		RepoID  string `json:"repo_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RepoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id is required"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_wiki (title, repo_id, content, updated_at) VALUES (?, ?, ?, ?) "+
			"ON CONFLICT(title, repo_id) DO UPDATE SET content=excluded.content, updated_at=excluded.updated_at",
		title, req.RepoID, req.Content, now)
	if err != nil {
		f.logger.Error("save wiki", "title", title, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, WikiPage{Title: title, RepoID: req.RepoID, Content: req.Content, UpdatedAt: now})
}
