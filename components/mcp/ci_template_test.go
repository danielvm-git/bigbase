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

func TestCITemplateRust(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_ci_template",
		Arguments: map[string]any{"app_type": "rust"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_ci_template, rust): %v", err)
	}
	content := result.Content[0].(*mcpsdk.TextContent).Text
	if !strings.Contains(content, "rust") {
		t.Errorf("expected rust in template content, got: %s", content)
	}
	if strings.Contains(content, "No template for") {
		t.Fatalf("get_ci_template returned no template for rust: %s", content)
	}
}

func TestCITemplatePHP(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_ci_template",
		Arguments: map[string]any{"app_type": "php"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_ci_template, php): %v", err)
	}
	content := result.Content[0].(*mcpsdk.TextContent).Text
	if !strings.Contains(content, "php") {
		t.Errorf("expected php in template content, got: %s", content)
	}
	if strings.Contains(content, "No template for") {
		t.Fatalf("get_ci_template returned no template for php: %s", content)
	}
}

func TestCITemplateJava(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_ci_template",
		Arguments: map[string]any{"app_type": "java"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_ci_template, java): %v", err)
	}
	content := result.Content[0].(*mcpsdk.TextContent).Text
	if !strings.Contains(content, "java") {
		t.Errorf("expected java in template content, got: %s", content)
	}
	if strings.Contains(content, "No template for") {
		t.Fatalf("get_ci_template returned no template for java: %s", content)
	}
}

func TestCITemplateRuby(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_ci_template",
		Arguments: map[string]any{"app_type": "ruby"},
	})
	if err != nil {
		t.Fatalf("CallTool(get_ci_template, ruby): %v", err)
	}
	content := result.Content[0].(*mcpsdk.TextContent).Text
	if !strings.Contains(content, "ruby") {
		t.Errorf("expected ruby in template content, got: %s", content)
	}
	if strings.Contains(content, "No template for") {
		t.Fatalf("get_ci_template returned no template for ruby: %s", content)
	}
}

func TestCITemplatesAll(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)

	types := []string{"static", "node", "go", "python", "php", "rust", "java", "ruby"}
	for _, appType := range types {
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
			Name:      "get_ci_template",
			Arguments: map[string]any{"app_type": appType},
		})
		if err != nil {
			t.Errorf("CallTool(get_ci_template, %s) error: %v", appType, err)
			continue
		}
		content := result.Content[0].(*mcpsdk.TextContent).Text
		if strings.Contains(content, "No template for") {
			t.Errorf("missing template for %s: %s", appType, content)
		}
	}
}
