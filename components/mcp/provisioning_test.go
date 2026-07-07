package mcp_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
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
			"git_repo_id":  "repo-1",
			"name":         "app",
			"auto_deploy":  true,
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
