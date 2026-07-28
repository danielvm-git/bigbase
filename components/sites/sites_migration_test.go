package sites

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sourceFile returns the absolute path of a .go file in this package.
// It locates the package directory via this test file's own location, so the
// guard works regardless of the module checkout path.
func sourceFile(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	p := filepath.Join(dir, name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// TestSitesMigrationConstExtracted guards BUG-2026-07-10T160102: the inline
// CREATE TABLE migration that lived inside Start() must be extracted into the
// package-level sitesMigration const (mirroring domainsMigration), and Start()
// must invoke Migrate with that const rather than an inline SQL literal.
//
// The assertion is purely structural against sites.go's source so it stays
// green when the refactor is reverted (inline literal returns) and green again
// once the const is reintroduced — i.e. it fails if and only if the migration
// SQL is inlined in Start().
func TestSitesMigrationConstExtracted(t *testing.T) {
	src := sourceFile(t, "sites.go")

	if !strings.Contains(src, "const sitesMigration =") {
		t.Errorf("sites.go must declare a package-level `const sitesMigration = ...` (mirror domainsMigration); not found")
	}

	// The CREATE TABLE for sites must no longer be inlined inside Start();
	// it must live only in the const. We anchor on the table-defining statement.
	const inlineSitesCreate = "CREATE TABLE IF NOT EXISTS sites ("
	startIdx := strings.Index(src, "func (s *Sites) Start(")
	if startIdx < 0 {
		t.Fatal("could not locate Start() in sites.go")
	}
	// Bound the search to the Start method body (up to the next top-level func).
	rest := src[startIdx:]
	if end := strings.Index(rest, "\nfunc "); end > 0 {
		rest = rest[:end]
	}
	if strings.Contains(rest, inlineSitesCreate) {
		t.Errorf("Start() still inlines %q; it must call s.db.Migrate(sitesMigration) instead", inlineSitesCreate)
	}

	// Start() must reference the const by name.
	if !strings.Contains(rest, "sitesMigration") {
		t.Errorf("Start() does not reference the sitesMigration const; expected s.db.Migrate(sitesMigration)")
	}
}
