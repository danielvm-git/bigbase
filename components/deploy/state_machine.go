package deploy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// DeploymentState constants represent the valid states in the deployment lifecycle.
const (
	StatePending   = "pending"
	StateBuilding  = "building"
	StateDeploying = "deploying"
	StateLive      = "live"
	StateFailed    = "failed"
)

// runningState is the legacy name for a live deployment.
// The state machine accepts "running" as valid for backward compatibility.
const runningState = "running"

// StatusTransition records a single state change with a timestamp.
type StatusTransition struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Timestamp string `json:"timestamp"`
}

// stateMachine enforces valid deployment state transitions.
type stateMachine struct {
	valid map[string][]string
}

// NewStateMachine creates a stateMachine with the defined transition rules.
func NewStateMachine() *stateMachine {
	return &stateMachine{
		valid: map[string][]string{
			StatePending:   {StateBuilding, StateFailed},
			StateBuilding:  {StateDeploying, runningState, StateFailed},
			StateDeploying: {StateLive, runningState, StateFailed},
			StateLive:      {StateFailed},
			runningState:   {StateFailed},
		},
	}
}

// CanTransition returns true if moving from state `from` to state `to` is valid.
func (sm *stateMachine) CanTransition(from, to string) bool {
	allowed, ok := sm.valid[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// ValidTransitions returns the list of states that can be transitioned to from the given state.
func (sm *stateMachine) ValidTransitions(from string) []string {
	allowed, ok := sm.valid[from]
	if !ok {
		return nil
	}
	out := make([]string, len(allowed))
	copy(out, allowed)
	return out
}

// TransitionState validates and persists a deployment state transition.
// It returns an error if the transition is invalid or the deployment doesn't exist.
func (d *Deploy) TransitionState(ctx context.Context, id, newStatus string) error {
	// Read current status from DB
	var currentStatus string
	err := d.db.QueryRowContext(ctx,
		"SELECT status FROM deployments WHERE id = ?", id).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Validate transition
	if d.sm == nil {
		return fmt.Errorf("state machine not initialized")
	}
	if !d.sm.CanTransition(currentStatus, newStatus) {
		allowed := d.sm.ValidTransitions(currentStatus)
		return fmt.Errorf("invalid transition: %s -> %s (allowed: %v)",
			currentStatus, newStatus, allowed)
	}

	// Build transition record
	now := time.Now().UTC().Format(time.RFC3339)
	tr := StatusTransition{
		From:      currentStatus,
		To:        newStatus,
		Timestamp: now,
	}

	// Update status in DB
	_, err = d.db.ExecContext(ctx,
		"UPDATE deployments SET status = ? WHERE id = ?", newStatus, id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// Append transition to status_history
	_, _ = d.db.ExecContext(ctx,
		`UPDATE deployments SET status_history = json_insert(
			status_history, '$[#]',
			json_object('from', ?, 'to', ?, 'timestamp', ?)
		) WHERE id = ?`,
		tr.From, tr.To, tr.Timestamp, id)

	// Emit state change event on kernel event bus (nil-safe for tests without kernel)
	if d.eventBus != nil {
		_ = d.eventBus.Emit(kernel.Event{
			Name: "deploy.state_changed",
			Data: map[string]any{
				"deployment_id": id,
				"from_state":    currentStatus,
				"to_state":      newStatus,
				"timestamp":     tr.Timestamp,
			},
		}, nil)
	}

	// Persist build_log for terminal states (includes legacy "running")
	if newStatus == StateLive || newStatus == StateFailed || newStatus == runningState {
		lines := d.getDeployLogs(id)
		if len(lines) > 0 {
			_, _ = d.db.ExecContext(ctx,
				"UPDATE deployments SET build_log = ? WHERE id = ?",
				strings.Join(lines, "\n"), id)
		}
	}

	return nil
}
