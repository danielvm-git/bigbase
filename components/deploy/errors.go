package deploy

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for domain-level error handling.
// These replace fragile string comparisons with errors.Is checks.
var (
	ErrRepoNotFound       = errors.New("repo not found")
	ErrDeploymentNotFound = errors.New("deployment not found")
)

// CodedError is a deploy failure with a stable machine code and operator hint.
type CodedError struct {
	Code    string
	Message string
	Hint    string
}

func (e *CodedError) Error() string {
	if e.Hint == "" {
		return fmt.Sprintf("%s (%s)", e.Message, e.Code)
	}
	return fmt.Sprintf("%s (%s): %s", e.Message, e.Code, e.Hint)
}

func codedErr(code, message, hint string) error {
	return &CodedError{Code: code, Message: message, Hint: hint}
}

// isDuplicateColumnError reports whether err is a "duplicate column" error
// from either SQLite or PostgreSQL, used during migration idempotency checks.
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column")
}
