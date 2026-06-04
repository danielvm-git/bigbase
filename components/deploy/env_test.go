package deploy_test

import (
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
)

func TestBuildEnvSetsHomeAndNpmCache(t *testing.T) {
	env := deploy.BuildEnv("/opt/bigbase")
	var home, cache string
	for _, e := range env {
		if strings.HasPrefix(e, "HOME=") {
			home = e
		}
		if strings.HasPrefix(e, "NPM_CONFIG_CACHE=") {
			cache = e
		}
	}
	if home != "HOME=/opt/bigbase" {
		t.Fatalf("HOME: got %q", home)
	}
	if cache != "NPM_CONFIG_CACHE=/opt/bigbase/.npm" {
		t.Fatalf("NPM_CONFIG_CACHE: got %q", cache)
	}
}

func TestBuildEnvEmptyHomePreservesEnviron(t *testing.T) {
	if len(deploy.BuildEnv("")) == 0 {
		t.Fatal("expected non-empty environ")
	}
}
