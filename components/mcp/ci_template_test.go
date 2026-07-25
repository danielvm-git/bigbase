package mcp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockSiteListerForCI struct{}

func (m *mockSiteListerForCI) ListSites(_ context.Context) ([]mcp.SiteInfo, error) {
	return nil, nil
}

func (m *mockSiteListerForCI) GetSite(_ context.Context, siteID string) (*mcp.SiteInfo, error) {
	if siteID == "site-1" {
		return &mcp.SiteInfo{
			ID:   "site-1",
			Name: "my-site",
			DeployDefaults: &mcp.SiteDeployDefaults{
				AppType:      "go",
				BuildCommand: "make build",
				StartCommand: "./bin/app",
				Env: map[string]string{
					"FOO": "bar",
				},
			},
		}, nil
	}
	return nil, nil
}

func TestCITemplateWithDefaults(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true, SiteLister: &mockSiteListerForCI{}})
	ctx, session := connectMCPSession(t, c)

	t.Run("defaults via site info", func(t *testing.T) {
		args := map[string]any{"site_id": "site-1"}
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_ci_template", Arguments: args})
		if err != nil {
			t.Fatalf("CallTool(get_ci_template): %v", err)
		}
		
		content := result.Content[0].(*mcpsdk.TextContent).Text
		if !strings.Contains(content, "build_command: \"make build\"") {
			t.Errorf("expected build_command: \"make build\" in content, got: %s", content)
		}
		if !strings.Contains(content, "FOO: \"bar\"") {
			t.Errorf("expected FOO env var in content")
		}
	})

	t.Run("defaults via manifest str", func(t *testing.T) {
		manifestStr := `
framework = "next"
[build]
command = "npm run build"
[start]
command = "npm run start"
`
		args := map[string]any{"manifest": manifestStr}
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_ci_template", Arguments: args})
		if err != nil {
			t.Fatalf("CallTool(get_ci_template): %v", err)
		}
		
		content := result.Content[0].(*mcpsdk.TextContent).Text
		if !strings.Contains(content, "build_command: \"npm run build\"") {
			t.Errorf("expected build_command: \"npm run build\" in content, got: %s", content)
		}
	})
}

