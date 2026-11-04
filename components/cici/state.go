package cici

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// RunStatus is the lifecycle state of a CICI pipeline run.
//
//	queued -> running -> success | failure | cancelled
//
// A queued run may also fail or be cancelled before execution starts.
// Terminal statuses never transition again.
type RunStatus string

const (
	// RunStatusQueued means the run was accepted but execution has not begun.
	RunStatusQueued RunStatus = "queued"
	// RunStatusRunning means a process is executing the run's steps.
	RunStatusRunning RunStatus = "running"
	// RunStatusSuccess means every step completed successfully.
	RunStatusSuccess RunStatus = "success"
	// RunStatusFailure means a step failed, the deadline expired, or the run
	// was orphaned by a process restart.
	RunStatusFailure RunStatus = "failure"
	// RunStatusCancelled means the run was cancelled before completion.
	RunStatusCancelled RunStatus = "cancelled"
)

// legacyPendingStatus is the status default used by older cici_runs schemas.
// It is treated as an alias for RunStatusQueued for backward compatibility.
const legacyPendingStatus = "pending"

// runTimeout bounds a single run's execution. It is persisted in
// cici_runs.deadline_at and re-applied from the database when execution starts.
const runTimeout = 10 * time.Minute

var (
	// ErrInvalidTransition is returned when a run attempts an illegal state change.
	ErrInvalidTransition = errors.New("invalid run status transition")
	// ErrInvalidRunStatus is returned when a status string is not recognized.
	ErrInvalidRunStatus = errors.New("invalid run status")
)

// TransitionError describes a rejected state change.
type TransitionError struct {
	From   RunStatus
	To     RunStatus
	Reason string
}

func (e *TransitionError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("invalid run status transition %q -> %q: %s", e.From, e.To, e.Reason)
	}
	return fmt.Sprintf("invalid run status transition %q -> %q", e.From, e.To)
}

// Unwrap makes errors.Is(err, ErrInvalidTransition) work for callers.
func (e *TransitionError) Unwrap() error { return ErrInvalidTransition }

// StatusError describes an unrecognized run status string.
type StatusError struct {
	Status string
}

func (e *StatusError) Error() string { return fmt.Sprintf("invalid run status %q", e.Status) }

// Unwrap makes errors.Is(err, ErrInvalidRunStatus) work for callers.
func (e *StatusError) Unwrap() error { return ErrInvalidRunStatus }

var runTransitions = map[RunStatus][]RunStatus{
	RunStatusQueued:    {RunStatusRunning, RunStatusFailure, RunStatusCancelled},
	RunStatusRunning:   {RunStatusSuccess, RunStatusFailure, RunStatusCancelled},
	RunStatusSuccess:   nil,
	RunStatusFailure:   nil,
	RunStatusCancelled: nil,
}

// NormalizeRunStatus maps the legacy "pending" status to "queued" and
// lowercases/trims the value. Unknown values are returned normalized but are
// not validated; use ParseRunStatus or ValidateStatus for that.
func NormalizeRunStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == legacyPendingStatus {
		return string(RunStatusQueued)
	}
	return normalized
}

// IsTerminal reports whether a status is final and can no longer transition.
func IsTerminal(status RunStatus) bool {
	switch status {
	case RunStatusSuccess, RunStatusFailure, RunStatusCancelled:
		return true
	default:
		return false
	}
}

func isKnownRunStatus(status RunStatus) bool {
	_, ok := runTransitions[status]
	return ok
}

// ValidateStatus reports whether status is a recognized run status.
func ValidateStatus(status RunStatus) error {
	if !isKnownRunStatus(status) {
		return &StatusError{Status: string(status)}
	}
	return nil
}

// ParseRunStatus validates a raw status string, treating legacy "pending" as
// "queued".
func ParseRunStatus(status string) (RunStatus, error) {
	parsed := RunStatus(NormalizeRunStatus(status))
	if err := ValidateStatus(parsed); err != nil {
		return "", err
	}
	return parsed, nil
}

// CanTransition reports whether moving from one run status to another is valid.
// Self-transitions are rejected.
func CanTransition(from, to RunStatus) bool {
	if from == to {
		return false
	}
	for _, next := range runTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// ValidateTransition returns a typed TransitionError when the transition is not
// allowed by the run state machine.
func ValidateTransition(from, to RunStatus) error {
	if !isKnownRunStatus(from) {
		return &TransitionError{From: from, To: to, Reason: "unknown source status"}
	}
	if !isKnownRunStatus(to) {
		return &TransitionError{From: from, To: to, Reason: "unknown target status"}
	}
	if from == to {
		return &TransitionError{From: from, To: to, Reason: "already in target status"}
	}
	if !CanTransition(from, to) {
		return &TransitionError{From: from, To: to}
	}
	return nil
}
