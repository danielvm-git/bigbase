package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScopeProvision is the scope required for write-tier MCP tools.
// Auth components reference this to issue correctly-scoped org keys.
const ScopeProvision = "mcp:provision"

// OrgKeyAuthenticator resolves bb_ organization API keys for MCP HTTP auth.
// Implemented by auth at the composition root (main.go); MCP does not import auth.
type OrgKeyAuthenticator interface {
	ResolveOrgKey(rawKey string) (orgID int64, scopes []string, err error)
}

type toolTier int

const (
	tierPublic toolTier = iota
	tierRead
	tierWrite
)

var publicTools = map[string]struct{}{
	"ping":             {},
	"list_services":    {},
	"get_service_docs": {},
	"get_code_example": {},
	"list_frameworks":  {},
	"get_ci_template":  {},
	"deploy_guide":     {},
}

var writeTools = map[string]struct{}{
	"create_repo":              {},
	"create_site":              {},
	"deploy_site":              {},
	"provision_ci_credentials": {},
	"list_site_keys":           {},
	"revoke_site_key":          {},
}

func toolTierFor(name string) toolTier {
	if _, ok := publicTools[name]; ok {
		return tierPublic
	}
	if _, ok := writeTools[name]; ok {
		return tierWrite
	}
	return tierRead
}

type orgAuthKeyType struct{}

var orgAuthKey orgAuthKeyType

type orgAuth struct {
	orgID  int64
	scopes []string
}

// WithOrgAuth attaches resolved org credentials to context (used after Bearer validation).
func WithOrgAuth(ctx context.Context, orgID int64, scopes []string) context.Context {
	return context.WithValue(ctx, orgAuthKey, orgAuth{orgID: orgID, scopes: scopes})
}

// OrgAuthFromContext returns org_id and scopes set by Bearer validation.
func OrgAuthFromContext(ctx context.Context) (orgID int64, scopes []string, ok bool) {
	v, ok := ctx.Value(orgAuthKey).(orgAuth)
	if !ok {
		return 0, nil, false
	}
	return v.orgID, v.scopes, true
}

func bearerToken(h http.Header) string {
	if h == nil {
		return ""
	}
	auth := h.Get("Authorization")
	if len(auth) < 7 {
		return ""
	}
	// Accept both "Bearer" (RFC 6750) and "bearer" for non-conforming clients.
	if !strings.EqualFold(auth[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(auth[7:])
}

func hasScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

func authToolError(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
		IsError: true,
	}
}

func (c *Component) enforceToolAuth(ctx context.Context, req *mcpsdk.CallToolRequest, tier toolTier) (context.Context, *mcpsdk.CallToolResult) {
	if c.orgKeyAuth == nil || tier == tierPublic {
		return ctx, nil
	}

	_, scopes, ok := OrgAuthFromContext(ctx)
	// In the normal HTTP flow the Bearer middleware pre-sets context.
	// This fallback handles edge cases where context is stripped or a
	// caller constructs HTTP-like requests without going through the
	// middleware (e.g. test harnesses that simulate HTTP transports).
	if !ok && req != nil && req.Extra != nil {
		token := bearerToken(req.Extra.Header)
		if token != "" {
			var err error
			var orgID int64
			orgID, scopes, err = c.orgKeyAuth.ResolveOrgKey(token)
			if err != nil {
				return ctx, authToolError("authorization required")
			}
			ctx = WithOrgAuth(ctx, orgID, scopes)
			ok = true
		}
	}
	if !ok {
		return ctx, authToolError("authorization required")
	}
	if tier == tierWrite && !hasScope(scopes, ScopeProvision) {
		return ctx, authToolError("insufficient scope")
	}
	return ctx, nil
}

type toolHandler func(context.Context, *mcpsdk.CallToolRequest, any) (*mcpsdk.CallToolResult, any, error)

func (c *Component) registerTool(srv *mcpsdk.Server, tier toolTier, tool *mcpsdk.Tool, handler toolHandler) {
	mcpsdk.AddTool(srv, tool, func(ctx context.Context, req *mcpsdk.CallToolRequest, input any) (*mcpsdk.CallToolResult, any, error) {
		ctx, authErr := c.enforceToolAuth(ctx, req, tier)
		if authErr != nil {
			return authErr, nil, nil
		}
		return handler(ctx, req, input)
	})
}

type toolCallParams struct {
	Name string `json:"name"`
}

func parseToolCall(body []byte) (string, bool) {
	var req struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Method != "tools/call" {
		return "", false
	}
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
		return "", false
	}
	return params.Name, true
}

// authenticatePost validates the Bearer token for a non-public tool call.
// Writes HTTP error and returns nil context on failure.
func (c *Component) authenticatePost(w http.ResponseWriter, r *http.Request, toolName string) context.Context {
	tier := toolTierFor(toolName)
	if tier == tierPublic {
		return r.Context()
	}

	token := bearerToken(r.Header)
	if token == "" {
		c.logger.Warn("mcp auth failed: missing bearer token", "tool", toolName)
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return nil
	}

	orgID, scopes, err := c.orgKeyAuth.ResolveOrgKey(token)
	if err != nil {
		c.logger.Warn("mcp auth failed: invalid key", "tool", toolName, "err", err)
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return nil
	}
	if tier == tierWrite && !hasScope(scopes, ScopeProvision) {
		c.logger.Warn("mcp auth failed: insufficient scope", "tool", toolName, "scopes", fmt.Sprint(scopes))
		http.Error(w, "insufficient scope", http.StatusForbidden)
		return nil
	}

	return WithOrgAuth(r.Context(), orgID, scopes)
}

func (c *Component) bearerAuthMiddleware(next http.Handler) http.Handler {
	if c.orgKeyAuth == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" || r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		toolName, isToolCall := parseToolCall(body)
		if !isToolCall {
			next.ServeHTTP(w, r)
			return
		}

		ctx := c.authenticatePost(w, r, toolName)
		if ctx == nil {
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func truncPrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
