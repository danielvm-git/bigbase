package deploy

import (
	"context"
	"net/http"
	"strings"
)

// SampleApp describes a pre-built sample application users can deploy in one click.
type SampleApp struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repo_url"`
	Description string `json:"description"`
}

// catalogSamples is the static list of sample apps available for one-click deploy.
var catalogSamples = []SampleApp{
	{
		Name:        "todo-app",
		RepoURL:     "https://github.com/bigbase-samples/todo-app",
		Description: "A simple to-do list built with vanilla JS and BigBase Collections API.",
	},
	{
		Name:        "blog",
		RepoURL:     "https://github.com/bigbase-samples/blog",
		Description: "A Markdown-powered blog using BigBase Storage and Functions.",
	},
	{
		Name:        "chat",
		RepoURL:     "https://github.com/bigbase-samples/chat",
		Description: "Real-time chat powered by BigBase Realtime and Messaging.",
	},
}

// sampleByName returns the sample with the given name, or nil.
func sampleByName(name string) *SampleApp {
	for i := range catalogSamples {
		if catalogSamples[i].Name == name {
			return &catalogSamples[i]
		}
	}
	return nil
}

// handleSamples routes /api/samples and /api/samples/:name/deploy.
func (d *Deploy) handleSamples(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/samples")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		if r.Method != "GET" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": catalogSamples})
		return
	}

	// Expect :name/deploy
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[1] != "deploy" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	name := parts[0]
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	sample := sampleByName(name)
	if sample == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sample not found"})
		return
	}

	repoID, err := d.ensureSampleRepo(r.Context(), sample)
	if err != nil {
		d.logger.Error("ensure sample repo", "name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	deployment, err := d.Trigger(r.Context(), repoID, "main", "", "sample-site", nil, "", "")
	if err != nil {
		d.logger.Error("trigger sample deploy", "name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, deployment)
}

// ensureSampleRepo returns the git_repos id for the sample, creating a stub row if needed.
func (d *Deploy) ensureSampleRepo(ctx context.Context, sample *SampleApp) (string, error) {
	var id string
	err := d.db.QueryRowContext(ctx, "SELECT id FROM git_repos WHERE name = ?", sample.Name).Scan(&id)
	if err == nil {
		return id, nil
	}

	newID, genErr := generateID()
	if genErr != nil {
		return "", genErr
	}

	_, execErr := d.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO git_repos (id, name, owner_id, private, default_branch, description) VALUES (?, ?, 0, 0, 'main', ?)",
		newID, sample.Name, sample.Description)
	if execErr != nil {
		// Re-query: another goroutine may have inserted concurrently.
		_ = d.db.QueryRowContext(ctx, "SELECT id FROM git_repos WHERE name = ?", sample.Name).Scan(&newID)
	}

	return newID, nil
}
