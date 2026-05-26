package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type DBer interface {
	DB() *sql.DB
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
}

func New(opts Options) *API {
	return &API{
		db:     opts.DB,
		logger: opts.Logger,
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
	return mux
}

func (a *API) ensureTable(collection string) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		data TEXT
	)`, sanitize(collection))
	return a.db.Migrate(query)
}

func (a *API) handleCollection(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	parts := strings.SplitN(path, "/", 2)
	collection := sanitize(parts[0])
	if collection == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "collection name required"})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := a.db.Query(fmt.Sprintf("SELECT id, data FROM %s ORDER BY id", sanitize(collection)))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer func() { _ = rows.Close() }()

	result := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var dataStr string
		if err := rows.Scan(&id, &dataStr); err != nil {
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
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (a *API) getRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	row := a.db.QueryRow(fmt.Sprintf("SELECT id, data FROM %s WHERE id = ?", sanitize(collection)), id)
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	dataBytes, _ := json.Marshal(body)
	res, err := a.db.Exec(fmt.Sprintf("INSERT INTO %s (data) VALUES (?)", sanitize(collection)), string(dataBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (a *API) updateRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// read existing data
	row := a.db.QueryRow(fmt.Sprintf("SELECT data FROM %s WHERE id = ?", sanitize(collection)), id)
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
	_, err := a.db.Exec(fmt.Sprintf("UPDATE %s SET data = ? WHERE id = ?", sanitize(collection)), string(merged), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (a *API) deleteRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_, err := a.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", sanitize(collection)), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
