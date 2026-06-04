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

func (d *Deploy) buildCmdEnv() []string {
	return BuildEnv(d.buildHomeDir())
}
