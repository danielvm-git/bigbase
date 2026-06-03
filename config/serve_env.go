package config

import "os"

// FlagOrEnv returns flagVal when non-empty, otherwise the value of envKey from the environment.
func FlagOrEnv(flagVal, envKey string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv(envKey)
}
