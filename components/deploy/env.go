package deploy

import (
	"os"
	"path/filepath"
	"strings"
)

// BuildEnv returns process environment for npm/go build subprocesses.
// When home is non-empty, sets HOME and NPM_CONFIG_CACHE under that directory.
func BuildEnv(home string) []string {
	if home == "" {
		return os.Environ()
	}
	env := os.Environ()
	env = withoutEnvKey(env, "HOME")
	env = withoutEnvKey(env, "NPM_CONFIG_CACHE")
	return append(env,
		"HOME="+home,
		"NPM_CONFIG_CACHE="+filepath.Join(home, ".npm"),
		// Prevent Puppeteer from downloading Chrome during npm install.
		// Not needed for static site builds; frequently fails on VPS.
		"PUPPETEER_SKIP_DOWNLOAD=true",
		"PUPPETEER_SKIP_BROWSER_DOWNLOAD=true",
	)
}

func withoutEnvKey(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (d *Deploy) buildHomeDir() string {
	if d.buildHome != "" {
		return d.buildHome
	}
	return os.Getenv("BIGBASE_HOME")
}

// EnvResolver exposes the single env-resolution + redaction owner used by
// the build and runtime paths (issue #41). Callers must not mutate the
// returned resolver; it is shared.
func (d *Deploy) EnvResolver() *EnvResolver {
	return d.envResolver
}

// manifestEnv returns the manifest [env] map, or nil when no manifest is
// present. Safe to pass directly to ResolveOptions.ManifestEnv.
func manifestEnv(m *Manifest) map[string]string {
	if m == nil {
		return nil
	}
	return m.Env
}

// buildCmdEnv is the platform build-time baseline (layer 1 of Resolve's
// precedence): os.Environ plus build home/cache tuning. Site env vars are
// layered on top via EnvResolver.Resolve.
func (d *Deploy) buildCmdEnv() []string {
	return BuildEnv(d.buildHomeDir())
}
