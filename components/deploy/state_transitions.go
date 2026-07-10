package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (d *Deploy) TransitionState(ctx context.Context, id, newStatus string) error {
	if d.sm == nil {
		return fmt.Errorf("state machine not initialized")
	}

	// Query current status
	var current string
	if err := d.db.QueryRowContext(ctx,
		"SELECT status FROM deployments WHERE id = ?", id).Scan(&current); err != nil {
		return fmt.Errorf("query current status: %w", err)
	}

	// Idempotent: same status is a no-op
	if current == newStatus {
		return nil
	}

	// Lenient mode: if current status is unknown to the state machine, allow transition
	if !d.sm.IsValidState(current) {
		d.logger.Warn("transition from unknown state (lenient mode)",
			"id", id, "from", current, "to", newStatus)
	} else if !d.sm.CanTransition(current, newStatus) {
		allowed := d.sm.ValidTransitions(current)
		return fmt.Errorf("invalid transition: %s → %s (allowed: %v)", current, newStatus, allowed)
	}

	// Persist status change
	d.mu.Lock()
	defer d.mu.Unlock()
	_, _ = d.db.ExecContext(ctx,
		"UPDATE deployments SET status = ? WHERE id = ?", newStatus, id)

	// Append to status_history
	transition := StatusTransition{
		From:      current,
		To:        newStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	d.appendStatusHistory(ctx, id, transition)

	// Emit event (nil-safe)
	if d.eventBus != nil {
		_ = d.eventBus.Emit(kernel.Event{
			Name: "deploy.state_changed",
			Data: map[string]any{
				"deployment_id": id,
				"from_state":    current,
				"to_state":      newStatus,
				"timestamp":     transition.Timestamp,
			},
		}, &kernel.Context{})
		if newStatus == string(StateFailed) && current != string(StateFailed) {
			d.emitDeployFailed(ctx, id)
		}
	}

	// Persist build_log on terminal states
	if newStatus == string(StateRunning) || newStatus == string(StateFailed) {
		lines := d.getDeployLogs(id)
		if len(lines) > 0 {
			_, _ = d.db.ExecContext(ctx,
				"UPDATE deployments SET build_log = ? WHERE id = ?", strings.Join(lines, "\n"), id)
		}
	}

	return nil
}
func (d *Deploy) appendStatusHistory(ctx context.Context, id string, tr StatusTransition) {
	var current string
	if err := d.db.QueryRowContext(ctx,
		"SELECT status_history FROM deployments WHERE id = ?", id).Scan(&current); err != nil {
		return
	}
	if current == "" {
		current = "[]"
	}
	var history []StatusTransition
	if err := json.Unmarshal([]byte(current), &history); err != nil {
		history = nil
	}
	history = append(history, tr)
	data, err := json.Marshal(history)
	if err != nil {
		return
	}
	_, _ = d.db.ExecContext(ctx,
		"UPDATE deployments SET status_history = ? WHERE id = ?", string(data), id)
}
func (d *Deploy) updateStatus(id, status string) {
	_ = d.TransitionState(context.Background(), id, status)
}
