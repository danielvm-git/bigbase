package deploy_test

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/internal/envcrypto"
	"github.com/danielvm/bigbase/kernel"
)

// envcryptoHexToBytes parses a hex string into a byte slice (the form
// parseEnvEncryptionKey expects). Kept local so the test does not depend on
// the unexported deploy parser.
func envcryptoHexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// resolveEnvEncrypted is resolveEnv for cases where the caller has already
// encrypted the stored value (e.g. to exercise the decrypt path with a real
// key). The value passed in vars is written verbatim to value_encrypted.
func resolveEnvEncrypted(t *testing.T, encKey string, vars [][4]any) *deploy.EnvResolver {
	t.Helper()
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	migrateEnvVarsTable(t, database)
	for _, v := range vars {
		key, value := v[0].(string), v[1].(string)
		bt, rt := v[2].(bool), v[3].(bool)
		insertEnvVar(t, database, "site-x", key, value, bt, rt)
	}

	dep := deploy.New(deploy.Options{DB: database, Logger: logger, EnvEncryptionKey: encKey})
	_ = dep.Start(&kernel.Context{})
	t.Cleanup(func() { _ = dep.Stop(&kernel.Context{}) })
	return dep.EnvResolver()
}

// resolveEnv sets up an in-memory deploy + site_env_vars table and returns a
// resolver seeded with the given env vars (key, value, buildTime, runtime).
func resolveEnv(t *testing.T, encKey string, vars [][4]any) *deploy.EnvResolver {
	t.Helper()
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	migrateEnvVarsTable(t, database)
	for _, v := range vars {
		key, value := v[0].(string), v[1].(string)
		bt, rt := v[2].(bool), v[3].(bool)
		insertEnvVar(t, database, "site-x", key, value, bt, rt)
	}

	dep := deploy.New(deploy.Options{DB: database, Logger: logger, EnvEncryptionKey: encKey})
	_ = dep.Start(&kernel.Context{})
	t.Cleanup(func() { _ = dep.Stop(&kernel.Context{}) })
	return dep.EnvResolver()
}

// toMap converts a []string of "KEY=value" into a map for easy assertions.
func toMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			m[kv] = ""
			continue
		}
		m[kv[:i]] = kv[i+1:]
	}
	return m
}

func TestEnvResolverPrecedence(t *testing.T) {
	t.Run("site_env_vars_override_build_defaults", func(t *testing.T) {
		// A var present in the platform defaults (os.Environ) must yield to the
		// site-defined value — the site is the single source of truth for what
		// the deployment sees. This is the precedence the resolver owns.
		t.Setenv("NODE_ENV", "development")
		r := resolveEnv(t, "", [][4]any{
			{"NODE_ENV", "production", true, false},
			{"API_TOKEN", "abc123", true, false},
		})

		resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		m := toMap(resolved.Environ)
		if m["NODE_ENV"] != "production" {
			t.Fatalf("site env var NODE_ENV should override platform default, got %q", m["NODE_ENV"])
		}
		if m["API_TOKEN"] != "abc123" {
			t.Fatalf("expected API_TOKEN=abc123, got %q", m["API_TOKEN"])
		}
	})

	t.Run("scope_build_excludes_runtime_only_vars", func(t *testing.T) {
		r := resolveEnv(t, "", [][4]any{
			{"BUILD_VAR", "b", true, false},
			{"RUNTIME_VAR", "r", false, true},
			{"BOTH_VAR", "both", true, true},
		})

		resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
		if err != nil {
			t.Fatalf("Resolve build: %v", err)
		}
		m := toMap(resolved.Environ)
		if _, ok := m["RUNTIME_VAR"]; ok {
			t.Fatalf("runtime-only var leaked into build scope: %q", m["RUNTIME_VAR"])
		}
		if m["BUILD_VAR"] != "b" {
			t.Fatalf("expected BUILD_VAR=b, got %q", m["BUILD_VAR"])
		}
		if m["BOTH_VAR"] != "both" {
			t.Fatalf("expected BOTH_VAR in build scope, got %q", m["BOTH_VAR"])
		}
	})

	t.Run("scope_runtime_excludes_build_only_vars", func(t *testing.T) {
		r := resolveEnv(t, "", [][4]any{
			{"BUILD_ONLY", "b", true, false},
			{"RUNTIME_ONLY", "r", false, true},
		})

		resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeRuntime)
		if err != nil {
			t.Fatalf("Resolve runtime: %v", err)
		}
		m := toMap(resolved.Environ)
		if _, ok := m["BUILD_ONLY"]; ok {
			t.Fatalf("build-only var leaked into runtime scope: %q", m["BUILD_ONLY"])
		}
		if m["RUNTIME_ONLY"] != "r" {
			t.Fatalf("expected RUNTIME_ONLY=r, got %q", m["RUNTIME_ONLY"])
		}
	})

	t.Run("build_and_runtime_scopes_agree_on_shared_vars", func(t *testing.T) {
		// Both paths must resolve through the same resolver so a var flagged
		// both build+runtime sees an identical value in each path.
		r := resolveEnv(t, "", [][4]any{
			{"SHARED", "same-value", true, true},
		})
		buildEnv, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
		if err != nil {
			t.Fatalf("Resolve build: %v", err)
		}
		rtEnv, err := r.Resolve(context.Background(), "site-x", deploy.ScopeRuntime)
		if err != nil {
			t.Fatalf("Resolve runtime: %v", err)
		}
		if toMap(buildEnv.Environ)["SHARED"] != toMap(rtEnv.Environ)["SHARED"] {
			t.Fatal("build and runtime scopes disagree on a shared var")
		}
	})

	t.Run("empty_site_returns_empty_no_error", func(t *testing.T) {
		r := resolveEnv(t, "", nil)
		resolved, err := r.Resolve(context.Background(), "no-such-site", deploy.ScopeBuild)
		if err != nil {
			t.Fatalf("Resolve unknown site: %v", err)
		}
		if len(resolved.SecretKeys()) != 0 {
			t.Fatalf("expected no secret keys for unknown site, got %v", resolved.SecretKeys())
		}
	})
}

func TestEnvResolverRedaction(t *testing.T) {
	t.Run("redact_masks_secret_sourced_values", func(t *testing.T) {
		// Every value sourced from the secret store (site_env_vars) must be
		// masked in any view derived from ResolvedEnv that a consumer might
		// log. The resolver knows which keys came from the store; Redact uses
		// that knowledge. A non-secret value must come from the platform
		// baseline (os.Environ), NOT the site store — the store is the secret
		// surface, so anything in it is treated as a secret for redaction.
		t.Setenv("PUBLIC_VAR", "hello")
		r := resolveEnv(t, "", [][4]any{
			{"SECRET_KEY", "super-secret-token", true, false},
		})

		resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}

		redacted := deploy.Redact(resolved.Environ, resolved.SecretKeys())
		m := toMap(redacted)
		if m["SECRET_KEY"] == "super-secret-token" {
			t.Fatal("secret value appeared unmasked in redacted output")
		}
		if m["SECRET_KEY"] == "" {
			t.Fatal("redacted secret key missing entirely from output")
		}
		if strings.Contains(strings.Join(redacted, "\n"), "super-secret-token") {
			t.Fatal("plaintext secret present in redacted env")
		}
		// PUBLIC_VAR comes from os.Environ (platform baseline), so it is NOT in
		// secretKeys and must be preserved verbatim.
		if m["PUBLIC_VAR"] != "hello" {
			t.Fatalf("non-secret (platform) value should be preserved, got %q", m["PUBLIC_VAR"])
		}
	})

	t.Run("redact_with_encryption_key_still_masks", func(t *testing.T) {
		// Encryption-at-rest is orthogonal to redaction-in-logs. Even with a
		// configured key (values decrypted at resolve time), the redacted view
		// must still mask them. The stored value is encrypted with the same key
		// the resolver decrypts with.
		encKey := strings.Repeat("b2", 32)
		keyBytes, decErr := envcryptoHexToBytes(encKey)
		if decErr != nil {
			t.Fatalf("parse encKey: %v", decErr)
		}
		encrypted, encErr := envcrypto.Encrypt(keyBytes, "hunter2-hunter2")
		if encErr != nil {
			t.Fatalf("encrypt test value: %v", encErr)
		}
		r := resolveEnvEncrypted(t, encKey, [][4]any{
			{"DB_PASSWORD", encrypted, true, false},
		})
		resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if toMap(resolved.Environ)["DB_PASSWORD"] != "hunter2-hunter2" {
			t.Fatalf("decryption path failed: %q", toMap(resolved.Environ)["DB_PASSWORD"])
		}
		redacted := deploy.Redact(resolved.Environ, resolved.SecretKeys())
		if strings.Contains(strings.Join(redacted, "\n"), "hunter2-hunter2") {
			t.Fatal("plaintext secret present in redacted env with encryption enabled")
		}
	})
}

// TestSecretNeverAppearsInBuildLog is the load-bearing security property for
// issue #41: a secret value must NEVER appear in build-log output. It runs the
// resolved build env through the same redacting-log path the build command
// uses and asserts the plaintext is absent from the captured log.
func TestSecretNeverAppearsInBuildLog(t *testing.T) {
	secretValue := "leak-me-and-you-are-fired-9f3a"
	r := resolveEnv(t, "", [][4]any{
		{"LEAKY_SECRET", secretValue, true, false},
	})

	resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Simulate the build path capturing a tool's stderr that echoes the env
	// (the realistic leak vector: a build script prints its own environment).
	rawStderr := strings.Join(resolved.Environ, "\n")
	logged := deploy.RedactLogText(rawStderr, resolved)

	if strings.Contains(logged, secretValue) {
		t.Fatalf("SECRET LEAKED into build log:\n%s", logged)
	}
}

func TestEnvResolverDecryptionFailureStopsResolution(t *testing.T) {
	key := strings.Repeat("ab", 32)
	resolver := resolveEnvEncrypted(t, key, [][4]any{{"BROKEN_SECRET", "not-ciphertext", false, true}})
	if _, err := resolver.Resolve(context.Background(), "site-x", deploy.ScopeRuntime); err == nil {
		t.Fatal("selected decryption failure was silently ignored")
	}
}
