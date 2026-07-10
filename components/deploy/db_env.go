package deploy

import (
	"path/filepath"
	"strings"
)

const (
	driverSQLite   = "sqlite"
	driverPostgres = "postgres"
)

// NativeDBEnv returns KEY=value env pairs for direct database access in deployed apps.
// SQLite uses DB_PATH; PostgreSQL uses DATABASE_URL. Returns nil when dsn is empty.
func NativeDBEnv(driver, dsn string) []string {
	if dsn == "" {
		return nil
	}
	switch normalizeDBDriver(driver) {
	case driverPostgres:
		return []string{"DATABASE_URL=" + dsn}
	default:
		path := dsn
		if path != ":memory:" {
			if abs, err := filepath.Abs(path); err == nil {
				path = abs
			}
		}
		return []string{"DB_PATH=" + path}
	}
}

func (d *Deploy) nativeDBEnv() []string {
	return NativeDBEnv(d.dbDriver, d.dbDSN)
}

func normalizeDBDriver(driver string) string {
	d := strings.ToLower(strings.TrimSpace(driver))
	if d == "" {
		return driverSQLite
	}
	return d
}
