// Package mcp implements an MCP (Model Context Protocol) server for BigBase.
// It exposes tools that teach AI coding agents how to use BigBase services,
// deploy applications, and integrate with frameworks.
package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.1.0"

// DeployTrigger triggers a deployment from a git repo.
type DeployTrigger interface {
	Trigger(ctx context.Context, repoID, branch, siteName, siteID string) (*deploy.Deployment, error)
}

// Options configure the MCP component.
type Options struct {
	// Logger is the structured logger. If nil, a no-op logger is used.
	Logger Logger
	// Port is the HTTP port for the MCP server (default: 3900).
	Port int
	// Transport is "stdio" or "http" (default: "http").
	Transport string
	// Enabled controls whether the MCP server starts (default: true).
	Enabled bool
	// DB is the database for deploy tools (list_repos, get_deploy_status, etc.).
	DB DBer
	// Deployer triggers deployments. Set to the deploy component for the
	// deploy_site tool to work as a real trigger instead of a doc-only tool.
	Deployer DeployTrigger
}

// DBer is the database interface for deploy tools.
type DBer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Component is the ECC component for the MCP server.
type Component struct {
	logger    Logger
	port      int
	transport string
	enabled   bool
	db        DBer
	deployer  DeployTrigger
}

// New creates a new MCP component with the given options.
func New(opts Options) *Component {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	port := opts.Port
	if port == 0 {
		port = 3900
	}
	transport := opts.Transport
	if transport == "" {
		transport = "http"
	}
	return &Component{
		logger:    logger,
		port:      port,
		transport: transport,
		enabled:   opts.Enabled,
		db:        opts.DB,
		deployer:  opts.Deployer,
	}
}

func (c *Component) Name() string    { return "mcp" }
func (c *Component) Version() string { return version }
func (c *Component) Dependencies() []string {
	if c.db != nil {
		return []string{"db"}
	}
	return nil
}
func (c *Component) ConfigSchema() json.RawMessage { return nil }
func (c *Component) Hooks() []kernel.HookDef       { return nil }

func (c *Component) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (c *Component) Start(ctx *kernel.Context) error {
	if !c.enabled {
		c.logger.Info("mcp server disabled")
		return nil
	}
	c.logger.Info("mcp server starting", "transport", c.transport, "port", c.port)

	switch c.transport {
	case "stdio":
		go func() {
			if err := c.ServeStdio(context.Background()); err != nil {
				c.logger.Error("mcp stdio server failed", "error", err)
			}
		}()
	case "http":
		go func() {
			addr := fmt.Sprintf(":%d", c.port)
			c.logger.Info("mcp http server listening", "addr", addr)
			if err := http.ListenAndServe(addr, c.Handler()); err != nil {
				c.logger.Error("mcp http server failed", "error", err)
			}
		}()
	}
	return nil
}

func (c *Component) Stop(ctx *kernel.Context) error {
	c.logger.Info("mcp server stopped")
	return nil
}

// NewMCPServer creates and configures an MCP server with registered tools.
func (c *Component) NewMCPServer() (*mcpsdk.Server, error) {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "BigBase",
		Version: version,
	}, &mcpsdk.ServerOptions{
		Instructions: `BigBase is an open-source Backend-as-a-Service platform — like Supabase or Firebase, but self-hosted.

Available services: deploy, auth, db (auto-API), storage, functions, realtime, messaging, webhooks, forge, cici, monitoring.

Workflow: pick a framework → get code examples → deploy to bigbase.click.

Use list_services to see the full catalog, get_service_docs for details on a specific service, get_code_example for framework-specific snippets, and list_frameworks to see supported frameworks.`,
	})

	// registerPingTool
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "ping",
		Description: "Simple ping to verify the MCP server is alive.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "pong"}},
		}, nil, nil
	})

	// registerListServices
	services, err := loadServices()
	if err != nil {
		return nil, fmt.Errorf("load services: %w", err)
	}
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_services",
		Description: "List all BigBase services with capabilities and status. Use this to discover what the platform offers before diving into a specific service.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: formatServicesList(services)}},
		}, nil, nil
	})

	// registerGetServiceDocs
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_service_docs",
		Description: "Get detailed documentation for a specific BigBase service, including API endpoints and capabilities. Use after list_services to dive deeper into a service.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		var args map[string]any
		if req.Params.Arguments != nil {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}
		name, _ := args["service"].(string)
		for _, s := range services {
			if s.Name == name {
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: formatServiceDoc(s)}},
				}, nil, nil
			}
		}
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Service %q not found. Use list_services to see available services.", name)}},
		}, nil, nil
	})

	// registerGetCodeExample
	examples, err := loadCodeExamples()
	if err != nil {
		return nil, fmt.Errorf("load examples: %w", err)
	}
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_code_example",
		Description: "Get ready-to-paste code snippets for a BigBase service in a specific framework (sveltekit, react, nextjs, vue, etc.). Use this to quickly integrate auth, db, storage, or realtime.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		var args map[string]any
		if req.Params.Arguments != nil {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}
		svc, _ := args["service"].(string)
		fw, _ := args["framework"].(string)
		for _, ex := range examples {
			if ex.Service == svc && ex.Framework == fw {
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: formatCodeExample(ex)}},
				}, nil, nil
			}
		}
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("No example for %s/%s. Try list_frameworks and list_services to see what's available.", svc, fw)}},
		}, nil, nil
	})

	// registerListFrameworks
	frameworks, err := loadFrameworks()
	if err != nil {
		return nil, fmt.Errorf("load frameworks: %w", err)
	}
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_frameworks",
		Description: "List frameworks BigBase supports with maturity level (full, partial, planned). Use this to choose the right framework for your project.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: formatFrameworksList(frameworks)}},
		}, nil, nil
	})

	// --- Deploy workflow tools ---

	// registerListRepos
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_repos",
		Description: "List git repositories available for deployment on this BigBase instance.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.db == nil {
			return textResult("Deploy tools require a database connection. Start BigBase with `serve` to enable deploy workflows."), nil, nil
		}
		rows, err := c.db.QueryContext(ctx, "SELECT id, name, description, updated_at FROM git_repos ORDER BY updated_at DESC LIMIT 50")
		if err != nil {
			return textResult(fmt.Sprintf("Error listing repos: %v", err)), nil, nil
		}
		defer func() { _ = rows.Close() }()
		var b strings.Builder
		b.WriteString("# Git Repositories\n\n")
		count := 0
		for rows.Next() {
			var id, name, desc, updated string
			if err := rows.Scan(&id, &name, &desc, &updated); err != nil {
				continue
			}
			fmt.Fprintf(&b, "- **%s** (`%s`) — %s\n", name, id, desc)
			count++
		}
		if count == 0 {
			b.WriteString("No repositories found. Push a repo or create one via the Git component.\n")
		}
		b.WriteString("\n→ To deploy: use deploy_site with the repo_id\n")
		return textResult(b.String()), nil, nil
	})

	// registerDeploySite
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "deploy_site",
		Description: "Trigger a deployment for a git repository. Provide repo_id and optionally branch (default: main) and site_name.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.db == nil {
			return textResult("Deploy tools require a database connection. Start BigBase with `serve` to enable deploy workflows."), nil, nil
		}
		var args map[string]any
		if req.Params.Arguments != nil {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}
		repoID, _ := args["repo_id"].(string)
		if repoID == "" {
			return textResult("repo_id is required. Use list_repos to find available repositories."), nil, nil
		}
		branch, _ := args["branch"].(string)
		if branch == "" {
			branch = "main"
		}
		siteName, _ := args["site_name"].(string)
		siteID, _ := args["site_id"].(string)

		var repoName string
		err := c.db.QueryRowContext(ctx, "SELECT name FROM git_repos WHERE id = ?", repoID).Scan(&repoName)
		if err != nil {
			return textResult(fmt.Sprintf("Repository %q not found. Use list_repos to see available repositories.", repoID)), nil, nil
		}
		if siteName == "" {
			siteName = repoName
		}

		// If a Deployer is wired in, trigger the actual deployment.
		if c.deployer != nil {
			dep, err := c.deployer.Trigger(ctx, repoID, branch, siteName, siteID)
			if err != nil {
				return textResult(fmt.Sprintf("Deploy failed: %v", err)), nil, nil
			}
			shortID := dep.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			var b strings.Builder
			fmt.Fprintf(&b, "# Deploy Started: %s\n\n", shortID)
			fmt.Fprintf(&b, "- **Repo:** %s\n", repoName)
			fmt.Fprintf(&b, "- **Branch:** %s\n", branch)
			fmt.Fprintf(&b, "- **Site:** %s\n", dep.URL)
			fmt.Fprintf(&b, "- **Status:** %s\n", dep.Status)
			fmt.Fprintf(&b, "\n→ Use `get_deploy_status` with `deployment_id: %s` to monitor.\n", dep.ID)
			return textResult(b.String()), nil, nil
		}

		return textResult(fmt.Sprintf("# Deploy Request Received\n\n- **Repo:** %s\n- **Branch:** %s\n- **Site:** %s\n\nTo trigger the deployment, send a POST to `/api/deploy` with:\n```json\n{\"repo_id\": \"%s\", \"branch\": \"%s\", \"site_name\": \"%s\"}\n```\n\nThen use `get_deploy_status` to monitor progress.",
			repoName, branch, siteName, repoID, branch, siteName)), nil, nil
	})

	// registerGetDeployStatus
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_deploy_status",
		Description: "Check the status of a deployment by its ID. Returns status, URL, and build info.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.db == nil {
			return textResult("Deploy tools require a database connection."), nil, nil
		}
		var args map[string]any
		if req.Params.Arguments != nil {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}
		deployID, _ := args["deployment_id"].(string)
		if deployID == "" {
			// List recent deployments
			rows, err := c.db.QueryContext(ctx, "SELECT id, status, url, app_type, created_at FROM deployments ORDER BY created_at DESC LIMIT 10")
			if err != nil {
				return textResult(fmt.Sprintf("Error: %v", err)), nil, nil
			}
			defer func() { _ = rows.Close() }()
			var b strings.Builder
			b.WriteString("# Recent Deployments\n\n")
			for rows.Next() {
				var id, status, url, appType, created string
				_ = rows.Scan(&id, &status, &url, &appType, &created)
				fmt.Fprintf(&b, "- `%s` — **%s** %s (%s)\n", id[:8], status, url, appType)
			}
			b.WriteString("\n→ Use `get_deploy_status` with a `deployment_id` for details.\n")
			return textResult(b.String()), nil, nil
		}
		var status, url, appType, commitSHA, errMsg string
		_ = c.db.QueryRowContext(ctx,
			"SELECT status, COALESCE(url,''), COALESCE(app_type,''), COALESCE(commit_sha,''), COALESCE(error_message,'') FROM deployments WHERE id = ?", deployID,
		).Scan(&status, &url, &appType, &commitSHA, &errMsg)

		var b strings.Builder
		fmt.Fprintf(&b, "# Deployment %s\n\n- **Status:** %s\n- **URL:** %s\n- **Type:** %s\n", deployID[:8], status, url, appType)
		if commitSHA != "" {
			fmt.Fprintf(&b, "- **Commit:** %s\n", commitSHA[:7])
		}
		if errMsg != "" {
			fmt.Fprintf(&b, "- **Error:** %s\n", errMsg)
		}
		switch status {
		case "failed":
			b.WriteString("\n→ Check build logs with `get_deploy_logs`.\n")
		case "running":
			fmt.Fprintf(&b, "\n→ Site is live at %s\n", url)
		}
		return textResult(b.String()), nil, nil
	})

	// registerGetDeployLogs
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_deploy_logs",
		Description: "Fetch build logs for a deployment. Use this to debug failed deployments.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.db == nil {
			return textResult("Deploy tools require a database connection."), nil, nil
		}
		var args map[string]any
		if req.Params.Arguments != nil {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}
		deployID, _ := args["deployment_id"].(string)
		if deployID == "" {
			return textResult("deployment_id is required. Use get_deploy_status to find deployment IDs."), nil, nil
		}
		var buildLog string
		err := c.db.QueryRowContext(ctx, "SELECT COALESCE(build_log,'') FROM deployments WHERE id = ?", deployID).Scan(&buildLog)
		if err != nil {
			return textResult(fmt.Sprintf("Deployment %q not found.", deployID)), nil, nil
		}
		if buildLog == "" {
			return textResult("No build logs yet. The deployment may still be in progress."), nil, nil
		}
		return textResult(fmt.Sprintf("# Build Logs for %s\n\n```\n%s\n```", deployID[:8], buildLog)), nil, nil
	})

	// registerDeployGuide
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "deploy_guide",
		Description: "Step-by-step guide for deploying an application to bigbase.click. Use this when you need to understand the full deploy workflow.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		guide := `# Deploying to BigBase

## 1. Push your code to a git repo on this BigBase instance
Create a repo via the Git component or admin UI.

## 2. List available repos
Use ` + "`list_repos`" + ` to find your repo and get its ID.

## 3. Deploy your site
Use ` + "`deploy_site`" + ` with your repo_id and branch.
BigBase auto-detects your app type (Node, Go, Python, Static) and builds it.

## 4. Monitor progress
Use ` + "`get_deploy_status`" + ` to check build status.

## 5. Debug failures
Use ` + "`get_deploy_logs`" + ` to view build output.

## Framework-specific notes
- **SvelteKit**: Use adapter-static for static sites, adapter-node for SSR.
- **Next.js**: Static export (` + "`output: 'export'`" + `) recommended.
- **Astro**: Works out of the box with static output.
- **React/Vue (Vite)**: ` + "`npm run build`" + ` produces ` + "`dist/`" + ` served automatically.
- **Go**: ` + "`go build`" + ` produces a binary that runs on the assigned port.

## Environment
Port is set via ` + "`PORT`" + ` env var. Database is SQLite by default.
`
		return textResult(guide), nil, nil
	})

	return srv, nil
}

func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

// ServeStdio runs the MCP server over stdio (stdin/stdout).
// This blocks until the context is cancelled or stdin closes.
func (c *Component) ServeStdio(ctx context.Context) error {
	srv, err := c.NewMCPServer()
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	return srv.Run(ctx, &mcpsdk.IOTransport{
		Reader: os.Stdin,
		Writer: os.Stdout,
	})
}

// Handler returns an HTTP handler for the MCP server.
// Routes: POST /mcp — MCP messages, GET /health — health check.
func (c *Component) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		srv, err := c.NewMCPServer()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		handler := mcpsdk.NewStreamableHTTPHandler(func(_ *http.Request) *mcpsdk.Server {
			return srv
		}, nil)
		handler.ServeHTTP(w, r)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ok","version":%q}`, version)
	})

	return mux
}

// Logger is the subset of slog used by this component.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}
