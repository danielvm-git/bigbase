package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type DBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Migrate(migration string) error
}

type Options struct {
	DB     DBer
	Logger Logger
}

type API struct {
	db     DBer
	logger Logger
	tables map[string]bool
}

func New(opts Options) *API {
	return &API{
		db:     opts.DB,
		logger: opts.Logger,
		tables: make(map[string]bool),
	}
}

func (a *API) Name() string                 { return "api" }
func (a *API) Version() string              { return version }
func (a *API) Dependencies() []string       { return []string{"db"} }
func (a *API) ConfigSchema() json.RawMessage { return nil }
func (a *API) Hooks() []kernel.HookDef      { return nil }

func (a *API) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (a *API) Start(ctx *kernel.Context) error {
	a.logger.Info("api component ready")
	return nil
}

func (a *API) Stop(ctx *kernel.Context) error {
	return nil
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/collections/", a.handleCollection)
	mux.HandleFunc("/api/sql", a.handleSQL)
	return mux
}

func (a *API) ensureTable(collection string) error {
	if _, err := sanitize(collection); err != nil {
		return err
	}
	if a.tables[collection] {
		return nil
	}
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		data TEXT
	)`, collection)
	if err := a.db.Migrate(query); err != nil {
		return err
	}
	a.tables[collection] = true
	return nil
}

func (a *API) handleCollection(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	parts := strings.SplitN(path, "/", 2)
	if parts[0] == "" {
		a.listCollections(w, r)
		return
	}

	collection, err := sanitize(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	switch r.Method {
	case "GET":
		if len(parts) == 2 && parts[1] != "" {
			a.getRecord(w, r, collection, parts[1])
		} else {
			a.listRecords(w, r, collection)
		}
	case "POST":
		a.createRecord(w, r, collection)
	case "PATCH":
		if len(parts) == 2 && parts[1] != "" {
			a.updateRecord(w, r, collection, parts[1])
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		}
	case "DELETE":
		if len(parts) == 2 && parts[1] != "" {
			a.deleteRecord(w, r, collection, parts[1])
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		}
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) listRecords(w http.ResponseWriter, r *http.Request, collection string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := a.db.QueryContext(ctx, fmt.Sprintf("SELECT id, data FROM %s ORDER BY id LIMIT ? OFFSET ?", collection), limit, offset)
	if err != nil {
		a.logger.Error("list records query", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			a.logger.Error("close rows in listRecords", "collection", collection, "error", cerr)
		}
	}()

	result := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var dataStr string
		if err := rows.Scan(&id, &dataStr); err != nil {
			a.logger.Error("scan row in listRecords", "collection", collection, "error", err)
			continue
		}
		record := map[string]any{"id": id}
		var data map[string]any
		if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
			for k, v := range data {
				record[k] = v
			}
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		a.logger.Error("iterate records", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (a *API) getRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.db.QueryRowContext(ctx, fmt.Sprintf("SELECT id, data FROM %s WHERE id = ?", collection), id)
	var recordID int64
	var dataStr string
	if err := row.Scan(&recordID, &dataStr); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	record := map[string]any{"id": recordID}
	var data map[string]any
	if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
		for k, v := range data {
			record[k] = v
		}
	}
	writeJSON(w, http.StatusOK, record)
}

func (a *API) createRecord(w http.ResponseWriter, r *http.Request, collection string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}

	dataBytes, _ := json.Marshal(body)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := a.db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (data) VALUES (?)", collection), string(dataBytes))
	if err != nil {
		a.logger.Error("insert record", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (a *API) updateRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.db.QueryRowContext(ctx, fmt.Sprintf("SELECT data FROM %s WHERE id = ?", collection), id)
	var existingStr string
	if err := row.Scan(&existingStr); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	var existing map[string]any
	_ = json.Unmarshal([]byte(existingStr), &existing)
	for k, v := range body {
		existing[k] = v
	}

	merged, _ := json.Marshal(existing)
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET data = ? WHERE id = ?", collection), string(merged), id)
	if err != nil {
		a.logger.Error("update record", "collection", collection, "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (a *API) deleteRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := a.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", collection), id)
	if err != nil {
		a.logger.Error("delete record", "collection", collection, "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) listCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	names, err := a.fetchCollectionNames(ctx)
	if err != nil {
		a.logger.Error("fetch collection names", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": names})
}

func (a *API) fetchCollectionNames(ctx context.Context) ([]string, error) {
	rows, err := a.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			a.logger.Error("close rows in fetchCollectionNames", "error", cerr)
		}
	}()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			a.logger.Error("scan collection name", "error", err)
			continue
		}
		if name != "users" {
			result = append(result, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *API) handleSQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}

	q := strings.TrimSpace(req.Query)
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query is required"})
		return
	}

	sensitiveTables := []string{"users", "storage_files"}
	for _, t := range sensitiveTables {
		if strings.Contains(strings.ToUpper(q), strings.ToUpper(t)) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "access to internal tables denied"})
			return
		}
	}

	if strings.Contains(q, ";") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only single-statement queries are allowed"})
		return
	}

	firstWord := strings.ToUpper(strings.Fields(q)[0])
	switch firstWord {
	case "SELECT", "EXPLAIN", "PRAGMA", "WITH":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only read-only queries are allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := a.db.QueryContext(ctx, q)
	if err != nil {
		a.logger.Error("sql execution failed", "query", q, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query execution failed"})
		return
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			a.logger.Error("close rows in handleSQL", "error", cerr)
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		a.logger.Error("get columns in handleSQL", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	result, err := rowsToMaps(rows, columns)
	if err != nil {
		a.logger.Error("rows to maps in handleSQL", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"columns": columns, "rows": result})
}

func rowsToMaps(rows *sql.Rows, columns []string) ([]map[string]any, error) {
	// Uses any because SQLite column types are dynamic at runtime.
	result := make([]map[string]any, 0)
	for rows.Next() {
		vals := make([]any, len(columns))
		valPtrs := make([]any, len(columns))
		for i := range columns {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			val := vals[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			row[col] = val
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func sanitize(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("collection name required")
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return "", fmt.Errorf("invalid character %q in collection name", r)
		}
	}
	return name, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
