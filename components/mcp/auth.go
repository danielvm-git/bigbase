// story: e71s01
// story: e71s02
// story: e71s03
// story: e71s04
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScopeProvision is the scope required for write-tier MCP tools.
// Auth components reference this to issue correctly-scoped org keys.
const ScopeProvision = "mcp:provision"

// ScopeSecretsRead is the narrow scope required for secret metadata and
// explicit value-read tools (list/get/read value). It never grants mutation.
const ScopeSecretsRead = "mcp:secrets:read"

// ScopeSecretsWrite is the narrow scope required for secret mutation tools
// (set/delete). It is enforced independently per tool on both transports.
const ScopeSecretsWrite = "mcp:secrets:write"

// OrgKeyAuthenticator resolves bb_ organization API keys for MCP HTTP auth.
// Implemented by auth at the composition root (main.go); MCP does not import auth.
type OrgKeyAuthenticator interface {
	ResolveOrgKey(rawKey string) (orgID int64, scopes []string, err error)
}

// SiteKeyAuthenticator resolves bb_dep_ Site deploy keys to the Site they are
// bound to plus the key's own scopes. Implemented by auth at the composition
// root. The Site binding is derived from the credential, never from caller
// arguments, so a leaked Site key can never impersonate another Site.
type SiteKeyAuthenticator interface {
	ResolveSiteKeyScopes(rawKey string) (siteID string, scopes []string, err error)
}

// PrincipalKind distinguishes the two credential shapes MCP accepts:
// organization API keys (bb_) and Site deploy keys (bb_dep_).
type PrincipalKind int

const (
	// PrincipalOrg is an organization API key (bb_). It carries an org ID and
	// the key's scopes and may act on any target the org owns, subject to
	// per-tool scope gates.
	PrincipalOrg PrincipalKind = iota
	// PrincipalSite is a Site deploy key (bb_dep_). It is bound to exactly one
	// Site and may only act on that Site (or, with an explicit secret scope,
	// the Project of that Site).
	PrincipalSite
)

// Principal is the authenticated MCP caller identity. It is derived
// exclusively from the presented credential — never from tool arguments.
type Principal struct {
	Kind   PrincipalKind
	OrgID  int64  // valid when Kind == PrincipalOrg
	SiteID string // valid when Kind == PrincipalSite
	Scopes []string
}

type principalKeyType struct{}

var principalKey principalKeyType

// WithPrincipal attaches the authenticated principal to context. Org
// principals also receive the legacy org-auth value so existing callers of
// OrgAuthFromContext keep working.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	ctx = context.WithValue(ctx, principalKey, p)
	if p.Kind == PrincipalOrg {
		ctx = WithOrgAuth(ctx, p.OrgID, p.Scopes)
	}
	return ctx
}

// PrincipalFromContext returns the authenticated principal set by
// WithPrincipal or by the legacy WithOrgAuth path. Returns false when no
// principal is present (unauthenticated transport).
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if p, ok := ctx.Value(principalKey).(Principal); ok {
		return p, true
	}
	if orgID, scopes, ok := OrgAuthFromContext(ctx); ok {
		return Principal{Kind: PrincipalOrg, OrgID: orgID, Scopes: scopes}, true
	}
	return Principal{}, false
}

// resolvePrincipal resolves a raw bearer credential into a Principal using the
// authenticators wired at the composition root. bb_dep_ credentials resolve to
// Site principals; everything else is treated as an organization API key.
func (c *Component) resolvePrincipal(rawKey string) (Principal, error) {
	if strings.HasPrefix(rawKey, "bb_dep_") {
		if c.siteKeyAuth == nil {
			return Principal{}, errors.New("invalid api key")
		}
		siteID, scopes, err := c.siteKeyAuth.ResolveSiteKeyScopes(rawKey)
		if err != nil {
			return Principal{}, err
		}
		return Principal{Kind: PrincipalSite, SiteID: siteID, Scopes: scopes}, nil
	}
	if c.orgKeyAuth == nil {
		return Principal{}, errors.New("authentication not configured")
	}
	orgID, scopes, err := c.orgKeyAuth.ResolveOrgKey(rawKey)
	if err != nil {
		return Principal{}, err
	}
	return Principal{Kind: PrincipalOrg, OrgID: orgID, Scopes: scopes}, nil
}

type toolTier int

const (
	tierPublic toolTier = iota
	tierRead
	tierWrite
)

var publicTools = map[string]struct{}{
	"ping":                    {},
	"list_services":           {},
	"get_service_docs":        {},
	"get_code_example":        {},
	"list_frameworks":         {},
	"get_ci_template":         {},
	"deploy_guide":            {},
	"validate_ci_credentials": {},
}

var writeTools = map[string]struct{}{
	"create_repo":              {},
	"create_site":              {},
	"deploy_site":              {},
	"provision_ci_credentials": {},
	"revoke_site_key":          {},
	"set_site_env_vars":        {},
	"delete_site_env_var":      {},
	"set_project_secret":       {},
	"delete_project_secret":    {},
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

// toolGate carries the per-tool authorization rules beyond the tier model.
//
//   - required: scopes an organization principal must hold (e.g.
//     mcp:secrets:read). Site principals are checked against siteRequired
//     instead — a Site key is only granted narrow scopes by explicit policy.
//   - siteAllowed: whether a Site deploy-key principal may call the tool at
//     all. Site principals are denied everything except Site-scoped tools so
//     a leaked bb_dep_ key cannot enumerate org resources.
//   - siteRequired: the scope a Site principal must hold to call the tool
//     (explicit policy). Empty means the Site binding alone authorizes.
type toolGate struct {
	required     []string
	siteAllowed  bool
	siteRequired string
}

func (c *Component) enforceToolAuth(ctx context.Context, req *mcpsdk.CallToolRequest, tier toolTier, gate toolGate) (context.Context, *mcpsdk.CallToolResult) {
	if (c.orgKeyAuth == nil && c.siteKeyAuth == nil) || tier == tierPublic {
		return ctx, nil
	}

	p, ok := PrincipalFromContext(ctx)
	// In the normal HTTP flow the Bearer middleware pre-sets context.
	// This fallback handles edge cases where context is stripped or a
	// caller constructs HTTP-like requests without going through the
	// middleware (e.g. test harnesses that simulate HTTP transports).
	if !ok && req != nil && req.Extra != nil {
		token := bearerToken(req.Extra.Header)
		if token != "" {
			resolved, err := c.resolvePrincipal(token)
			if err != nil {
				return ctx, authToolError("authorization required")
			}
			ctx = WithPrincipal(ctx, resolved)
			p = resolved
			ok = true
		}
	}
	if !ok {
		return ctx, authToolError("authorization required")
	}

	switch p.Kind {
	case PrincipalOrg:
		if tier == tierWrite && !hasScope(p.Scopes, ScopeProvision) {
			return ctx, authToolError("insufficient scope")
		}
		for _, s := range gate.required {
			if !hasScope(p.Scopes, s) {
				return ctx, authToolError("insufficient scope")
			}
		}
	case PrincipalSite:
		// Site deploy keys are Site-bound: they only reach tools explicitly
		// opened to Site principals, and only with an explicitly granted
		// narrow scope when one is required.
		if !gate.siteAllowed {
			return ctx, authToolError("insufficient scope")
		}
		if len(gate.required) > 0 && gate.siteRequired == "" {
			// The tool demands an org scope but declares no Site policy:
			// a Site principal gets no implicit grant.
			return ctx, authToolError("insufficient scope")
		}
		if gate.siteRequired != "" && !hasScope(p.Scopes, gate.siteRequired) {
			return ctx, authToolError("insufficient scope")
		}
	}
	return ctx, nil
}

func (c *Component) authorizeSiteTarget(ctx context.Context, siteID string) *mcpsdk.CallToolResult {
	if c.orgKeyAuth == nil && c.siteKeyAuth == nil {
		return nil
	}
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return authToolError("site authorization required")
	}
	switch p.Kind {
	case PrincipalSite:
		// A Site deploy key is bound to exactly one Site; the target must be
		// that Site. Caller-supplied target identity is never authorization.
		if p.SiteID != siteID {
			c.logger.Warn("mcp site authorization denied: site key bound to different site", "target_site_id", siteID, "bound_site_id", p.SiteID)
			return authToolError("site authorization denied")
		}
		return nil
	case PrincipalOrg:
		if c.siteTargetAuthorizer == nil {
			return authToolError("site authorization required")
		}
		if err := c.siteTargetAuthorizer.AuthorizeSiteTarget(ctx, siteID, p.OrgID); err != nil {
			c.logger.Warn("mcp site authorization denied", "site_id", siteID)
			return authToolError("site authorization denied")
		}
		return nil
	default:
		return authToolError("site authorization required")
	}
}

// authorizeProjectTarget binds a caller-supplied Project ID to the
// authenticated principal before any SecretManager call. Ownership is derived
// from the credential by the composition-root ProjectTargetAuthorizer; both
// cross-organization targets and missing Projects collapse to one
// non-disclosing denial.
func (c *Component) authorizeProjectTarget(ctx context.Context, projectID string) *mcpsdk.CallToolResult {
	if c.orgKeyAuth == nil && c.siteKeyAuth == nil {
		return nil
	}
	p, ok := PrincipalFromContext(ctx)
	if !ok || c.projectTargetAuthorizer == nil {
		return authToolError("project authorization required")
	}
	if err := c.projectTargetAuthorizer.AuthorizeProjectTarget(ctx, projectID, p); err != nil {
		c.logger.Warn("mcp project authorization denied", "project_id", projectID)
		return authToolError("project authorization denied")
	}
	return nil
}

type toolHandler func(context.Context, *mcpsdk.CallToolRequest, any) (*mcpsdk.CallToolResult, any, error)

func (c *Component) registerTool(srv *mcpsdk.Server, tier toolTier, tool *mcpsdk.Tool, handler toolHandler) {
	c.registerScopedTool(srv, tier, toolGate{}, tool, handler)
}

func (c *Component) registerScopedTool(srv *mcpsdk.Server, tier toolTier, gate toolGate, tool *mcpsdk.Tool, handler toolHandler) {
	mcpsdk.AddTool(srv, tool, func(ctx context.Context, req *mcpsdk.CallToolRequest, input any) (*mcpsdk.CallToolResult, any, error) {
		ctx, authErr := c.enforceToolAuth(ctx, req, tier, gate)
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

	principal, err := c.resolvePrincipal(token)
	if err != nil {
		c.logger.Warn("mcp auth failed: invalid key", "tool", toolName)
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return nil
	}
	// Organization keys need mcp:provision for write-tier tools. Site deploy
	// keys skip this gate: their blast radius is bounded by Site binding and
	// the per-tool gate instead.
	if tier == tierWrite && principal.Kind == PrincipalOrg && !hasScope(principal.Scopes, ScopeProvision) {
		c.logger.Warn("mcp auth failed: insufficient scope", "tool", toolName, "scopes", fmt.Sprint(principal.Scopes))
		http.Error(w, "insufficient scope", http.StatusForbidden)
		return nil
	}

	return WithPrincipal(r.Context(), principal)
}

func (c *Component) bearerAuthMiddleware(next http.Handler) http.Handler {
	if c.orgKeyAuth == nil && c.siteKeyAuth == nil {
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
