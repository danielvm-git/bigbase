package git

// Regression test (e81s06) for the repo-id path-traversal guard in deleteRepo.
// Repo ids are normally server-generated and cannot contain "/" or "..", but
// the handler still cleans and rejects such ids as defense-in-depth before the
// filesystem RemoveAll. Repo ids that escape must yield 400, never touch the
// filesystem. We seed rows directly because the public API never mints such ids.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func TestPathTraversal(t *testing.T) {
	logger := kernel.NoopLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := New(Options{DB: d, Logger: logger, Dir: t.TempDir()})
	k.Register(d)
	k.Register(g)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	badIDs := []string{"../evil", "a/b", "../../etc/passwd", "sub/../../escape"}
	for i, badID := range badIDs {
		if _, err := g.db.ExecContext(context.Background(),
			"INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			badID, "seeded-repo-"+string(rune('a'+i)), 0, 0, "main", "", "2026-01-01T00:00:00Z",
		); err != nil {
			t.Fatalf("seed row %q: %v", badID, err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/git/repos/x", nil)
		w := httptest.NewRecorder()
		g.deleteRepo(w, req, badID)

		if w.Code != http.StatusBadRequest {
			t.Errorf("deleteRepo(id=%q): got %d, want 400 (path traversal must be rejected)", badID, w.Code)
		}
	}
}
