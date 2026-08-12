// story: e89s07
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// secrets.go implements the e89s07 native Project secret MCP tools. Every tool
// enforces a narrow secret scope (mcp:secrets:read or mcp:secrets:write) on
// both HTTP and stdio transports, binds the caller-supplied Project target to
// the authenticated principal before any SecretManager call, returns masked
// metadata by default, bounds its arguments, and maps internal failures to
// generic value-free errors.
//
// MCP deliberately does not import auth or secrets: the ProjectSecretManager
// and ProjectTargetAuthorizer seams are implemented at the composition root
// (adapters.go), mirroring the existing OrgKeyAuthenticator and
// SiteTargetAuthorizer injection pattern.

// secretDefaultFolderName is the folder MCP tools resolve when the caller does
// not pass folder_id. It matches the REST surface (e89s04) so both adapters
// expose the same default scope.
const secretDefaultFolderName = "default"

// Security bounds (e89s04 §13, mirrored here so MCP responses stay bounded
// before any SecretManager call). The frozen manager re-validates as defense
// in depth.
const (
	maxMCPSecretIDLen    = 200
	maxMCPSecretKeyLen   = 128
	maxMCPSecretValueLen = 64 * 1024
	maxMCPSecretList     = 1000
)

var validMCPSecretKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// SecretMetadata is the MCP-facing metadata-only projection of a project
// secret. It never carries plaintext or ciphertext; ValuePreview is a masked
// preview computed by the SecretManager.
type SecretMetadata struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	EnvironmentID  string `json:"environment_id"`
	FolderID       string `json:"folder_id"`
	Key            string `json:"key"`
	CurrentVersion int    `json:"current_version"`
	ValuePreview   string `json:"value_preview"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// SecretValue is the MCP-facing explicit value-read result. It is
// intentionally distinct from SecretMetadata so no caller can mistake a list
// or mutation response for a value read.
type SecretValue struct {
	SecretID  string `json:"secret_id"`
	Key       string `json:"key"`
	Version   int    `json:"version"`
	Value     string `json:"value"`
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
}

// ProjectSecretManager is the MCP-facing projection of the frozen
// SecretManager seam. It is implemented at the composition root by an adapter
// that bridges the authenticated MCP principal into the SecretManager's auth
// context and translates storage errors into the sentinels below. Responses
// are metadata-only except ReadSecretValue, the explicit audited value read.
type ProjectSecretManager interface {
	// EnsureFolder resolves the folder identified by name, creating it when
	// missing, and returns its ID. It verifies the project → environment chain
	// and organization ownership.
	EnsureFolder(ctx context.Context, p Principal, projectID, environmentID, name string) (string, error)
	// ListSecrets lists masked metadata in a folder.
	ListSecrets(ctx context.Context, p Principal, projectID, environmentID, folderID string) ([]SecretMetadata, error)
	// GetSecretMetadata returns metadata (masked preview only) for one secret.
	GetSecretMetadata(ctx context.Context, p Principal, projectID, environmentID, folderID, key string) (SecretMetadata, error)
	// ReadSecretValue returns the plaintext of the current version. It is the
	// only operation that returns plaintext and must be explicitly scoped.
	ReadSecretValue(ctx context.Context, p Principal, projectID, environmentID, folderID, key string) (SecretValue, error)
	// CreateSecret stores the first immutable version of a secret.
	CreateSecret(ctx context.Context, p Principal, projectID, environmentID, folderID, key, value string) (SecretMetadata, error)
	// UpdateSecret appends an immutable version and makes it current.
	UpdateSecret(ctx context.Context, p Principal, projectID, environmentID, folderID, key, value string) (SecretMetadata, error)
	// DeleteSecret removes a secret and all its versions.
	DeleteSecret(ctx context.Context, p Principal, projectID, environmentID, folderID, key string) error
}

// ProjectTargetAuthorizer binds a caller-supplied Project ID to the
// authenticated principal. Organization principals must own the Project; Site
// deploy keys are restricted to the Project of their bound Site. Implementations
// derive all ownership from the authenticated principal and return
// non-disclosing errors.
type ProjectTargetAuthorizer interface {
	AuthorizeProjectTarget(ctx context.Context, projectID string, p Principal) error
}

// Sentinel errors returned by ProjectSecretManager implementations and mapped
// by safeSecretError. The mapping contract is the shared semantic with the
// e89s04 REST adapter: cross-organization and missing targets collapse into
// one non-disclosing error, crypto and unexpected failures collapse into a
// generic internal error that never carries SQL, key material, or plaintext.
var (
	// ErrSecretNotFound means the secret key does not exist in the target scope.
	ErrSecretNotFound = errors.New("secret not found")
	// ErrSecretTarget means the project, environment, or folder is missing or
	// belongs to a different principal (non-disclosing).
	ErrSecretTarget = errors.New("secret target not found")
	// ErrSecretConflict means the secret already exists (create on existing).
	ErrSecretConflict = errors.New("secret already exists")
	// ErrSecretInvalidKey means the secret key failed the identifier contract.
	ErrSecretInvalidKey = errors.New("invalid secret key")
	// ErrSecretTooLarge means the value exceeds the bound.
	ErrSecretTooLarge = errors.New("secret value too large")
	// ErrSecretInternal is the generic mapping for crypto and unexpected
	// storage failures; the underlying cause stays server-side.
	ErrSecretInternal = errors.New("secret operation failed")
)

// safeSecretError maps a ProjectSecretManager failure to a value-free message.
// Anything not recognized — including raw SQL or decryption errors — becomes
// "internal error" so the client never sees SQL, key material, or plaintext
// (SC-e89s07-P0-05).
func safeSecretError(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrSecretNotFound):
		return "secret not found"
	case errors.Is(err, ErrSecretTarget):
		return "not found"
	case errors.Is(err, ErrSecretConflict):
		return "secret already exists"
	case errors.Is(err, ErrSecretInvalidKey):
		return "invalid secret key"
	case errors.Is(err, ErrSecretTooLarge):
		return "secret value too large"
	default:
		return "internal error"
	}
}

// secretArgs is the parsed and bounded argument set shared by the five secret
// tools. Caller-supplied values are resource locators only — authorization is
// always derived from the authenticated principal.
type secretArgs struct {
	projectID     string
	environmentID string
	folderID      string
	key           string
	value         string
}

// parseSecretArgs extracts and bounds the common arguments. key and value are
// only required for the tools that use them; empty key/value are accepted here
// and validated by the caller when required. Value is never trimmed — secret
// values are byte-exact.
func parseSecretArgs(args map[string]any, wantKey, wantValue bool) (secretArgs, *mcpsdk.CallToolResult) {
	var out secretArgs
	out.projectID = secretIDArg(args, "project_id")
	out.environmentID = secretIDArg(args, "environment_id")
	out.folderID = secretIDArg(args, "folder_id")
	out.key = secretIDArg(args, "key")
	if args != nil {
		if v, ok := args["value"].(string); ok {
			out.value = v
		}
	}

	if out.projectID == "" || len(out.projectID) > maxMCPSecretIDLen {
		return out, authToolError("project_id is required")
	}
	if out.environmentID == "" || len(out.environmentID) > maxMCPSecretIDLen {
		return out, authToolError("environment_id is required")
	}
	if out.folderID != "" && len(out.folderID) > maxMCPSecretIDLen {
		return out, authToolError("folder_id is invalid")
	}
	if wantKey && (out.key == "" || len(out.key) > maxMCPSecretKeyLen || !validMCPSecretKey.MatchString(out.key)) {
		return out, authToolError("invalid secret key")
	}
	if wantValue && len(out.value) > maxMCPSecretValueLen {
		return out, authToolError("secret value too large")
	}
	return out, nil
}

// secretIDArg extracts a trimmed identifier argument.
func secretIDArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

// principalFromToolContext returns the authenticated principal for a secret
// tool call. Secret tools fail closed when no principal is present.
func principalFromToolContext(ctx context.Context) (Principal, *mcpsdk.CallToolResult) {
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return Principal{}, authToolError("authorization required")
	}
	return p, nil
}

// actorMetadata is the safe audit actor projection included in tool results.
// It never carries the raw credential or the scope list.
func (c *Component) actorMetadata(ctx context.Context) map[string]any {
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return map[string]any{"kind": "anonymous"}
	}
	switch p.Kind {
	case PrincipalOrg:
		return map[string]any{"kind": "org", "org_id": p.OrgID}
	case PrincipalSite:
		return map[string]any{"kind": "site", "site_id": p.SiteID}
	default:
		return map[string]any{"kind": "anonymous"}
	}
}

// auditSecret logs a value-free audit event with actor and scope metadata.
// Plaintext, ciphertext, and raw credentials never reach the log.
func (c *Component) auditSecret(ctx context.Context, tool, action, projectID, environmentID, folderID, key string) {
	c.logger.Info("mcp secret audit",
		"tool", tool,
		"action", action,
		"actor", fmt.Sprint(c.actorMetadata(ctx)),
		"project_id", projectID,
		"environment_id", environmentID,
		"folder_id", folderID,
		"key", key,
	)
}

// resolveSecretFolder returns the folder ID to operate on: the caller-supplied
// folder_id when present, otherwise the default folder resolved through the
// manager (which verifies the project → environment chain and ownership).
func (c *Component) resolveSecretFolder(ctx context.Context, p Principal, projectID, environmentID, folderID string) (string, *mcpsdk.CallToolResult) {
	if folderID != "" {
		return folderID, nil
	}
	if c.projectSecretManager == nil {
		return "", authToolError("secret tools require a SecretManager")
	}
	id, err := c.projectSecretManager.EnsureFolder(ctx, p, projectID, environmentID, secretDefaultFolderName)
	if err != nil {
		c.logger.Warn("mcp secret folder resolution failed", "project_id", projectID, "error", err)
		return "", authToolError(safeSecretError(err))
	}
	return id, nil
}

// registerSecretTools registers the five native Project secret tools.
func (c *Component) registerSecretTools(srv *mcpsdk.Server) {
	readGate := toolGate{required: []string{ScopeSecretsRead}, siteAllowed: true, siteRequired: ScopeSecretsRead}
	writeGate := toolGate{required: []string{ScopeSecretsWrite}, siteAllowed: true, siteRequired: ScopeSecretsWrite}

	c.registerScopedTool(srv, tierRead, readGate, &mcpsdk.Tool{
		Name:        "list_project_secrets",
		Description: "List project secret metadata (masked previews only, never values) for an environment folder. Requires project_id and environment_id; folder_id defaults to the default folder.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.projectSecretManager == nil {
			return textResult("Secret tools require a SecretManager. Start BigBase with the secret manager enabled."), nil, nil
		}
		args, errResult := parseToolArguments(req)
		if errResult != nil {
			return errResult, nil, nil
		}
		sa, errResult := parseSecretArgs(args, false, false)
		if errResult != nil {
			return errResult, nil, nil
		}
		p, errResult := principalFromToolContext(ctx)
		if errResult != nil {
			return errResult, nil, nil
		}
		if errResult := c.authorizeProjectTarget(ctx, sa.projectID); errResult != nil {
			return errResult, nil, nil
		}
		folderID, errResult := c.resolveSecretFolder(ctx, p, sa.projectID, sa.environmentID, sa.folderID)
		if errResult != nil {
			return errResult, nil, nil
		}
		items, err := c.projectSecretManager.ListSecrets(ctx, p, sa.projectID, sa.environmentID, folderID)
		if err != nil {
			c.logger.Error("mcp list_project_secrets failed", "project_id", sa.projectID, "error", err)
			return authToolError(safeSecretError(err)), nil, nil
		}
		if len(items) > maxMCPSecretList {
			items = items[:maxMCPSecretList]
		}
		c.auditSecret(ctx, "list_project_secrets", "list_secrets", sa.projectID, sa.environmentID, folderID, "")
		return textResult(formatJSON(map[string]any{
			"project_id":     sa.projectID,
			"environment_id": sa.environmentID,
			"folder_id":      folderID,
			"secrets":        items,
			"count":          len(items),
			"actor":          c.actorMetadata(ctx),
		})), nil, nil
	})

	c.registerScopedTool(srv, tierRead, readGate, &mcpsdk.Tool{
		Name:        "get_project_secret",
		Description: "Get metadata for one project secret. Metadata-only: returns a masked value preview and never returns plaintext. Use read_project_secret_value for an explicit value read.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.projectSecretManager == nil {
			return textResult("Secret tools require a SecretManager. Start BigBase with the secret manager enabled."), nil, nil
		}
		args, errResult := parseToolArguments(req)
		if errResult != nil {
			return errResult, nil, nil
		}
		sa, errResult := parseSecretArgs(args, true, false)
		if errResult != nil {
			return errResult, nil, nil
		}
		p, errResult := principalFromToolContext(ctx)
		if errResult != nil {
			return errResult, nil, nil
		}
		if errResult := c.authorizeProjectTarget(ctx, sa.projectID); errResult != nil {
			return errResult, nil, nil
		}
		folderID, errResult := c.resolveSecretFolder(ctx, p, sa.projectID, sa.environmentID, sa.folderID)
		if errResult != nil {
			return errResult, nil, nil
		}
		meta, err := c.projectSecretManager.GetSecretMetadata(ctx, p, sa.projectID, sa.environmentID, folderID, sa.key)
		if err != nil {
			c.logger.Error("mcp get_project_secret failed", "project_id", sa.projectID, "key", sa.key, "error", err)
			return authToolError(safeSecretError(err)), nil, nil
		}
		c.auditSecret(ctx, "get_project_secret", "describe_secret", sa.projectID, sa.environmentID, folderID, sa.key)
		return textResult(formatJSON(map[string]any{
			"secret": meta,
			"actor":  c.actorMetadata(ctx),
		})), nil, nil
	})

	c.registerScopedTool(srv, tierRead, readGate, &mcpsdk.Tool{
		Name:        "read_project_secret_value",
		Description: "Explicitly read the current plaintext value of a project secret. This is the ONLY tool that returns plaintext; it requires mcp:secrets:read and is audited.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.projectSecretManager == nil {
			return textResult("Secret tools require a SecretManager. Start BigBase with the secret manager enabled."), nil, nil
		}
		args, errResult := parseToolArguments(req)
		if errResult != nil {
			return errResult, nil, nil
		}
		sa, errResult := parseSecretArgs(args, true, false)
		if errResult != nil {
			return errResult, nil, nil
		}
		p, errResult := principalFromToolContext(ctx)
		if errResult != nil {
			return errResult, nil, nil
		}
		if errResult := c.authorizeProjectTarget(ctx, sa.projectID); errResult != nil {
			return errResult, nil, nil
		}
		folderID, errResult := c.resolveSecretFolder(ctx, p, sa.projectID, sa.environmentID, sa.folderID)
		if errResult != nil {
			return errResult, nil, nil
		}
		value, err := c.projectSecretManager.ReadSecretValue(ctx, p, sa.projectID, sa.environmentID, folderID, sa.key)
		if err != nil {
			c.logger.Error("mcp read_project_secret_value failed", "project_id", sa.projectID, "key", sa.key, "error", err)
			return authToolError(safeSecretError(err)), nil, nil
		}
		c.auditSecret(ctx, "read_project_secret_value", "read_secret_value", sa.projectID, sa.environmentID, folderID, sa.key)
		return textResult(formatJSON(map[string]any{
			"secret_id": value.SecretID,
			"key":       value.Key,
			"version":   value.Version,
			"value":     value.Value,
			"actor":     c.actorMetadata(ctx),
		})), nil, nil
	})

	c.registerScopedTool(srv, tierWrite, writeGate, &mcpsdk.Tool{
		Name:        "set_project_secret",
		Description: "Create or update a project secret (immutable versions; update appends a version). Requires mcp:secrets:write. Returns masked metadata only.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.projectSecretManager == nil {
			return textResult("Secret tools require a SecretManager. Start BigBase with the secret manager enabled."), nil, nil
		}
		args, errResult := parseToolArguments(req)
		if errResult != nil {
			return errResult, nil, nil
		}
		sa, errResult := parseSecretArgs(args, true, true)
		if errResult != nil {
			return errResult, nil, nil
		}
		if sa.value == "" {
			return authToolError("value is required"), nil, nil
		}
		p, errResult := principalFromToolContext(ctx)
		if errResult != nil {
			return errResult, nil, nil
		}
		if errResult := c.authorizeProjectTarget(ctx, sa.projectID); errResult != nil {
			return errResult, nil, nil
		}
		folderID, errResult := c.resolveSecretFolder(ctx, p, sa.projectID, sa.environmentID, sa.folderID)
		if errResult != nil {
			return errResult, nil, nil
		}
		meta, err := c.projectSecretManager.UpdateSecret(ctx, p, sa.projectID, sa.environmentID, folderID, sa.key, sa.value)
		if errors.Is(err, ErrSecretNotFound) {
			// set is create-or-update: the first write creates the first
			// immutable version, later writes append versions.
			meta, err = c.projectSecretManager.CreateSecret(ctx, p, sa.projectID, sa.environmentID, folderID, sa.key, sa.value)
		}
		if err != nil {
			c.logger.Error("mcp set_project_secret failed", "project_id", sa.projectID, "key", sa.key, "error", err)
			return authToolError(safeSecretError(err)), nil, nil
		}
		c.auditSecret(ctx, "set_project_secret", "set_secret", sa.projectID, sa.environmentID, folderID, sa.key)
		return textResult(formatJSON(map[string]any{
			"secret": meta,
			"action": "set",
			"actor":  c.actorMetadata(ctx),
		})), nil, nil
	})

	c.registerScopedTool(srv, tierWrite, writeGate, &mcpsdk.Tool{
		Name:        "delete_project_secret",
		Description: "Delete a project secret and all its versions. Requires mcp:secrets:write.",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, any, error) {
		if c.projectSecretManager == nil {
			return textResult("Secret tools require a SecretManager. Start BigBase with the secret manager enabled."), nil, nil
		}
		args, errResult := parseToolArguments(req)
		if errResult != nil {
			return errResult, nil, nil
		}
		sa, errResult := parseSecretArgs(args, true, false)
		if errResult != nil {
			return errResult, nil, nil
		}
		p, errResult := principalFromToolContext(ctx)
		if errResult != nil {
			return errResult, nil, nil
		}
		if errResult := c.authorizeProjectTarget(ctx, sa.projectID); errResult != nil {
			return errResult, nil, nil
		}
		folderID, errResult := c.resolveSecretFolder(ctx, p, sa.projectID, sa.environmentID, sa.folderID)
		if errResult != nil {
			return errResult, nil, nil
		}
		if err := c.projectSecretManager.DeleteSecret(ctx, p, sa.projectID, sa.environmentID, folderID, sa.key); err != nil {
			c.logger.Error("mcp delete_project_secret failed", "project_id", sa.projectID, "key", sa.key, "error", err)
			return authToolError(safeSecretError(err)), nil, nil
		}
		c.auditSecret(ctx, "delete_project_secret", "delete_secret", sa.projectID, sa.environmentID, folderID, sa.key)
		return textResult(formatJSON(map[string]any{
			"project_id":     sa.projectID,
			"environment_id": sa.environmentID,
			"key":            sa.key,
			"status":         "deleted",
			"actor":          c.actorMetadata(ctx),
		})), nil, nil
	})
}

// parseToolArguments unmarshals the tool request arguments into a map. A
// malformed argument payload is a bounded generic error.
func parseToolArguments(req *mcpsdk.CallToolRequest) (map[string]any, *mcpsdk.CallToolResult) {
	if req == nil || req.Params.Arguments == nil {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, authToolError("invalid arguments")
	}
	return args, nil
}
