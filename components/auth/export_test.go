package auth

// ResetTestModeForTesting overrides the internal isTestMode flag.
// Only compiled into test binaries.
func ResetTestModeForTesting(val bool) {
	testModeOverride = &val
}

// CreateJWTForTesting mints a real access token identical to what the register/
// login handlers issue, so tests can drive JWT-authenticated handlers along the
// realistic production attack path. Used by the cross-tenant route matrix guard
// (see components/auth/route_matrix_test.go). Only compiled into test binaries.
func CreateJWTForTesting(a *Auth, userID int64, email, role string, orgID int64) (string, error) {
	return createJWT(userID, email, role, orgID, a.secret, a.accessExpiry)
}
