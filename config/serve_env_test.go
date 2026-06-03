package config_test

import (
	"os"
	"testing"

	"github.com/danielvm/bigbase/config"
)

func TestFlagOrEnv(t *testing.T) {
	t.Setenv("TEST_SERVE_ENV", "from-env")

	tests := []struct {
		name    string
		flagVal string
		envKey  string
		want    string
	}{
		{name: "flag wins", flagVal: "from-flag", envKey: "TEST_SERVE_ENV", want: "from-flag"},
		{name: "env fallback", flagVal: "", envKey: "TEST_SERVE_ENV", want: "from-env"},
		{name: "empty both", flagVal: "", envKey: "MISSING_ENV", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.FlagOrEnv(tt.flagVal, tt.envKey); got != tt.want {
				t.Fatalf("FlagOrEnv(%q, %q) = %q, want %q", tt.flagVal, tt.envKey, got, tt.want)
			}
		})
	}
}

func TestFlagOrEnvGitHubKeys(t *testing.T) {
	const key = "GITHUB_APP_ID"
	t.Setenv(key, "3950847")
	if got := config.FlagOrEnv("", key); got != "3950847" {
		t.Fatalf("got %q", got)
	}
	_ = os.Unsetenv(key)
}
