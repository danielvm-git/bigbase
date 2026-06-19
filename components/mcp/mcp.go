// Package mcp implements an MCP (Model Context Protocol) server for BigBase.
// It exposes tools that teach AI coding agents how to use BigBase services,
// deploy applications, and integrate with frameworks.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/danielvm/bigbase/kernel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.1.0"

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
}

// Component is the ECC component for the MCP server.
type Component struct {
	logger    Logger
	port      int
	transport string
	enabled   bool
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
	}
}

func (c *Component) Name() string                   { return "mcp" }
func (c *Component) Version() string                { return version }
func (c *Component) Dependencies() []string         { return nil }
func (c *Component) ConfigSchema() json.RawMessage  { return nil }
func (c *Component) Hooks() []kernel.HookDef        { return nil }

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
		Instructions: "BigBase is an open-source Backend-as-a-Service platform. Use the tools to learn about services, get code examples, and deploy applications.",
	})

	// Register ping tool
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "ping",
		Description: "Simple ping to verify the MCP server is alive.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: "pong"},
			},
		}, nil, nil
	})

	return srv, nil
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
		fmt.Fprintf(w, `{"status":"ok","version":%q}`, version)
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
