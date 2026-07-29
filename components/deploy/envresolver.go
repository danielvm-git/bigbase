package deploy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/danielvm/bigbase/components/internal/envcrypto"
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

// EnvResolver is the single owner of "what environment does this deployment
// see, and what must never appear in a log." Both the build path (the deploy
// engine's build command) and the runtime path (the started app) resolve env
// through it, so precedence and redaction are defined in exactly one place.
//
// This is the seam introduced for issue #41; it unblocks epics e41 (Env Vars
// UI), e47 (Secrets Mgmt), and e51 (Preview Envs).
type EnvResolver struct {
	db        kernel.DBer
	envKey    []byte
	logger    kernel.Logger
	buildHome string
}

// NewEnvResolver constructs a resolver backed by the given DB and (optional)
// AES-256-GCM encryption key. The key matches the one configured on the
// deploy component; when nil, stored values are treated as plaintext
// (envcrypto no-op mode), matching the existing FetchSiteEnvVars behaviour.
// buildHome is the tuned build cache home used for the build scope baseline
// (mirrors the deploy component's buildHomeDir); empty falls back to
// BIGBASE_HOME then os.Environ, the same as BuildEnv.
func NewEnvResolver(db kernel.DBer, envKey []byte, logger kernel.Logger, buildHome string) *EnvResolver {
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	return &EnvResolver{db: db, envKey: envKey, logger: logger, buildHome: strings.TrimSpace(buildHome)}
}

// ResolvedEnv is the result of Resolve: the full environment plus the set of
// keys whose values were sourced from the secret store. SecretKeys is the
// redaction primitive — callers that log env consult it via Redact.
type ResolvedEnv struct {
	// Environ is the merged environment as "KEY=value" lines, ready for
	// exec.Cmd.Env.
	Environ []string
	// secretKeys is the set of keys that came from site_env_vars. Values from
	// this store are encrypted at rest and must never appear in logs, so any
	// log-facing view must mask them via Redact.
	secretKeys map[string]struct{}
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

// Resolve computes the environment for a deployment in the given scope.
//
// Precedence (lowest → highest, last write wins):
//
//  1. Platform build/runtime defaults — os.Environ() plus the deploy
//     component's build-time home/cache tuning (see BuildEnv). These are the
//     system-supplied baseline.
//  2. Site env vars for the requested scope, decrypted from the secret
//     store. Because every site env var is stored encrypted at rest, all of
//     these are treated as secrets for redaction purposes.
//
// Within a scope, a later-applied layer overrides an earlier one, so a
// site-defined var (e.g. NODE_ENV=production) takes precedence over a
// platform default. This keeps the precedence rules auditable in one module.
//
// TODO(e47): when a dedicated secrets layer lands, it will be applied last
// (secrets override user env vars on conflict) — this is the documented seam
// for that addition; the loop below already supports appending further
// layers without touching consumers.
func (r *EnvResolver) Resolve(ctx context.Context, siteID string, scope EnvScope) (*ResolvedEnv, error) {
	out := &ResolvedEnv{
		Environ:    r.defaultEnviron(scope),
		secretKeys: make(map[string]struct{}),
	}

	if siteID == "" || r.db == nil {
		return out, nil
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT key, value_encrypted FROM site_env_vars WHERE site_id = ? AND "+scope.column()+" = 1",
		siteID)
	if err != nil {
		// The table may not exist in older deployments — treat as empty, the
		// same as FetchSiteEnvVars does, so resolution never blocks a deploy.
		if isNoSuchTable(err) {
			return out, nil
		}
		return nil, fmt.Errorf("resolve site env vars: %w", err)
	}
	defer func() { _ = rows.Close() }()

	merged := keyValueMap(out.Environ)
	for rows.Next() {
		var key, encrypted string
		if err := rows.Scan(&key, &encrypted); err != nil {
			r.logger.Warn("scan env var row", "error", err)
			continue
		}
		value, err := envcrypto.Decrypt(r.envKey, encrypted)
		if err != nil {
			r.logger.Warn("decrypt env var", "key", key, "error", err)
			continue
		}
		merged[key] = value
		out.secretKeys[key] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolve site env vars iteration: %w", err)
	}

	out.Environ = mapToEnviron(merged)
	return out, nil
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
