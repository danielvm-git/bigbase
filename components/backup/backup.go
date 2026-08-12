// Package backup provides SQLite dump/restore functionality for BigBase.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/danielvm/bigbase/kernel"
)

// SQLiteDB is a *sql.DB opened against a SQLite file.
// Exposed so main.go can hold the type without importing database/sql directly.
type SQLiteDB = *sql.DB

// DBer is the database abstraction used by dump/restore.
type DBer = kernel.QueryExecDBer

// OpenSQLite opens a SQLite database at path and returns the raw *sql.DB.
func OpenSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// Dump writes a SQL dump of all user tables in db to w.
// The output is a series of CREATE TABLE + INSERT statements that can be
// replayed against a SQLite database to restore the schema and data.
func Dump(ctx context.Context, db DBer, w io.Writer) error {
	tableRows, err := db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	defer func() { _ = tableRows.Close() }()

	var tables []string
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err != nil {
			return fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	if err := tableRows.Err(); err != nil {
		return fmt.Errorf("iterate tables: %w", err)
	}
	_ = tableRows.Close()

	if _, err := fmt.Fprintf(w, "-- BigBase SQLite dump %s;\nPRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\n\n",
		time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, table := range tables {
		if err := dumpTable(ctx, db, table, w); err != nil {
			return fmt.Errorf("dump table %q: %w", table, err)
		}
	}

	if _, err := fmt.Fprintln(w, "COMMIT;"); err != nil {
		return fmt.Errorf("write footer: %w", err)
	}
	return nil
}

func dumpTable(ctx context.Context, db DBer, table string, w io.Writer) error {
	// Write CREATE TABLE statement.
	var ddl string
	ddlRows, err := db.QueryContext(ctx,
		"SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table)
	if err != nil {
		return fmt.Errorf("get DDL: %w", err)
	}
	defer func() { _ = ddlRows.Close() }()

	if ddlRows.Next() {
		if err := ddlRows.Scan(&ddl); err != nil {
			return fmt.Errorf("scan DDL: %w", err)
		}
	}
	_ = ddlRows.Close()

	if ddl == "" {
		return nil
	}

	if _, err := fmt.Fprintf(w, "DROP TABLE IF EXISTS %q;\n%s;\n\n", table, ddl); err != nil {
		return fmt.Errorf("write DDL: %w", err)
	}

	// Fetch all rows.
	colRows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %q", table))
	if err != nil {
		return fmt.Errorf("select rows: %w", err)
	}
	defer func() { _ = colRows.Close() }()

	columns, err := colRows.Columns()
	if err != nil {
		return fmt.Errorf("get columns: %w", err)
	}

	for colRows.Next() {
		vals := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := colRows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}

		quoted := make([]string, len(vals))
		for i, v := range vals {
			quoted[i] = sqlLiteral(v)
		}

		if _, err := fmt.Fprintf(w, "INSERT INTO %q VALUES (%s);\n",
			table, strings.Join(quoted, ",")); err != nil {
			return fmt.Errorf("write insert: %w", err)
		}
	}
	if err := colRows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %w", err)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	return nil
}

// Restore executes the SQL statements in sqlDump against db.
// It runs each non-empty, non-comment line as a separate statement.
func Restore(ctx context.Context, db DBer, sqlDump string) error {
	// Split on semicolons to get individual statements.
	stmts := splitSQL(sqlDump)
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", truncate(stmt, 80), err)
		}
	}
	return nil
}

// splitSQL splits a SQL dump on semicolons, respecting quoted strings.
func splitSQL(s string) []string {
	var stmts []string
	var cur strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\'' && !inStr {
			inStr = true
			cur.WriteByte(ch)
		} else if ch == '\'' && inStr {
			cur.WriteByte(ch)
			if i+1 < len(s) && s[i+1] == '\'' {
				// escaped quote
				i++
				cur.WriteByte(s[i])
			} else {
				inStr = false
			}
		} else if ch == ';' && !inStr {
			stmts = append(stmts, cur.String())
			cur.Reset()
		} else {
			cur.WriteByte(ch)
		}
	}
	if remaining := strings.TrimSpace(cur.String()); remaining != "" {
		stmts = append(stmts, remaining)
	}
	return stmts
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// sqlLiteral converts a Go value to a SQL literal string safe to embed in a dump.
func sqlLiteral(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case []byte:
		return sqlQuoteString(string(val))
	case string:
		return sqlQuoteString(val)
	default:
		return sqlQuoteString(fmt.Sprintf("%v", v))
	}
}

// sqlQuoteString returns a SQL-quoted string literal.
func sqlQuoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
