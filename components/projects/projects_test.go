package projects_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) Debug(string, ...any) {}

func projectFixture(t *testing.T) (*db.DB, *projects.Projects) {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	p := projects.New(projects.Options{DB: d, Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(p)
	if err := k.Start(); err != nil {
		t.Fatalf("start components: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return d, p
}

func orgRequest(ctx context.Context, method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	return r.WithContext(auth.WithOrgID(ctx, 1))
}

func TestProjectAndEnvironmentOrgIsolation(t *testing.T) {
	_, p := projectFixture(t)
	create := orgRequest(context.Background(), http.MethodPost, "/api/projects", `{"name":"Payments"}`)
	w := httptest.NewRecorder()
	p.Handler().ServeHTTP(w, create)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", w.Code, w.Body.String())
	}
	var envelope struct {
		Data projects.Project `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	if envelope.Data.OrgID != 1 || envelope.Data.Name != "Payments" {
		t.Fatalf("unexpected project: %+v", envelope.Data)
	}

	other := httptest.NewRequest(http.MethodGet, "/api/projects/"+envelope.Data.ID, nil)
	other = other.WithContext(auth.WithOrgID(context.Background(), 2))
	w = httptest.NewRecorder()
	p.Handler().ServeHTTP(w, other)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-org project read status=%d body=%s", w.Code, w.Body.String())
	}

	env := orgRequest(context.Background(), http.MethodPost, "/api/projects/"+envelope.Data.ID+"/environments", `{"slug":"staging","name":"Staging"}`)
	w = httptest.NewRecorder()
	p.Handler().ServeHTTP(w, env)
	if w.Code != http.StatusCreated {
		t.Fatalf("create environment status=%d body=%s", w.Code, w.Body.String())
	}
	otherEnv := httptest.NewRequest(http.MethodGet, "/api/projects/"+envelope.Data.ID+"/environments", nil)
	otherEnv = otherEnv.WithContext(auth.WithOrgID(context.Background(), 2))
	w = httptest.NewRecorder()
	p.Handler().ServeHTTP(w, otherEnv)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-org environment read status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestProjectMigrationIsIdempotentAndPreservesSiteValues(t *testing.T) {
	d, p := projectFixture(t)
	if _, err := d.Exec(`CREATE TABLE sites (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, org_id INTEGER NOT NULL,
		project_id TEXT REFERENCES projects(id))`); err != nil {
		t.Fatalf("create sites: %v", err)
	}
	if _, err := d.Exec(`CREATE TABLE site_env_vars (site_id TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create site env vars: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO sites (id, name, org_id) VALUES ('site-1', 'legacy', 7)`); err != nil {
		t.Fatalf("insert site: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO site_env_vars (site_id, key, value) VALUES ('site-1', 'TOKEN', 'unchanged')`); err != nil {
		t.Fatalf("insert site value: %v", err)
	}
	if err := p.MigrateSiteAttachments(context.Background()); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := p.MigrateSiteAttachments(context.Background()); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	var projectID string
	if err := d.QueryRow(`SELECT project_id FROM sites WHERE id = 'site-1'`).Scan(&projectID); err != nil {
		t.Fatalf("read attachment: %v", err)
	}
	var projectCount, environmentCount int
	if err := d.QueryRow(`SELECT COUNT(*) FROM projects WHERE org_id = 7`).Scan(&projectCount); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if err := d.QueryRow(`SELECT COUNT(*) FROM project_environments WHERE project_id = ? AND slug = 'production'`, projectID).Scan(&environmentCount); err != nil {
		t.Fatalf("count environments: %v", err)
	}
	if projectCount != 1 || environmentCount != 1 {
		t.Fatalf("migration counts projects=%d environments=%d", projectCount, environmentCount)
	}
	var value string
	if err := d.QueryRow(`SELECT value FROM site_env_vars WHERE site_id = 'site-1' AND key = 'TOKEN'`).Scan(&value); err != nil || value != "unchanged" {
		t.Fatalf("site value changed: %q, %v", value, err)
	}
}

func TestProjectDeletionBlockedByAttachedSite(t *testing.T) {
	d, p := projectFixture(t)
	if _, err := d.Exec(`CREATE TABLE sites (id TEXT PRIMARY KEY, name TEXT NOT NULL, org_id INTEGER NOT NULL, project_id TEXT REFERENCES projects(id))`); err != nil {
		t.Fatalf("create sites: %v", err)
	}
	ctx := auth.WithOrgID(context.Background(), 1)
	project, err := p.CreateProject(ctx, "Attached")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO sites (id, name, org_id, project_id) VALUES ('site-1', 'attached', 1, ?)`, project.ID); err != nil {
		t.Fatalf("attach site: %v", err)
	}
	if err := p.DeleteProject(ctx, project.ID); !errors.Is(err, projects.ErrProjectHasSites) {
		t.Fatalf("delete attached project error=%v", err)
	}
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM sites WHERE project_id = ?`, project.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("attachment was not preserved: count=%d err=%v", count, err)
	}
}
