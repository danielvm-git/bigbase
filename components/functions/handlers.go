package functions

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (f *Functions) handleFunctions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		f.handleCreate(w, r)
	case "GET":
		f.handleList(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Functions) handleFunctionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/functions/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if len(parts) == 2 && parts[1] == "run" {
		if r.Method == "POST" {
			f.handleRun(w, r, id)
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	switch r.Method {
	case "GET":
		f.handleGet(w, r, id)
	case "PUT":
		f.handleUpdate(w, r, id)
	case "DELETE":
		f.handleDelete(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Functions) handleCreate(w http.ResponseWriter, r *http.Request) {
	fn, err := f.decodeFunction(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	fn.ID, err = generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	envJSON, _ := json.Marshal(fn.Env)
	_, err = f.db.ExecContext(r.Context(),
		"INSERT INTO functions (id, name, runtime, source, trigger, schedule, env, timeout, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))",
		fn.ID, fn.Name, fn.Runtime, fn.Source, fn.Trigger, fn.Schedule, string(envJSON), fn.Timeout)
	if err != nil {
		f.logger.Error("insert function", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	fn.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusCreated, fn)
}

func (f *Functions) decodeFunction(r *http.Request) (Function, error) {
	var fn Function
	if err := json.NewDecoder(r.Body).Decode(&fn); err != nil {
		return fn, err
	}
	if fn.Name == "" {
		return fn, errMissingName
	}
	if fn.Source == "" {
		return fn, errMissingSource
	}
	if fn.Runtime == "" {
		fn.Runtime = "javascript"
	}
	if fn.Trigger == "" {
		fn.Trigger = "http"
	}
	if fn.Timeout == 0 {
		fn.Timeout = f.timeout
	}
	return fn, nil
}

func (f *Functions) handleList(w http.ResponseWriter, r *http.Request) {
	rows, err := f.db.QueryContext(r.Context(),
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions ORDER BY created_at DESC")
	if err != nil {
		f.logger.Error("list functions", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	funcs := make([]Function, 0)
	for rows.Next() {
		fn, scanErr := f.scanFunction(rows)
		if scanErr != nil {
			f.logger.Error("scan function", "error", scanErr)
			continue
		}
		funcs = append(funcs, fn)
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": funcs})
}

func (f *Functions) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	fn, err := f.fetchFunctionByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}
	writeJSON(w, http.StatusOK, fn)
}

func (f *Functions) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var fn Function
	if err := json.NewDecoder(r.Body).Decode(&fn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	envJSON, _ := json.Marshal(fn.Env)
	result, err := f.db.ExecContext(r.Context(),
		"UPDATE functions SET name=?, runtime=?, source=?, trigger=?, schedule=?, env=?, timeout=? WHERE id=?",
		fn.Name, fn.Runtime, fn.Source, fn.Trigger, fn.Schedule, string(envJSON), fn.Timeout, id)
	if err != nil {
		f.logger.Error("update function", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	fn.ID = id
	writeJSON(w, http.StatusOK, fn)
}

func (f *Functions) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	result, err := f.db.ExecContext(r.Context(), "DELETE FROM functions WHERE id = ?", id)
	if err != nil {
		f.logger.Error("delete function", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (f *Functions) handleRun(w http.ResponseWriter, r *http.Request, id string) {
	fn, err := f.fetchFunctionByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	rt, ok := runtimes[fn.Runtime]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported runtime: " + fn.Runtime})
		return
	}

	output, execErr := rt.Execute(fn.Source, fn.Timeout)
	if execErr != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"logs":   output.Logs,
			"error":  execErr.Error(),
			"result": nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"logs":   output.Logs,
		"error":  nil,
		"result": output.Result,
	})
}
