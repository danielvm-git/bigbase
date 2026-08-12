package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// secrets_api.go implements the e89s04 secret REST surface. The adapter is
// deliberately thin: it resolves the authenticated organization and Project
// role through the shared auth.SecretPolicy helper (the single authorization
// source for list, value-read, mutation, and version routes), enforces the
// security bounds and mutation rate limits at the boundary, delegates storage
// to the frozen secrets.SecretManager seam, and emits value-free audit events.
//
// Responses are metadata-only: lists and mutations carry masked previews and
// never contain plaintext or ciphertext. The explicit value route
// (…/secrets/{key}/value) is the sole response that carries a plaintext value
// and it requires the read-value action.
//
// main.go is coordinator-owned; this file exposes SecretsAPIOptions and
// SecretsAPI.Handler for composition-root wiring instead of editing main.go.

const (
	// secretDefaultFolderName is the folder the REST surface exposes. The
	// first release exposes the default folder; folder-specific policies are
	// represented for future extension (e89s04 §9).
	secretDefaultFolderName = "default"

	// Security bounds (e89s04 §13). These are security contracts, not UI
	// hints: over-limit requests are rejected before persistence.
	maxSecretKeyBytes   = 128
	maxSecretValueBytes = 64 * 1024
	// MaxSecretBatchKeys bounds a single import/mutation batch. The canonical
	// routes are single-key; the bound anchors the future import endpoint and
	// the s05 .env import loop, and caps list/version response sizes below.
	MaxSecretBatchKeys    = 1000
	maxSecretListItems    = MaxSecretBatchKeys
	maxSecretVersionItems = MaxSecretBatchKeys
	maxSecretBodyBytes    = 1 << 20 // 1 MiB: one 64 KiB value plus JSON overhead

	// defaultSecretMutationLimit is the contract mutation budget: at most 30
	// mutating requests per minute per authenticated actor and Project.
	defaultSecretMutationLimit = 30
)

var validSecretKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// SecretsAPIOptions configures the secret REST adapter. Manager is required;
// DB is used for audit records and must be the shared component database.
// Policy defaults to auth.NewSecretPolicy(DB) when nil. MutationLimit and
// MutationWindow default to the contract values (30 per minute) when zero.
type SecretsAPIOptions struct {
	Manager secrets.SecretManager
	DB      kernel.DBer
	Logger  kernel.Logger
	Policy  *auth.SecretPolicy

	MutationLimit  int
	MutationWindow time.Duration
}

// SecretsAPI is the canonical Project secret REST adapter. Construct one via
// NewSecretsAPI and mount Handler (or ServeHTTP) at the canonical route
// prefixes from the composition root.
type SecretsAPI struct {
	manager secrets.SecretManager
	db      kernel.DBer
	logger  kernel.Logger
	policy  *auth.SecretPolicy
	limiter *secretRateLimiter
	audit   *secretAudit
	mux     *http.ServeMux
}

// NewSecretsAPI validates options and constructs the adapter.
func NewSecretsAPI(opts SecretsAPIOptions) (*SecretsAPI, error) {
	if opts.Manager == nil {
		return nil, errors.New("secrets api: SecretManager is required")
	}
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	policy := opts.Policy
	if policy == nil {
		if opts.DB == nil {
			return nil, errors.New("secrets api: DB is required when no policy is provided")
		}
		policy = auth.NewSecretPolicy(opts.DB)
	}
	limit := opts.MutationLimit
	if limit <= 0 {
		limit = defaultSecretMutationLimit
	}
	window := opts.MutationWindow
	if window <= 0 {
		window = time.Minute
	}
	a := &SecretsAPI{
		manager: opts.Manager,
		db:      opts.DB,
		logger:  logger,
		policy:  policy,
		limiter: newSecretRateLimiter(limit, window),
		audit:   newSecretAudit(opts.DB, logger),
	}
	a.mux = http.NewServeMux()
	a.mux.HandleFunc("GET /api/projects/{project}/environments/{env}/secrets", a.handleListSecrets)
	a.mux.HandleFunc("POST /api/projects/{project}/environments/{env}/secrets", a.handleCreateSecret)
	a.mux.HandleFunc("GET /api/projects/{project}/environments/{env}/secrets/{key}", a.handleGetSecret)
	a.mux.HandleFunc("PUT /api/projects/{project}/environments/{env}/secrets/{key}", a.handleUpdateSecret)
	a.mux.HandleFunc("DELETE /api/projects/{project}/environments/{env}/secrets/{key}", a.handleDeleteSecret)
	a.mux.HandleFunc("GET /api/projects/{project}/environments/{env}/secrets/{key}/versions", a.handleListVersions)
	a.mux.HandleFunc("GET /api/projects/{project}/environments/{env}/secrets/{key}/value", a.handleReadValue)
	return a, nil
}

// Handler returns the mounted route mux. The coordinator mounts it at the two
// canonical prefixes:
//
//	p.Handle("/api/projects/{project}/environments/{env}/secrets", sec.ServeHTTP)
//	p.Handle("/api/projects/{project}/environments/{env}/secrets/", sec.ServeHTTP)
//
// Both patterns are strictly more specific than the projects component's
// /api/projects/ mount, so ServeMux precedence keeps secret routes here.
func (a *SecretsAPI) Handler() http.Handler { return a.mux }

// ServeHTTP lets the coordinator mount the adapter handler directly.
func (a *SecretsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

// handleListSecrets returns metadata-only secret listings (SC-e89s04-P0-01).
func (a *SecretsAPI) handleListSecrets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	if !a.authorize(w, r, projectID, auth.SecretActionDescribe) {
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	items, err := a.manager.ListSecrets(r.Context(), projectID, envID, folderID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	if len(items) > maxSecretListItems {
		items = items[:maxSecretListItems]
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

// handleCreateSecret creates a secret and returns metadata plus a masked
// preview only (SC-e89s04-P0-01).
func (a *SecretsAPI) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	if !a.authorize(w, r, projectID, auth.SecretActionCreate) {
		return
	}
	if !a.allowMutation(w, r, projectID) {
		return
	}
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if !decodeSecretBody(w, r, &req) {
		return
	}
	if err := validateSecretInput(req.Key, req.Value); err != nil {
		writeSecretError(w, http.StatusBadRequest, err.Error())
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	meta, err := a.manager.CreateSecret(r.Context(), projectID, envID, folderID, req.Key, req.Value)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	a.audit.record(r, "secret.created", req.Key, projectID, envID, "create_secret")
	kernel.WriteJSON(w, http.StatusCreated, map[string]any{"data": meta})
}

// handleGetSecret returns metadata plus a masked preview for one secret.
func (a *SecretsAPI) handleGetSecret(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	key := r.PathValue("key")
	if !a.authorize(w, r, projectID, auth.SecretActionDescribe) {
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	meta, err := a.manager.GetSecretMetadata(r.Context(), projectID, envID, folderID, key)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": meta})
}

// handleUpdateSecret appends an immutable version and returns metadata plus a
// masked preview only.
func (a *SecretsAPI) handleUpdateSecret(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	key := r.PathValue("key")
	if !a.authorize(w, r, projectID, auth.SecretActionUpdate) {
		return
	}
	if !a.allowMutation(w, r, projectID) {
		return
	}
	var req struct {
		Value string `json:"value"`
	}
	if !decodeSecretBody(w, r, &req) {
		return
	}
	if err := validateSecretInput(key, req.Value); err != nil {
		writeSecretError(w, http.StatusBadRequest, err.Error())
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	meta, err := a.manager.UpdateSecret(r.Context(), projectID, envID, folderID, key, req.Value)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	a.audit.record(r, "secret.updated", key, projectID, envID, "update_secret")
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": meta})
}

// handleDeleteSecret removes a secret and its versions.
func (a *SecretsAPI) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	key := r.PathValue("key")
	if !a.authorize(w, r, projectID, auth.SecretActionDelete) {
		return
	}
	if !a.allowMutation(w, r, projectID) {
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	if err := a.manager.DeleteSecret(r.Context(), projectID, envID, folderID, key); err != nil {
		a.writeManagerError(w, err)
		return
	}
	a.audit.record(r, "secret.deleted", key, projectID, envID, "delete_secret")
	w.WriteHeader(http.StatusNoContent)
}

// handleListVersions returns bounded, ordered, metadata-only version listings.
// It is independent from value-read authorization (SC-e89s04-P1-06): describe
// standing is sufficient.
func (a *SecretsAPI) handleListVersions(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	key := r.PathValue("key")
	if !a.authorize(w, r, projectID, auth.SecretActionListVersions) {
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	versions, err := a.manager.ListVersions(r.Context(), projectID, envID, folderID, key)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	if len(versions) > maxSecretVersionItems {
		versions = versions[len(versions)-maxSecretVersionItems:]
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": versions})
}

// handleReadValue is the explicit value-read route. It is the ONLY response
// that carries plaintext and it requires the read-value action
// (SC-e89s04-P0-02).
func (a *SecretsAPI) handleReadValue(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	envID := r.PathValue("env")
	key := r.PathValue("key")
	if !a.authorize(w, r, projectID, auth.SecretActionReadValue) {
		return
	}
	folderID, err := a.resolveFolder(r.Context(), projectID, envID)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	value, err := a.manager.ReadSecretValue(r.Context(), projectID, envID, folderID, key)
	if err != nil {
		a.writeManagerError(w, err)
		return
	}
	a.audit.record(r, "secret.value_read", key, projectID, envID, "read_secret_value")
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": value})
}

// authorize enforces the shared policy for the requested action and writes the
// standard JSON error on denial: 401 when no authenticated identity is in
// context, 403 for authenticated callers without the action or organization,
// and a non-disclosing 404 for cross-organization targets.
func (a *SecretsAPI) authorize(w http.ResponseWriter, r *http.Request, projectID string, action auth.SecretAction) bool {
	ctx := r.Context()
	_, hasUser := auth.UserIDFromContext(ctx)
	_, hasOrg := auth.OrgIDFromContext(ctx)
	switch {
	case !hasUser && !hasOrg:
		writeSecretError(w, http.StatusUnauthorized, "authorization required")
		return false
	case !hasOrg:
		writeSecretError(w, http.StatusForbidden, "no organization")
		return false
	}
	if err := a.policy.Authorize(ctx, projectID, action); err != nil {
		switch {
		case errors.Is(err, auth.ErrSecretProjectNotFound):
			// Cross-organization and missing targets are indistinguishable.
			writeSecretError(w, http.StatusNotFound, "not found")
		case errors.Is(err, auth.ErrSecretActionForbidden):
			writeSecretError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, auth.ErrSecretOrganizationRequired):
			writeSecretError(w, http.StatusForbidden, "organization required")
		default:
			a.logger.Error("secret policy check failed", "project", projectID, "action", action, "error", err)
			writeSecretError(w, http.StatusInternalServerError, "internal error")
		}
		return false
	}
	return true
}

// allowMutation applies the per-(actor, project) mutation rate limit before
// persistence. Over-limit requests get 429 with Retry-After.
func (a *SecretsAPI) allowMutation(w http.ResponseWriter, r *http.Request, projectID string) bool {
	actor := secretActorKey(r.Context())
	allowed, retryAfter := a.limiter.allow(actor, projectID)
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		writeSecretError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return false
	}
	return true
}

// resolveFolder resolves the exposed default folder, verifying the project →
// environment chain and organization ownership through the SecretManager seam
// (non-disclosing 404 for cross-org targets).
func (a *SecretsAPI) resolveFolder(ctx context.Context, projectID, envID string) (string, error) {
	folder, err := a.manager.EnsureFolder(ctx, projectID, envID, secretDefaultFolderName)
	if err != nil {
		return "", err
	}
	return folder.ID, nil
}

// decodeSecretBody decodes a bounded JSON body into dst. Malformed or
// oversized bodies are rejected with a generic 400 before any persistence.
func decodeSecretBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxSecretBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeSecretError(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}

// validateSecretInput enforces the secret key/value bounds at the boundary
// (e89s04 §13): 128-byte keys matching the uppercase identifier contract, and
// 64 KiB values. The SecretManager re-validates as defense in depth.
func validateSecretInput(key, value string) error {
	if key == "" || len(key) > maxSecretKeyBytes || !validSecretKey.MatchString(key) {
		return errors.New("invalid secret key")
	}
	if len(value) > maxSecretValueBytes {
		return fmt.Errorf("secret value exceeds %d bytes", maxSecretValueBytes)
	}
	return nil
}

// writeManagerError maps SecretManager failures to the standard JSON error
// shape. Cross-organization and missing targets collapse to a non-disclosing
// 404; crypto failures fail closed with a generic 500.
func (a *SecretsAPI) writeManagerError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal error"
	switch {
	case errors.Is(err, secrets.ErrOrganizationRequired):
		status, message = http.StatusForbidden, "organization required"
	case errors.Is(err, secrets.ErrProjectNotFound),
		errors.Is(err, secrets.ErrEnvironmentNotFound),
		errors.Is(err, secrets.ErrFolderNotFound),
		errors.Is(err, secrets.ErrSecretNotFound),
		errors.Is(err, secrets.ErrVersionNotFound):
		status, message = http.StatusNotFound, "not found"
	case errors.Is(err, secrets.ErrSecretAlreadyExists):
		status, message = http.StatusConflict, "secret already exists"
	case strings.Contains(err.Error(), "invalid secret key"),
		strings.Contains(err.Error(), "exceeds"),
		strings.Contains(err.Error(), "required"),
		strings.Contains(err.Error(), "too long"):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, secrets.ErrInvalidKeyMaterial),
		errors.Is(err, secrets.ErrDecryptionFailed),
		errors.Is(err, secrets.ErrActiveKeyNotFound):
		// Fail closed: never leak crypto or key details to clients.
		status, message = http.StatusInternalServerError, "internal error"
	default:
		a.logger.Error("secret manager operation failed", "error", err)
	}
	writeSecretError(w, status, message)
}

// writeSecretError writes the standard {"error": "..."} JSON error shape.
func writeSecretError(w http.ResponseWriter, status int, message string) {
	kernel.WriteJSON(w, status, map[string]string{"error": message})
}
