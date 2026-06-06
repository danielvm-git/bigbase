package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// kernelEvent constructs a kernel.Event for emission on the event bus.
func kernelEvent(name string, data map[string]any) kernel.Event {
	return kernel.Event{Name: name, Data: data}
}

// generateScaffoldID produces a random hex ID for scaffold resources.
func generateScaffoldID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// handleScaffoldDB handles POST /api/scaffold/db.
// Creates a named collection (table) with a default schema.
func (a *API) handleScaffoldDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if _, err := sanitize(name); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if internalTables[name] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reserved name"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.ensureTable(name); err != nil {
		a.logger.Error("scaffold db create table", "name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.emitMutation("scaffold_db", name, nil)

	_ = ctx
	writeJSON(w, http.StatusCreated, map[string]string{"name": name, "status": "created"})
}

// handleScaffoldRepo handles POST /api/scaffold/repo.
// Emits an event requesting the git component to initialise a new repo.
func (a *API) handleScaffoldRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if a.bus != nil {
		_ = a.bus.Emit(kernelEvent("scaffold_repo", map[string]any{"name": name}), nil)
	}

	writeJSON(w, http.StatusCreated, map[string]string{"name": name, "status": "initialised"})
}

// handleScaffoldFunction handles POST /api/scaffold/function.
// Creates a stub function record in the functions table.
func (a *API) handleScaffoldFunction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Ensure the functions table exists (created by the functions component on start;
	// we do a best-effort CREATE IF NOT EXISTS here for tests that run without it).
	_ = a.db.Migrate(`CREATE TABLE IF NOT EXISTS functions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		runtime TEXT NOT NULL DEFAULT 'node18',
		source TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)

	id := generateScaffoldID()
	if _, err := a.db.ExecContext(ctx,
		"INSERT INTO functions (id, name, runtime, source) VALUES (?, ?, 'node18', '')",
		id, name); err != nil {
		a.logger.Error("scaffold function insert", "name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.emitMutation("scaffold_function", "functions", id)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name, "status": "created"})
}
