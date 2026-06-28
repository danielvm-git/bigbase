package auth

// ResetTestModeForTesting overrides the internal isTestMode flag.
// Only compiled into test binaries.
func ResetTestModeForTesting(val bool) {
	testModeOverride = &val
}
