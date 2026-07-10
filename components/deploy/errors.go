package deploy

import (
	"errors"
	"strings"
)

// Sentinel errors for domain-level error handling.
// These replace fragile string comparisons with errors.Is checks.
var (
	ErrRepoNotFound       = errors.New("repo not found")
	ErrDeploymentNotFound = errors.New("deployment not found")
)

// isDuplicateColumnError reports whether err is a "duplicate column" error
// from either SQLite or PostgreSQL, used during migration idempotency checks.
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column")
}
