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

func TestFlagOrEnvBool(t *testing.T) {
	t.Setenv("BOOL_TRUE", "true")
	t.Setenv("BOOL_ONE", "1")
	t.Setenv("BOOL_FALSE", "false")
	t.Setenv("BOOL_ZERO", "0")
	t.Setenv("BOOL_EMPTY", "")
	t.Setenv("BOOL_GARBAGE", "garbage")

	tests := []struct {
		name    string
		flagVal bool
		envKey  string
		want    bool
	}{
		{name: "env set to true overrides flag false", flagVal: false, envKey: "BOOL_TRUE", want: true},
		{name: "env set to 1 overrides flag false", flagVal: false, envKey: "BOOL_ONE", want: true},
		{name: "env set to false overrides flag true", flagVal: true, envKey: "BOOL_FALSE", want: false},
		{name: "env set to 0 overrides flag true", flagVal: true, envKey: "BOOL_ZERO", want: false},
		{name: "env absent falls back to flag true", flagVal: true, envKey: "BOOL_MISSING", want: true},
		{name: "env absent falls back to flag false", flagVal: false, envKey: "BOOL_MISSING", want: false},
		{name: "env garbage treated as false", flagVal: true, envKey: "BOOL_GARBAGE", want: false},
		{name: "env empty string falls back to flag", flagVal: true, envKey: "BOOL_EMPTY", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.FlagOrEnvBool(tt.flagVal, tt.envKey); got != tt.want {
				t.Fatalf("FlagOrEnvBool(%v, %q) = %v, want %v", tt.flagVal, tt.envKey, got, tt.want)
			}
		})
	}
}
