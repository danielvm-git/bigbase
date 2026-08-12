package sites_test

import (
	"context"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

type projectAttachmentLogger struct{}

func (projectAttachmentLogger) Info(string, ...any)  {}
func (projectAttachmentLogger) Warn(string, ...any)  {}
func (projectAttachmentLogger) Error(string, ...any) {}
func (projectAttachmentLogger) Debug(string, ...any) {}

func TestSiteCreateGetsProjectAndProductionEnvironment(t *testing.T) {
	logger := projectAttachmentLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	p := projects.New(projects.Options{DB: d, Logger: logger})
	s := sites.New(sites.Options{DB: d, Logger: logger, ProjectProvisioner: p})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(g)
	k.Register(p)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("start components: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	if _, err := d.Exec(`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		VALUES ('repo-1', 'payments', 0, 1, 'main', '', datetime('now'))`); err != nil {
		t.Fatalf("insert repository: %v", err)
	}
	ctx := auth.WithOrgID(context.Background(), 11)
	siteID, _, err := s.CreateSite(ctx, "repo-1", "payments", "main")
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	var projectID string
	if err := d.QueryRow(`SELECT project_id FROM sites WHERE id = ?`, siteID).Scan(&projectID); err != nil || projectID == "" {
		t.Fatalf("site project attachment=%q err=%v", projectID, err)
	}
	var orgID int64
	if err := d.QueryRow(`SELECT org_id FROM projects WHERE id = ?`, projectID).Scan(&orgID); err != nil || orgID != 11 {
		t.Fatalf("project org=%d err=%v", orgID, err)
	}
	var environments int
	if err := d.QueryRow(`SELECT COUNT(*) FROM project_environments WHERE project_id = ? AND slug = 'production'`, projectID).Scan(&environments); err != nil || environments != 1 {
		t.Fatalf("production environments=%d err=%v", environments, err)
	}
}
