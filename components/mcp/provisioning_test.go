package mcp_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/danielvm/bigbase/components/mcp"
)

type mockGitCreator struct {
	id   string
	name string
	err  error
}

func (m *mockGitCreator) CreateRepo(_ context.Context, name, _ string, _ bool) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	if m.id == "" {
		return "repo-id-1", name, nil
	}
	return m.id, m.name, nil
}

type mockSiteCreator struct {
	id   string
	name string
	err  error
}

func (m *mockSiteCreator) CreateSite(_ context.Context, _, name, _ string) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	if m.id == "" {
		return "site-id-1", name, nil
	}
	return m.id, m.name, nil
}

type mockSiteKeyCreator struct {
	token string
	keyID string
	err   error
	keys  []mcp.SiteKeyEntry
}

func (m *mockSiteKeyCreator) CreateSiteKey(_ context.Context, _, _ string, _ []string) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	if m.token == "" {
		return "bb_dep_" + strings.Repeat("a", 64), "42", nil
	}
	return m.token, m.keyID, nil
}

func (m *mockSiteKeyCreator) ListSiteKeys(_ context.Context, siteID string) ([]mcp.SiteKeyEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.keys != nil {
		return m.keys, nil
	}
	return []mcp.SiteKeyEntry{
		{KeyID: "42", Name: "ci-bot", CreatedAt: "2026-01-01T00:00:00Z", Revoked: false},
	}, nil
}

func (m *mockSiteKeyCreator) RevokeSiteKey(_ context.Context, siteID, keyID string) error {
	if m.err != nil {
		return m.err
	}
	if keyID == "" {
		return errors.New("key_id is required")
	}
	if siteID == "" {
		return errors.New("site_id is required")
	}
	return nil
}

type mockSiteLister struct {
	sites []mcp.SiteInfo
	err   error
}

func (m *mockSiteLister) ListSites(_ context.Context) ([]mcp.SiteInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.sites != nil {
		return m.sites, nil
	}
	return []mcp.SiteInfo{
		{
			ID:               "site-1",
			Name:             "my-app",
			GitRepoID:        "repo-1",
			ProductionBranch: "main",
			LatestDeployment: &mcp.SiteDeployment{URL: "https://my-app.bigbase.click", Status: "running"},
		},
	}, nil
}

func (m *mockSiteLister) GetSite(_ context.Context, siteID string) (*mcp.SiteInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.sites {
		if s.ID == siteID {
			return &s, nil
		}
	}
	if siteID == "site-1" {
		return &mcp.SiteInfo{
			ID:               "site-1",
			Name:             "my-app",
			GitRepoID:        "repo-1",
			ProductionBranch: "main",
			LatestDeployment: &mcp.SiteDeployment{URL: "https://my-app.bigbase.click", Status: "running", ID: "dep-1"},
		}, nil
	}
	return nil, errors.New(`site "` + siteID + `" not found`)
}

func callToolText(t *testing.T, c *mcp.Component, name string, args map[string]any) string {
	t.Helper()
	ctx, session := connectMCPSession(t, c)
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestCreateRepo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, GitCreator: &mockGitCreator{id: "abc", name: "my-project"}})
		text := callToolText(t, c, "create_repo", map[string]any{"name": "my-project"})
		if !strings.Contains(text, `"repo_id":"abc"`) || !strings.Contains(text, `"name":"my-project"`) {
			t.Fatalf("unexpected response: %s", text)
		}
	})
	t.Run("nil creator", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true})
		text := callToolText(t, c, "create_repo", map[string]any{"name": "x"})
		if !strings.Contains(text, "Git tools require a Git component") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("empty name", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, GitCreator: &mockGitCreator{}})
		text := callToolText(t, c, "create_repo", map[string]any{})
		if !strings.Contains(text, "name is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, GitCreator: &mockGitCreator{err: errors.New("repo name already exists")}})
		text := callToolText(t, c, "create_repo", map[string]any{"name": "dup"})
		if !strings.Contains(text, "already exists") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}

func TestCreateSite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteCreator: &mockSiteCreator{id: "site-1", name: "my-app"}})
		text := callToolText(t, c, "create_site", map[string]any{"git_repo_id": "repo-1", "name": "my-app"})
		if !strings.Contains(text, `"site_id":"site-1"`) {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("nil creator", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true})
		text := callToolText(t, c, "create_site", map[string]any{"git_repo_id": "r1"})
		if !strings.Contains(text, "Site tools require a Sites component") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("missing git_repo_id", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteCreator: &mockSiteCreator{}})
		text := callToolText(t, c, "create_site", map[string]any{})
		if !strings.Contains(text, "git_repo_id is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("auto_deploy", func(t *testing.T) {
		c := mcp.New(mcp.Options{
			Enabled:     true,
			SiteCreator: &mockSiteCreator{id: "site-1", name: "app"},
			Deployer:    &mockDeployer{},
		})
		text := callToolText(t, c, "create_site", map[string]any{
			"git_repo_id": "repo-1",
			"name":        "app",
			"auto_deploy": true,
		})
		if !strings.Contains(text, `"url"`) {
			t.Fatalf("expected deploy url in response: %s", text)
		}
	})
}

func TestProvisionCICredentials(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "provision_ci_credentials", map[string]any{"site_id": "site-1"})
		if !strings.Contains(text, "bb_dep_") || !strings.Contains(text, `"key_id"`) {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("nil creator", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true})
		text := callToolText(t, c, "provision_ci_credentials", map[string]any{"site_id": "s1"})
		if !strings.Contains(text, "Credential provisioning requires an Auth component") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("missing site_id", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "provision_ci_credentials", map[string]any{})
		if !strings.Contains(text, "site_id is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}

func TestGetCITemplate(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})

	t.Run("catalog", func(t *testing.T) {
		text := callToolText(t, c, "get_ci_template", nil)
		if !strings.Contains(text, "node") || !strings.Contains(text, "static") {
			t.Fatalf("expected catalog types: %s", text)
		}
	})
	t.Run("static yaml", func(t *testing.T) {
		text := callToolText(t, c, "get_ci_template", map[string]any{"app_type": "static"})
		if !strings.Contains(text, "BIGBASE_SITE_ID") || !strings.Contains(text, "/api/deploy") {
			t.Fatalf("expected deploy yaml: %s", text)
		}
	})
	t.Run("unknown type", func(t *testing.T) {
		text := callToolText(t, c, "get_ci_template", map[string]any{"app_type": "unknown"})
		if !strings.Contains(text, "No template for") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}

func TestListSites(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteLister: &mockSiteLister{}})
		text := callToolText(t, c, "list_sites", nil)
		if !strings.Contains(text, `"site_id":"site-1"`) || !strings.Contains(text, `"git_repo_id":"repo-1"`) {
			t.Fatalf("unexpected: %s", text)
		}
		if !strings.Contains(text, "my-app.bigbase.click") {
			t.Fatalf("expected deploy url: %s", text)
		}
	})
	t.Run("nil lister", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true})
		text := callToolText(t, c, "list_sites", nil)
		if !strings.Contains(text, "Site discovery requires a Sites component") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}

func TestGetSite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteLister: &mockSiteLister{}})
		text := callToolText(t, c, "get_site", map[string]any{"site_id": "site-1"})
		if !strings.Contains(text, `"production_branch":"main"`) || !strings.Contains(text, `"last_deploy_status":"running"`) {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("missing site_id", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteLister: &mockSiteLister{}})
		text := callToolText(t, c, "get_site", nil)
		if !strings.Contains(text, "site_id is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("not found", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteLister: &mockSiteLister{}})
		text := callToolText(t, c, "get_site", map[string]any{"site_id": "missing"})
		if !strings.Contains(text, "not found") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}

func TestListSiteKeys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "list_site_keys", map[string]any{"site_id": "site-1"})
		if !strings.Contains(text, `"key_id":"42"`) || !strings.Contains(text, `"revoked":false`) {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("missing site_id", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "list_site_keys", nil)
		if !strings.Contains(text, "site_id is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}

func TestRevokeSiteKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "revoke_site_key", map[string]any{"key_id": "42", "site_id": "site-1"})
		if !strings.Contains(text, `"revoked":"true"`) {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("missing key_id", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "revoke_site_key", map[string]any{"site_id": "site-1"})
		if !strings.Contains(text, "key_id is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
	t.Run("missing site_id", func(t *testing.T) {
		c := mcp.New(mcp.Options{Enabled: true, SiteKeyCreator: &mockSiteKeyCreator{}})
		text := callToolText(t, c, "revoke_site_key", map[string]any{"key_id": "42"})
		if !strings.Contains(text, "site_id is required") {
			t.Fatalf("unexpected: %s", text)
		}
	})
}
