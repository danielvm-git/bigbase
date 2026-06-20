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

func TestBuildEnvSkipsPuppeteerDownload(t *testing.T) {
	env := deploy.BuildEnv("/opt/bigbase")
	found := false
	for _, e := range env {
		if e == "PUPPETEER_SKIP_DOWNLOAD=true" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("PUPPETEER_SKIP_DOWNLOAD=true not set in build env")
	}
}

func TestBuildEnvEmptyHomePreservesEnviron(t *testing.T) {
	if len(deploy.BuildEnv("")) == 0 {
		t.Fatal("expected non-empty environ")
	}
}
