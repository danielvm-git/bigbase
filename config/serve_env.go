package config

import (
	"os"
	"strconv"
)

// FlagOrEnv returns flagVal when non-empty, otherwise the value of envKey from the environment.
func FlagOrEnv(flagVal, envKey string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv(envKey)
}

// FlagOrEnvBool resolves a boolean configuration value. When the environment
// variable envKey is set (any value), it is parsed as a boolean
// ("true", "1" → true; anything else → false). When the env var is absent,
// flagVal is used as the fallback.
//
// This differs from FlagOrEnv (string) in priority: the environment variable
// always wins when present because boolean flags have a non-zero default that
// makes it impossible to distinguish "user didn't pass the flag" from "user
// passed the default value."
func FlagOrEnvBool(flagVal bool, envKey string) bool {
	e := os.Getenv(envKey)
	if e == "" {
		return flagVal
	}
	b, err := strconv.ParseBool(e)
	if err != nil {
		return e == "1"
	}
	return b
}
