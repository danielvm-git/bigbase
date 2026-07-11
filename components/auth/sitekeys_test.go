package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func setupSiteKeys(t *testing.T) (*auth.Auth, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(a)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	_, err := d.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("create sites table: %v", err)
	}
	_, err = d.Exec(`INSERT INTO sites (id, name, git_repo_id) VALUES ('site-1', 'app', 'repo-1')`)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	return a, d
}

func TestSiteKeyCreate(t *testing.T) {
	a, _ := setupSiteKeys(t)
	ctx := context.Background()

	token, keyID, err := a.CreateSiteKey(ctx, "site-1", "ci-bot", []string{"deploy"})
	if err != nil {
		t.Fatalf("CreateSiteKey: %v", err)
	}
	if keyID == "" {
		t.Fatal("expected key_id")
	}
	if !strings.HasPrefix(token, "bb_dep_") || len(token) != len("bb_dep_")+64 {
		t.Fatalf("unexpected token format: %q", token)
	}
}

func TestSiteKeyCreateMissingSite(t *testing.T) {
	a, _ := setupSiteKeys(t)
	_, _, err := a.CreateSiteKey(context.Background(), "missing", "ci", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestSiteKeyResolve(t *testing.T) {
	a, d := setupSiteKeys(t)
	ctx := context.Background()

	token, _, err := a.CreateSiteKey(ctx, "site-1", "ci", []string{"deploy"})
	if err != nil {
		t.Fatalf("CreateSiteKey: %v", err)
	}

	siteID, err := a.ResolveSiteKey(token)
	if err != nil {
		t.Fatalf("ResolveSiteKey: %v", err)
	}
	if siteID != "site-1" {
		t.Fatalf("expected site-1, got %q", siteID)
	}

	var storedHash string
	if err := d.QueryRow(`SELECT key_hash FROM org_api_keys WHERE site_id = 'site-1'`).Scan(&storedHash); err != nil {
		t.Fatalf("query key_hash: %v", err)
	}
	if storedHash == token {
		t.Fatal("raw token must not be stored in key_hash column")
	}
	if len(storedHash) != 64 {
		t.Fatalf("expected sha256 hex hash, got len %d", len(storedHash))
	}
}

func TestSiteKeyMiddleware(t *testing.T) {
	a, _ := setupSiteKeys(t)
	ctx := context.Background()
	token, _, err := a.CreateSiteKey(ctx, "site-1", "ci", nil)
	if err != nil {
		t.Fatalf("CreateSiteKey: %v", err)
	}

	var gotSiteID string
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := kernel.SiteIDFromContext(r.Context())
		if !ok {
			t.Error("expected site id in context")
		}
		gotSiteID = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotSiteID != "site-1" {
		t.Fatalf("expected site-1 in context, got %q", gotSiteID)
	}
}

func TestSiteKeyRevoked(t *testing.T) {
	a, d := setupSiteKeys(t)
	ctx := context.Background()
	token, keyID, err := a.CreateSiteKey(ctx, "site-1", "ci", nil)
	if err != nil {
		t.Fatalf("CreateSiteKey: %v", err)
	}
	if err := a.RevokeSiteKey(ctx, "site-1", keyID); err != nil {
		t.Fatalf("RevokeSiteKey: %v", err)
	}
	if _, err := a.ResolveSiteKey(token); err == nil {
		t.Fatal("expected error for revoked key")
	}
	var revoked int
	if err := d.QueryRow(`SELECT revoked FROM org_api_keys WHERE id = ?`, keyID).Scan(&revoked); err != nil {
		t.Fatalf("query revoked: %v", err)
	}
	if revoked != 1 {
		t.Fatalf("expected revoked=1, got %d", revoked)
	}
}

func TestListSiteKeys(t *testing.T) {
	a, _ := setupSiteKeys(t)
	ctx := context.Background()

	_, keyID1, err := a.CreateSiteKey(ctx, "site-1", "ci-a", []string{"deploy"})
	if err != nil {
		t.Fatalf("CreateSiteKey: %v", err)
	}
	_, _, err = a.CreateSiteKey(ctx, "site-1", "ci-b", []string{"deploy"})
	if err != nil {
		t.Fatalf("CreateSiteKey: %v", err)
	}

	keys, err := a.ListSiteKeys(ctx, "site-1")
	if err != nil {
		t.Fatalf("ListSiteKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].KeyID != keyID1 && keys[1].KeyID != keyID1 {
		t.Fatalf("expected key_id %s in list, got %+v", keyID1, keys)
	}
}

func TestListSiteKeysMissingSite(t *testing.T) {
	a, _ := setupSiteKeys(t)
	_, err := a.ListSiteKeys(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestRevokeSiteKeyNotFound(t *testing.T) {
	a, _ := setupSiteKeys(t)
	err := a.RevokeSiteKey(context.Background(), "site-1", "99999")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}
