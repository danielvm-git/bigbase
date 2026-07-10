package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (f *Functions) handleFunctions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		f.handleCreate(w, r)
	case "GET":
		f.handleList(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Functions) handleFunctionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/functions/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if len(parts) == 2 && parts[1] == "run" {
		if r.Method == "POST" {
			f.handleRun(w, r, id)
			return
		}
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		f.handleFunctionLogs(w, r, id)
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
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Functions) handleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	fn, err := f.decodeFunction(r)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	fn.ID, err = kernel.GenerateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	envJSON, _ := json.Marshal(fn.Env)
	_, err = f.db.ExecContext(r.Context(),
		"INSERT INTO functions (id, name, runtime, source, trigger, schedule, env, timeout, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))",
		fn.ID, fn.Name, fn.Runtime, fn.Source, fn.Trigger, fn.Schedule, string(envJSON), fn.Timeout)
	if err != nil {
		f.logger.Error("insert function", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	fn.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Reload schedule if the function is schedule-triggered
	if fn.Trigger == "schedule" && fn.Schedule != "" && f.scheduleEnabled {
		if err := f.reloadSchedule(); err != nil {
			f.logger.Error("reload schedule after create", "error", err)
		}
	}

	kernel.WriteJSON(w, http.StatusCreated, fn)
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
	if fn.Env == nil {
		fn.Env = map[string]string{}
	}
	return fn, nil
}

func (f *Functions) handleList(w http.ResponseWriter, r *http.Request) {
	rows, err := f.db.QueryContext(r.Context(),
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions ORDER BY created_at DESC")
	if err != nil {
		f.logger.Error("list functions", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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

	if err := rows.Err(); err != nil {
		f.logger.Error("iterate functions", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": funcs})
}

func (f *Functions) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	fn, err := f.fetchFunctionByID(r.Context(), id)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, fn)
}

func (f *Functions) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	fn, err := f.decodeFunction(r)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	envJSON, _ := json.Marshal(fn.Env)
	result, err := f.db.ExecContext(r.Context(),
		"UPDATE functions SET name=?, runtime=?, source=?, trigger=?, schedule=?, env=?, timeout=? WHERE id=?",
		fn.Name, fn.Runtime, fn.Source, fn.Trigger, fn.Schedule, string(envJSON), fn.Timeout, id)
	if err != nil {
		f.logger.Error("update function", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	fn, err = f.fetchFunctionByID(r.Context(), id)
	if err != nil {
		f.logger.Error("fetch after update", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Reload schedule in case trigger/schedule changed
	if f.scheduleEnabled {
		if err := f.reloadSchedule(); err != nil {
			f.logger.Error("reload schedule after update", "error", err)
		}
	}

	kernel.WriteJSON(w, http.StatusOK, fn)
}

func (f *Functions) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	result, err := f.db.ExecContext(r.Context(), "DELETE FROM functions WHERE id = ?", id)
	if err != nil {
		f.logger.Error("delete function", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	// Reload schedule in case a scheduled function was deleted
	if f.scheduleEnabled {
		if err := f.reloadSchedule(); err != nil {
			f.logger.Error("reload schedule after delete", "error", err)
		}
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (f *Functions) handleRun(w http.ResponseWriter, r *http.Request, id string) {
	fn, err := f.fetchFunctionByID(r.Context(), id)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	rt, ok := f.runtimes[fn.Runtime]
	if !ok {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported runtime: " + fn.Runtime})
		return
	}

	runCtx := RunContext{
		Env:     fn.Env,
		DB:      f.db,
		Request: r,
	}
	output, execErr := rt.Execute(fn.Source, fn.Timeout, runCtx)

	// Persist execution history
	execID, saveErr := f.saveExecution(r.Context(), fn.ID, output, execErr)
	if saveErr != nil {
		f.logger.Error("save execution", "error", saveErr)
	}

	if execErr != nil {
		kernel.WriteJSON(w, http.StatusOK, map[string]any{
			"execution_id": execID,
			"logs":         output.Logs,
			"error":        execErr.Error(),
			"result":       nil,
		})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"execution_id": execID,
		"logs":         output.Logs,
		"error":        nil,
		"result":       output.Result,
	})
}

func (f *Functions) saveExecution(ctx context.Context, fnID string, output *RunOutput, execErr error) (string, error) {
	id, err := kernel.GenerateID()
	if err != nil {
		return "", err
	}
	status := "success"
	errMsg := ""
	if execErr != nil {
		status = "error"
		errMsg = execErr.Error()
	}

	// Nil-guard: runtime may return nil output on hard failures
	logs := []string{}
	if output != nil {
		logs = output.Logs
	}

	logsJSON, marshalErr := json.Marshal(logs)
	if marshalErr != nil {
		return "", fmt.Errorf("marshal logs: %w", marshalErr)
	}
	_, dbErr := f.db.ExecContext(ctx,
		"INSERT INTO function_executions (id, function_id, status, logs, error, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
		id, fnID, status, string(logsJSON), errMsg)
	if dbErr != nil {
		return "", dbErr
	}
	return id, nil
}

func (f *Functions) handleFunctionLogs(w http.ResponseWriter, r *http.Request, fnID string) {
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Verify function exists
	_, err := f.fetchFunctionByID(r.Context(), fnID)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	rows, dberr := f.db.QueryContext(r.Context(),
		"SELECT id, status, logs, error, created_at FROM function_executions WHERE function_id = ? ORDER BY created_at DESC LIMIT 100",
		fnID)
	if dberr != nil {
		f.logger.Error("query logs", "error", dberr)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	type executionEntry struct {
		ID        string   `json:"id"`
		Status    string   `json:"status"`
		Logs      []string `json:"logs"`
		Error     string   `json:"error,omitempty"`
		CreatedAt string   `json:"created_at"`
	}

	results := make([]executionEntry, 0)
	for rows.Next() {
		var e executionEntry
		var logsStr string
		if scanErr := rows.Scan(&e.ID, &e.Status, &logsStr, &e.Error, &e.CreatedAt); scanErr != nil {
			f.logger.Error("scan log row", "error", scanErr)
			continue
		}
		if logsStr != "" {
			_ = json.Unmarshal([]byte(logsStr), &e.Logs)
		}
		if e.Logs == nil {
			e.Logs = []string{}
		}
		results = append(results, e)
	}

	if err := rows.Err(); err != nil {
		f.logger.Error("rows iteration error", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": results})
}
