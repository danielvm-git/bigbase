package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type contextKey string

const (
	ctxOrgID    contextKey = "org_id"
	ctxUserRole contextKey = "user_role"
)

// WithOrgID stores the org_id in the request context.
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	return context.WithValue(ctx, ctxOrgID, orgID)
}

// OrgIDFromContext retrieves the org_id from the request context.
// Returns (0, false) when no org_id is set.
func OrgIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxOrgID).(int64)
	return id, ok
}

// WithUserRole stores the user role in the request context.
func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxUserRole, role)
}

// UserRoleFromContext retrieves the user role from the request context.
// Returns ("", false) when no role is set.
func UserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(ctxUserRole).(string)
	return role, ok
}

var internalTables = map[string]bool{
	"users": true, "storage_files": true,
	"git_repos": true, "git_ssh_keys": true,
	"forge_issues": true, "forge_labels": true,
	"forge_comments": true, "forge_wiki": true,
	"cici_logs": true, "cici_runs": true, "cici_workflows": true,
	"deployments": true, "functions": true, "messages": true,
}

// Pre-compiled regex patterns for SQL endpoint security checks.
var (
	internalTableRegexes  []*regexp.Regexp
	blockedKeywordRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bJOIN\b`),
		regexp.MustCompile(`(?i)\bUNION\b`),
		regexp.MustCompile(`(?i)\bINTERSECT\b`),
		regexp.MustCompile(`(?i)\bEXCEPT\b`),
		regexp.MustCompile(`(?i)\bWITH\b`),
	}
	selectCountRegex = regexp.MustCompile(`(?i)\bSELECT\b`)
)

func init() {
	for t := range internalTables {
		internalTableRegexes = append(internalTableRegexes,
			regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(t)+`\b`))
	}
}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Options struct {
	DB     DBer
	Logger kernel.Logger
}

type API struct {
	db     DBer
	logger kernel.Logger
	tables map[string]bool
	bus    *kernel.EventBus
}

var _ kernel.Component = (*API)(nil)

func New(opts Options) *API {
	return &API{
		db:     opts.DB,
		logger: opts.Logger,
		tables: make(map[string]bool),
	}
}

func (a *API) Name() string                  { return "api" }
func (a *API) Version() string               { return version }
func (a *API) Dependencies() []string        { return []string{"db"} }

func (a *API) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (a *API) Start(ctx *kernel.Context) error {
	a.bus = ctx.Kernel.EventBus()
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
	mux.HandleFunc("/api/onboarding", a.handleOnboarding)
	mux.HandleFunc("/api/scaffold/db", a.handleScaffoldDB)
	mux.HandleFunc("/api/scaffold/repo", a.handleScaffoldRepo)
	mux.HandleFunc("/api/scaffold/function", a.handleScaffoldFunction)
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
		org_id INTEGER NOT NULL DEFAULT 0,
		data TEXT
	)`, collection)
	if err := a.db.Migrate(query); err != nil {
		return err
	}
	// Best-effort: add org_id to pre-existing tables (ignored if column already exists)
	_ = a.db.Migrate(fmt.Sprintf("ALTER TABLE %s ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0", collection))
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
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if internalTables[collection] {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
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
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		}
	case "DELETE":
		if len(parts) == 2 && parts[1] != "" {
			a.deleteRecord(w, r, collection, parts[1])
		} else {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		}
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *API) listRecords(w http.ResponseWriter, r *http.Request, collection string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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

	// Parse filter and sort params
	filters, filterArgs := parseFilters(r, collection)
	sortClause := parseSort(r, collection)

	// Build WHERE clause
	whereClause := ""
	args := []any{}
	if orgID, ok := OrgIDFromContext(r.Context()); ok {
		whereClause = " WHERE org_id = ?"
		args = append(args, orgID)
	}
	if len(filters) > 0 {
		if whereClause == "" {
			whereClause = " WHERE " + strings.Join(filters, " AND ")
		} else {
			whereClause += " AND " + strings.Join(filters, " AND ")
		}
		args = append(args, filterArgs...)
	}

	query := fmt.Sprintf("SELECT id, data FROM %s%s%s LIMIT ? OFFSET ?", collection, whereClause, sortClause)
	args = append(args, limit, offset)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		a.logger.Error("list records query", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
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
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (a *API) getRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Fetch record with org_id for isolation check
	row := a.db.QueryRowContext(ctx, fmt.Sprintf("SELECT id, org_id, data FROM %s WHERE id = ?", collection), id)
	var recordID, recordOrgID int64
	var dataStr string
	if err := row.Scan(&recordID, &recordOrgID, &dataStr); err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if orgID, ok := OrgIDFromContext(r.Context()); ok && recordOrgID != orgID {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return
	}
	record := map[string]any{"id": recordID}
	var data map[string]any
	if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
		for k, v := range data {
			record[k] = v
		}
	}
	kernel.WriteJSON(w, http.StatusOK, record)
}

func (a *API) emitMutation(mutType, collection string, id any, siteID string) {
	if a.bus != nil {
		data := map[string]any{
			"collection": collection,
			"type":       mutType,
			"id":         id,
		}
		if siteID != "" {
			data["site_id"] = siteID
		}
		if err := a.bus.Emit(kernel.Event{Name: "mutation", Data: data}, nil); err != nil {
			a.logger.Error("emit mutation event", "collection", collection, "type", mutType, "id", id, "error", err)
		}
	}
}

func siteIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Bigbase-Site-ID"))
}

func (a *API) createRecord(w http.ResponseWriter, r *http.Request, collection string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}

	dataBytes, _ := json.Marshal(body)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orgID, _ := OrgIDFromContext(r.Context())
	res, err := a.db.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (org_id, data) VALUES (?, ?)", collection),
		orgID, string(dataBytes))
	if err != nil {
		a.logger.Error("insert record", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	id, _ := res.LastInsertId()
	a.emitMutation("create", collection, id, siteIDFromRequest(r))
	kernel.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (a *API) updateRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.db.QueryRowContext(ctx, fmt.Sprintf("SELECT org_id, data FROM %s WHERE id = ?", collection), id)
	var existingOrgID int64
	var existingStr string
	if err := row.Scan(&existingOrgID, &existingStr); err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if orgID, ok := OrgIDFromContext(r.Context()); ok && existingOrgID != orgID {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
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
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.emitMutation("update", collection, id, siteIDFromRequest(r))
	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (a *API) deleteRecord(w http.ResponseWriter, r *http.Request, collection, id string) {
	if err := a.ensureTable(collection); err != nil {
		a.logger.Error("ensure table", "collection", collection, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Enforce org isolation: check ownership before deleting
	if orgID, ok := OrgIDFromContext(r.Context()); ok {
		var ownerOrgID int64
		err := a.db.QueryRowContext(ctx,
			fmt.Sprintf("SELECT org_id FROM %s WHERE id = ?", collection), id).Scan(&ownerOrgID)
		if err != nil {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if ownerOrgID != orgID {
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
			return
		}
	}

	res, err := a.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", collection), id)
	if err != nil {
		a.logger.Error("delete record", "collection", collection, "id", id, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	a.emitMutation("delete", collection, id, siteIDFromRequest(r))
	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) listCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	names, err := a.fetchCollectionNames(ctx)
	if err != nil {
		a.logger.Error("fetch collection names", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": names})
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
		if !internalTables[name] {
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
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Defense-in-depth: require admin role even though the route-level middleware
	// already gates on admin. This protects against future route misconfigurations.
	if role, ok := UserRoleFromContext(r.Context()); !ok || role != "admin" {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json or body too large"})
		return
	}

	q := strings.TrimSpace(req.Query)
	if q == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "query is required"})
		return
	}

	// Match internal tables using word boundaries to avoid false positives (e.g. my_users)
	for t := range internalTables {
		pattern := `(?i)\b` + regexp.QuoteMeta(t) + `\b`
		re, err := regexp.Compile(pattern)
		if err == nil && re.MatchString(q) {
			kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access to internal table " + t + " denied"})
			return
		}
	}

	if strings.Contains(q, ";") {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "only single-statement queries are allowed"})
		return
	}

	firstWord := strings.ToUpper(strings.Fields(q)[0])
	switch firstWord {
	case "SELECT", "EXPLAIN", "PRAGMA", "WITH":
	default:
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "only read-only queries are allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	execQuery := q
	var execArgs []any

	if orgID, orgOK := OrgIDFromContext(r.Context()); orgOK {
		// Normalize whitespace to single spaces to prevent bypasses like FROM\nitems
		normalized := strings.Join(strings.Fields(q), " ")
		tableRef := ExtractTableRef(normalized)

		if tableRef != "" && !isInternalOrSysTable(tableRef) {
			// Accesses a user collection table. Enforce strict isolation controls.
			// Block UNION, JOIN, WITH, and subqueries to prevent scoping bypasses.
			if !isSimpleQuery(normalized) {
				kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "only simple queries (no JOIN, UNION, WITH, or subqueries) are allowed on collection tables for tenant isolation"})
				return
			}
			execQuery, execArgs = scopeQueryForOrg(normalized, orgID)
			a.logger.Info("sql org-scoped", "table", tableRef, "org_id", orgID)
		}
	}

	rows, err := a.db.QueryContext(ctx, execQuery, execArgs...)
	if err != nil {
		a.logger.Error("sql execution failed", "query", q, "error", err)
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "query execution failed"})
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
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	result, err := rowsToMaps(rows, columns)
	if err != nil {
		a.logger.Error("rows to maps in handleSQL", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"columns": columns, "rows": result})
}

// isSimpleQuery returns true if the query contains no JOIN, UNION, INTERSECT,
// EXCEPT, or WITH keywords and does not contain subqueries.
func isSimpleQuery(q string) bool {
	for _, re := range blockedKeywordRegexes {
		if re.MatchString(q) {
			return false
		}
	}
	// A simple query contains at most one SELECT keyword.
	matches := selectCountRegex.FindAllStringIndex(q, -1)
	return len(matches) <= 1
}

func ExtractTableRef(q string) string {
	// Normalize all whitespace to single spaces
	normalized := strings.Join(strings.Fields(q), " ")
	upper := strings.ToUpper(normalized)

	// Handle WITH queries: find the SELECT after WITH
	if strings.HasPrefix(upper, "WITH") {
		selIdx := strings.LastIndex(upper, "SELECT")
		if selIdx < 0 {
			return ""
		}
		upper = upper[selIdx:]
		normalized = normalized[selIdx:]
	}

	// Handle EXPLAIN
	if strings.HasPrefix(upper, "EXPLAIN QUERY PLAN ") {
		upper = strings.TrimPrefix(upper, "EXPLAIN QUERY PLAN ")
		normalized = normalized[len("EXPLAIN QUERY PLAN "):]
	} else if strings.HasPrefix(upper, "EXPLAIN ") {
		upper = strings.TrimPrefix(upper, "EXPLAIN ")
		normalized = normalized[len("EXPLAIN "):]
	}

	// Find FROM keyword
	fromIdx := strings.Index(upper, " FROM ")
	if fromIdx < 0 {
		return ""
	}
	afterFrom := strings.TrimSpace(normalized[fromIdx+6:])

	// Extract the table name before any JOIN, WHERE, GROUP, ORDER, LIMIT, etc.
	endKeywords := []string{" JOIN ", " WHERE ", " GROUP ", " ORDER ", " LIMIT ", " OFFSET ", " HAVING ", " UNION ", " INTERSECT ", " EXCEPT "}
	endIdx := len(afterFrom)
	upperAfterFrom := strings.ToUpper(afterFrom)
	for _, kw := range endKeywords {
		if idx := strings.Index(upperAfterFrom, kw); idx >= 0 && idx < endIdx {
			endIdx = idx
		}
	}

	table := strings.TrimSpace(afterFrom[:endIdx])
	// Handle quoted table names
	table = strings.Trim(table, "\"`'")
	return strings.ToLower(table)
}

// scopeQueryForOrg injects an org_id filter into a read-only SQL query.
// Returns the scoped query string and bind arguments. Works by inserting
// WHERE org_id = ? (or appending AND org_id = ? if WHERE exists) before
// GROUP/HAVING/ORDER/LIMIT clauses or at the end of the query.
func scopeQueryForOrg(q string, orgID int64) (string, []any) {
	// Normalize all whitespace to single spaces to simplify parsing
	normalized := strings.Join(strings.Fields(q), " ")
	upper := strings.ToUpper(normalized)

	// Find insertion point: right before GROUP BY, HAVING, ORDER BY, LIMIT, or end
	keywords := []string{" GROUP BY ", " HAVING ", " ORDER BY ", " LIMIT "}
	insertPoint := len(normalized)
	for _, kw := range keywords {
		if idx := strings.Index(upper, kw); idx >= 0 && idx < insertPoint {
			insertPoint = idx
		}
	}

	// Trim trailing whitespace before insertion point
	for insertPoint > 0 && normalized[insertPoint-1] == ' ' {
		insertPoint--
	}

	// Check if WHERE already exists
	whereIdx := strings.Index(upper, " WHERE ")
	if whereIdx >= 0 && whereIdx < insertPoint {
		// Append AND org_id = ? before the next clause
		return normalized[:insertPoint] + " AND org_id = ? " + normalized[insertPoint:], []any{orgID}
	}

	// No WHERE clause — insert one
	return normalized[:insertPoint] + " WHERE org_id = ? " + normalized[insertPoint:], []any{orgID}
}

// isInternalOrSysTable returns true if the table is an internal component table
// or a sqlite system table.
func isInternalOrSysTable(name string) bool {
	if internalTables[name] {
		return true
	}
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "sqlite_") || lower == "" || lower == "dual"
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

// parseFilters extracts filter query params and returns SQL WHERE clauses
// with json_extract and corresponding bind arguments.
// Each ?filter=field=value becomes: json_extract(data, '$.field') = ?
// Each ?filter=field:op=value becomes: json_extract(data, '$.field') OP ?
func parseFilters(r *http.Request, collection string) ([]string, []any) {
	var clauses []string
	var args []any

	for _, raw := range r.URL.Query()["filter"] {
		field, op, value := parseFilterExpr(raw)

		// Validate field name
		if _, err := sanitize(field); err != nil {
			continue // skip invalid field names
		}

		extract := fmt.Sprintf("json_extract(data, '$.%s')", field)

		switch op {
		case "eq", "":
			if value == "" {
				clauses = append(clauses, extract+" IS NULL OR "+extract+" = ''")
			} else {
				clauses = append(clauses, extract+" = ?")
				args = append(args, value)
			}
		case "gt":
			clauses = append(clauses, "CAST("+extract+" AS REAL) > ?")
			args = append(args, value)
		case "gte":
			clauses = append(clauses, "CAST("+extract+" AS REAL) >= ?")
			args = append(args, value)
		case "lt":
			clauses = append(clauses, "CAST("+extract+" AS REAL) < ?")
			args = append(args, value)
		case "lte":
			clauses = append(clauses, "CAST("+extract+" AS REAL) <= ?")
			args = append(args, value)
		case "ne":
			clauses = append(clauses, extract+" != ?")
			args = append(args, value)
		case "like":
			clauses = append(clauses, extract+" LIKE ?")
			args = append(args, "%"+value+"%")
		}
	}

	return clauses, args
}

// parseFilterExpr splits a filter expression like "field=value" or "field:op=value"
// into (field, op, value). Default op is "eq".
func parseFilterExpr(raw string) (field, op, value string) {
	// Check for operator syntax: field:op=value
	firstColon := strings.Index(raw, ":")
	firstEq := strings.Index(raw, "=")

	if firstColon > 0 && (firstEq < 0 || firstColon < firstEq) {
		// Has operator: field:op=value
		field = raw[:firstColon]
		rest := raw[firstColon+1:]
		eqIdx := strings.Index(rest, "=")
		if eqIdx >= 0 {
			op = rest[:eqIdx]
			value = rest[eqIdx+1:]
		} else {
			op = "eq"
			value = rest
		}
	} else if firstEq > 0 {
		// Simple equality: field=value
		field = raw[:firstEq]
		value = raw[firstEq+1:]
		op = "eq"
	} else {
		field = raw
		op = "eq"
	}
	return field, op, value
}

// parseSort extracts the sort query param and returns an ORDER BY clause.
// ?sort=field → ORDER BY json_extract(data, '$.field') ASC
// ?sort=-field → ORDER BY json_extract(data, '$.field') DESC
// No sort param → ORDER BY id (default)
func parseSort(r *http.Request, collection string) string {
	raw := r.URL.Query().Get("sort")
	if raw == "" {
		return " ORDER BY id"
	}

	desc := false
	field := raw
	if strings.HasPrefix(raw, "-") {
		desc = true
		field = raw[1:]
	}

	// Validate field name
	if _, err := sanitize(field); err != nil {
		return " ORDER BY id" // fallback for invalid field
	}

	dir := "ASC"
	if desc {
		dir = "DESC"
	}

	return fmt.Sprintf(" ORDER BY json_extract(data, '$.%s') %s", field, dir)
}
