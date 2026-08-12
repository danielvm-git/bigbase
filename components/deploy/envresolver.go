package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/internal/envcrypto"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// EnvScope selects which subset of site env vars a resolution targets.
//
//   - ScopeBuild: vars flagged is_build_time=1, used by the build command env.
//   - ScopeRuntime: vars flagged is_runtime=1, used by the started app env.
type EnvScope int

const (
	// ScopeBuild resolves build-time env vars (the build tool's environment).
	ScopeBuild EnvScope = iota
	// ScopeRuntime resolves runtime env vars (the deployed app's environment).
	ScopeRuntime
)

// column returns the site_env_vars flag column this scope filters on.
func (s EnvScope) column() string {
	if s == ScopeBuild {
		return "is_build_time"
	}
	return "is_runtime"
}

// SecretResolver is the deployment-facing projection of the native SecretManager
// seam (e89s03). The secrets component satisfies it structurally at the
// composition root; deploy consumes the frozen public types and never touches
// the store implementation. Returning nil from NewEnvResolver disables native
// Project resolution entirely (legacy-only deployments).
type SecretResolver interface {
	ListFolders(ctx context.Context, projectID, environmentID string) ([]secrets.SecretFolder, error)
	ListSecrets(ctx context.Context, projectID, environmentID, folderID string) ([]secrets.SecretMetadata, error)
	ReadSecretValue(ctx context.Context, projectID, environmentID, folderID, key string) (secrets.SecretValue, error)
}

// ResolveOptions are the per-deployment inputs layered on top of the platform
// baseline by ResolveOptions. Both fields are optional.
type ResolveOptions struct {
	// ManifestEnv is the manifest (bigbase.yaml/toml) [env] configuration. It
	// is applied after the platform baseline and before Project/Site secrets,
	// so a secret can override a manifest default but a manifest value never
	// overrides a secret.
	ManifestEnv map[string]string
	// Reserved are "KEY=value" pairs applied last: they win over every other
	// layer and are never treated as secrets for redaction. They carry the
	// runtime-ownership values (PORT, WRITABLE_DIR, DB_PATH/DATABASE_URL) that
	// a deployment must never let a repo or secret override.
	Reserved []string
}

// EnvResolver is the single owner of "what environment does this deployment
// see, and what must never appear in a log." Both the build path (the deploy
// engine's build command) and the runtime path (the started app) resolve env
// through it, so precedence and redaction are defined in exactly one place.
//
// This is the seam introduced for issue #41; it unblocks epics e41 (Env Vars
// UI), e47 (Secrets Mgmt), and e51 (Preview Envs). e89s06 extended it with the
// native Project Environment layer, the legacy Site compatibility layer, and
// the reserved runtime layer.
type EnvResolver struct {
	db             kernel.DBer
	envKey         []byte
	logger         kernel.Logger
	buildHome      string
	secretResolver SecretResolver
}

// NewEnvResolver constructs a resolver backed by the given DB and (optional)
// AES-256-GCM encryption key. The key matches the one configured on the
// deploy component; when nil, stored values are treated as plaintext
// (envcrypto no-op mode), matching the existing FetchSiteEnvVars behaviour.
// buildHome is the tuned build cache home used for the build scope baseline
// (mirrors the deploy component's buildHomeDir); empty falls back to
// BIGBASE_HOME then os.Environ, the same as BuildEnv.
// secretResolver is the injected native SecretManager seam (e89s03); nil
// disables native Project resolution.
func NewEnvResolver(db kernel.DBer, envKey []byte, logger kernel.Logger, buildHome string, secretResolver SecretResolver) *EnvResolver {
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	return &EnvResolver{db: db, envKey: envKey, logger: logger, buildHome: strings.TrimSpace(buildHome), secretResolver: secretResolver}
}

// ResolvedEnv is the result of Resolve: the full environment plus the set of
// keys whose values were sourced from the secret store. SecretKeys is the
// redaction primitive — callers that log env consult it via Redact.
type ResolvedEnv struct {
	// Environ is the merged environment as "KEY=value" lines, ready for
	// exec.Cmd.Env.
	Environ []string
	// secretKeys is the set of keys that came from the secret stores (native
	// Project Environment values and legacy site_env_vars). Values from these
	// stores are encrypted at rest and must never appear in logs, so any
	// log-facing view must mask them via Redact.
	secretKeys map[string]struct{}
	// conflicts lists keys where a legacy Site compatibility value overrode a
	// native Project value during dual-read. Values are never recorded — only
	// keys — so operators see the collision surface without any plaintext.
	conflicts []string
}

// SecretKeys returns the set of keys whose values were sourced from the
// secret store. The returned map is a defensive copy and is safe for callers
// to retain.
func (r *ResolvedEnv) SecretKeys() map[string]struct{} {
	if r == nil || r.secretKeys == nil {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(r.secretKeys))
	for k := range r.secretKeys {
		out[k] = struct{}{}
	}
	return out
}

// IsSecret reports whether the given key was sourced from the secret store.
func (r *ResolvedEnv) IsSecret(key string) bool {
	if r == nil || r.secretKeys == nil {
		return false
	}
	_, ok := r.secretKeys[key]
	return ok
}

// Conflicts returns the keys where a legacy Site compatibility value overrode
// a native Project value during dual-read. The returned slice is a defensive
// copy; it contains keys only, never values.
func (r *ResolvedEnv) Conflicts() []string {
	if r == nil || len(r.conflicts) == 0 {
		return nil
	}
	out := make([]string, len(r.conflicts))
	copy(out, r.conflicts)
	return out
}

// Resolve computes the environment for a deployment in the given scope with
// no per-deployment inputs beyond the platform baseline and the Site chain.
// It is kept for callers that need only the base layers; startApp and buildApp
// use ResolveOptions to layer manifest configuration and reserved values.
func (r *EnvResolver) Resolve(ctx context.Context, siteID string, scope EnvScope) (*ResolvedEnv, error) {
	return r.ResolveOptions(ctx, siteID, scope, ResolveOptions{})
}

// ResolveOptions computes the environment for a deployment in the given scope.
//
// Precedence (lowest → highest, last write wins):
//
//  1. Platform build/runtime defaults — os.Environ() plus the deploy
//     component's build-time home/cache tuning (see BuildEnv). These are the
//     system-supplied baseline.
//  2. Manifest configuration (opts.ManifestEnv) — repo-declared [env] values.
//     Not a secret: the manifest ships in the repository.
//  3. Native Project Environment secrets — resolved through the injected
//     SecretResolver seam for the site's project and production environment.
//     Every value from this store is treated as a secret for redaction.
//  4. Legacy Site compatibility values — site_env_vars for the requested
//     scope, decrypted from the legacy store. A Site value overrides a
//     Project value on key collision (SC-e89s06-P0-01) and the collision is
//     reported through Conflicts without logging values.
//  5. Reserved values (opts.Reserved) — applied last so runtime ownership
//     values (PORT, WRITABLE_DIR, DB connection) always win.
//
// Within a scope, a later-applied layer overrides an earlier one, so a
// site-defined var (e.g. NODE_ENV=production) takes precedence over a
// platform default. This keeps the precedence rules auditable in one module.
//
// Protected decryption failures — a native Project value or a selected legacy
// Site value that cannot be decrypted — are fatal: delivery must stop rather
// than boot without a secret (SC-e89s01-P0-05).
func (r *EnvResolver) ResolveOptions(ctx context.Context, siteID string, scope EnvScope, opts ResolveOptions) (*ResolvedEnv, error) {
	out := &ResolvedEnv{
		Environ:    r.defaultEnviron(scope),
		secretKeys: make(map[string]struct{}),
	}
	merged := keyValueMap(out.Environ)

	// Layer 2: manifest configuration.
	for k, v := range opts.ManifestEnv {
		merged[k] = v
	}

	if siteID != "" && r.db != nil {
		// Layer 3: native Project Environment secrets.
		if err := r.resolveProjectSecrets(ctx, siteID, merged, out); err != nil {
			return nil, err
		}
		// Layer 4: legacy Site compatibility values (Site wins on collision).
		if err := r.resolveSiteEnvVars(ctx, siteID, scope, merged, out); err != nil {
			return nil, err
		}
	}

	// Layer 5: reserved runtime values — final and authoritative.
	for _, kv := range opts.Reserved {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		merged[kv[:idx]] = kv[idx+1:]
	}

	out.Environ = mapToEnviron(merged)
	return out, nil
}

// resolveProjectSecrets applies layer 3: every native Project secret in the
// site's production environment. Native secrets are scope-agnostic (the native
// model has no build/runtime flag), so they appear in both build and runtime
// scopes; scope isolation remains the legacy Site layer's responsibility.
// A value that cannot be decrypted is fatal.
func (r *EnvResolver) resolveProjectSecrets(ctx context.Context, siteID string, merged map[string]string, out *ResolvedEnv) error {
	if r.secretResolver == nil {
		return nil
	}
	projectID, envID, orgID, err := r.siteProjectEnvironment(ctx, siteID)
	if err != nil {
		return err
	}
	if projectID == "" || envID == "" {
		return nil
	}
	// The resolver is an internal trusted seam: tenant identity comes from the
	// persisted site → project chain, never from caller input.
	secCtx := auth.WithOrgID(ctx, orgID)

	folders, err := r.secretResolver.ListFolders(secCtx, projectID, envID)
	if err != nil {
		return fmt.Errorf("resolve project secret folders: %w", err)
	}
	// Folders are ordered by name from the seam; within a folder, secrets are
	// ordered by key, and a later folder overrides an earlier one on collision.
	for _, folder := range folders {
		metas, err := r.secretResolver.ListSecrets(secCtx, projectID, envID, folder.ID)
		if err != nil {
			return fmt.Errorf("resolve project secrets in %s: %w", folder.Name, err)
		}
		for _, meta := range metas {
			val, err := r.secretResolver.ReadSecretValue(secCtx, projectID, envID, folder.ID, meta.Key)
			if err != nil {
				// Fail closed: a selected secret that cannot be decrypted must
				// stop delivery rather than silently disappear.
				return fmt.Errorf("resolve project secret %s: %w", meta.Key, err)
			}
			merged[meta.Key] = val.Value
			out.secretKeys[meta.Key] = struct{}{}
		}
	}
	return nil
}

// resolveSiteEnvVars applies layer 4: legacy site_env_vars rows for the
// requested scope, decrypted from the legacy store. A Site value overrides a
// Project value on collision; the collision is reported (key only, never the
// value) so operators can drive the legacy migration. A selected decryption
// failure is fatal.
func (r *EnvResolver) resolveSiteEnvVars(ctx context.Context, siteID string, scope EnvScope, merged map[string]string, out *ResolvedEnv) error {
	rows, err := r.db.QueryContext(ctx,
		"SELECT key, value_encrypted FROM site_env_vars WHERE site_id = ? AND "+scope.column()+" = 1",
		siteID)
	if err != nil {
		// The table may not exist in older deployments — treat as empty, the
		// same as FetchSiteEnvVars does, so resolution never blocks a deploy.
		if isNoSuchTable(err) {
			return nil
		}
		return fmt.Errorf("resolve site env vars: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, encrypted string
		if err := rows.Scan(&key, &encrypted); err != nil {
			return fmt.Errorf("resolve site env vars")
		}
		value, err := envcrypto.Decrypt(r.envKey, encrypted)
		if err != nil {
			// Protected delivery: a selected Site-secret decryption failure is
			// fatal, never silently dropped (SC-e89s01-P0-05 / SC-e89s06-P0-05).
			return fmt.Errorf("resolve site env var %s: %w", key, err)
		}
		if _, fromProject := out.secretKeys[key]; fromProject {
			// Native-first dual-read collision: Site wins, report the key.
			out.conflicts = append(out.conflicts, key)
			r.logger.Info("site compatibility value overrides project secret", "key", key)
		}
		merged[key] = value
		out.secretKeys[key] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("resolve site env vars iteration: %w", err)
	}
	return nil
}

// siteProjectEnvironment resolves the site → project → production environment
// chain used by native secret resolution. Empty results mean the site has no
// project attachment yet (legacy-only mode). Missing tables degrade to empty
// so older databases resolve exactly as before e89s02.
func (r *EnvResolver) siteProjectEnvironment(ctx context.Context, siteID string) (projectID, envID string, orgID int64, err error) {
	var project sql.NullString
	if err := r.db.QueryRowContext(ctx, `SELECT project_id FROM sites WHERE id = ?`, siteID).Scan(&project); err != nil {
		// Older deployments may lack the sites table entirely or predate the
		// project_id attachment column — degrade to legacy-only resolution.
		if isNoSuchTable(err) || missingProjectColumn(err) || errors.Is(err, sql.ErrNoRows) {
			return "", "", 0, nil
		}
		return "", "", 0, fmt.Errorf("resolve site project: %w", err)
	}
	if !project.Valid || project.String == "" {
		return "", "", 0, nil
	}
	projectID = project.String

	if err := r.db.QueryRowContext(ctx, `SELECT org_id FROM projects WHERE id = ?`, projectID).Scan(&orgID); err != nil {
		if isNoSuchTable(err) || errors.Is(err, sql.ErrNoRows) {
			return "", "", 0, nil
		}
		return "", "", 0, fmt.Errorf("resolve project org: %w", err)
	}

	// Prefer the production environment (the compatibility environment created
	// by projects.EnsureSiteProject); fall back to the earliest environment.
	envID, err = r.productionEnvironmentID(ctx, projectID)
	if err != nil {
		return "", "", 0, err
	}
	return projectID, envID, orgID, nil
}

func (r *EnvResolver) productionEnvironmentID(ctx context.Context, projectID string) (string, error) {
	var envID string
	if err := r.db.QueryRowContext(ctx,
		`SELECT id FROM project_environments WHERE project_id = ? AND slug = 'production' ORDER BY created_at, id LIMIT 1`,
		projectID).Scan(&envID); err == nil {
		return envID, nil
	} else if !errors.Is(err, sql.ErrNoRows) && !isNoSuchTable(err) {
		return "", fmt.Errorf("resolve project environment: %w", err)
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT id FROM project_environments WHERE project_id = ? ORDER BY created_at, id LIMIT 1`,
		projectID).Scan(&envID); err != nil {
		if errors.Is(err, sql.ErrNoRows) || isNoSuchTable(err) {
			return "", nil
		}
		return "", fmt.Errorf("resolve project environment: %w", err)
	}
	return envID, nil
}

// Redact returns a copy of env with every value whose key is in secretKeys
// replaced by a masked form (envcrypto.MaskValue). Non-secret values are
// preserved verbatim. This is the single primitive any consumer that logs
// env must call — build-log capture, debug dumps, diagnostics.
//
// The mask reveals only the last 4 characters (or fully masks short values),
// matching the preview already shown by the sites list endpoint, so log
// output and the UI agree on what a secret "looks like."
func Redact(env []string, secretKeys map[string]struct{}) []string {
	if len(secretKeys) == 0 {
		// Defensive copy so callers can mutate the result freely.
		out := make([]string, len(env))
		copy(out, env)
		return out
	}
	out := make([]string, 0, len(env))
	for _, kv := range env {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			out = append(out, kv)
			continue
		}
		key := kv[:idx]
		if _, secret := secretKeys[key]; secret {
			out = append(out, key+"="+envcrypto.MaskValue(kv[idx+1:]))
			continue
		}
		out = append(out, kv)
	}
	return out
}

// RedactLogText masks secret values anywhere they appear in arbitrary log
// text (e.g. a build tool's captured stderr that echoes its own environment).
// Unlike Redact (which knows the KEY=value structure), this is a
// value-substitution scan over free text: every known secret value is
// replaced by its masked form. The plaintext values are drawn from resolved
// itself, which is why this takes the ResolvedEnv rather than a bare key set.
//
// This is the defense-in-depth path for the build-log leak vector called out
// in issue #41: even if a tool prints an env var verbatim, the value is
// scrubbed before it reaches appendDeployLog. Order secrets longest-first so
// a value that is a prefix of another is not left partially substituted.
func RedactLogText(text string, resolved *ResolvedEnv) string {
	if resolved == nil || len(resolved.secretKeys) == 0 {
		return text
	}
	// Build a deduplicated list of (plaintext, mask) pairs for the secret
	// values actually present in this resolution.
	var pairs []maskPair
	seen := make(map[string]struct{})
	for _, kv := range resolved.Environ {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		key := kv[:idx]
		if !resolved.IsSecret(key) {
			continue
		}
		value := kv[idx+1:]
		if value == "" {
			continue
		}
		if _, dup := seen[value]; dup {
			continue
		}
		seen[value] = struct{}{}
		pairs = append(pairs, maskPair{plain: value, masked: envcrypto.MaskValue(value)})
	}
	if len(pairs) == 0 {
		return text
	}
	// Longest plaintext first to avoid leaving suffixes when one secret value
	// contains another.
	sortPairsByLenDesc(pairs)
	out := text
	for _, p := range pairs {
		out = strings.ReplaceAll(out, p.plain, p.masked)
	}
	return out
}

// maskPair is a plaintext-to-masked substitution used by RedactLogText.
type maskPair struct{ plain, masked string }

// sortPairsByLenDesc orders pairs by descending plaintext length (stable on
// equal lengths). Kept simple and allocation-free.
func sortPairsByLenDesc(p []maskPair) {
	for i := 1; i < len(p); i++ {
		for j := i; j > 0 && len(p[j-1].plain) < len(p[j].plain); j-- {
			p[j-1], p[j] = p[j], p[j-1]
		}
	}
}

// missingProjectColumn reports the SQLite "no such column: project_id" error
// raised when a pre-e89s02 sites table has not yet received the attachment
// column. Resolution degrades to legacy-only in that case.
func missingProjectColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such column: project_id")
}

// defaultEnviron returns the platform-baseline environment for a scope: for
// build, the tuned build home/cache env; for runtime, the raw process env.
// This is layer 1 of Resolve's precedence.
func (r *EnvResolver) defaultEnviron(scope EnvScope) []string {
	if scope == ScopeBuild {
		home := r.buildHome
		if home == "" {
			home = os.Getenv("BIGBASE_HOME")
		}
		return BuildEnv(home)
	}
	return os.Environ()
}

// keyValueMap parses "KEY=value" lines into a map.
func keyValueMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			m[kv] = ""
			continue
		}
		m[kv[:idx]] = kv[idx+1:]
	}
	return m
}

// mapToEnviron flattens a map to sorted "KEY=value" lines. Sorted output
// keeps resolved environments deterministic (eases testing and diffing).
func mapToEnviron(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort: env counts are modest and we avoid pulling sort.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+m[k])
	}
	return out
}
