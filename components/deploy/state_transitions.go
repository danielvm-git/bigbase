package deploy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// ErrTransitionConflict is returned when a competing writer changed the
// deployment status while a transition was being applied, and the retry budget
// was exhausted.
var ErrTransitionConflict = errors.New("deployment status changed concurrently")

// ErrInvalidTransition is returned when a deployment attempts an illegal state change.
var ErrInvalidTransition = errors.New("invalid deployment status transition")

const transitionMaxAttempts = 4

// TransitionState applies a validated state transition atomically.
//
// It reads the current status, validates the transition, then performs a
// compare-and-swap UPDATE (WHERE id = ? AND status = ?) that must affect
// exactly one row. The status is re-read between attempts, so a lost race
// either resolves to the requested state (idempotent no-op) or fails
// deterministically. status_history and events are only written after the
// CAS succeeds, so two concurrent callers can never record two outcomes.
func (d *Deploy) TransitionState(ctx context.Context, id, newStatus string) error {
	if d.sm == nil {
		return fmt.Errorf("state machine not initialized")
	}

	for attempt := 0; attempt < transitionMaxAttempts; attempt++ {
		// Query current status outside the lock. The CAS below rejects a write
		// if another goroutine changed it between this read and the update.
		var current string
		if err := d.db.QueryRowContext(ctx,
			"SELECT status FROM deployments WHERE id = ?", id).Scan(&current); err != nil {
			return fmt.Errorf("query current status: %w", err)
		}

		// Idempotent: same status is a no-op
		if current == newStatus {
			return nil
		}

		// Lenient mode: legacy rows may carry a status the machine no longer
		// declares (e.g. "replaced"). Normal flows must always take the
		// validated path below.
		if !d.sm.IsValidState(current) {
			d.logger.Warn("transition from unknown state (legacy row)",
				"id", id, "from", current, "to", newStatus)
		} else if !d.sm.CanTransition(current, newStatus) {
			allowed := d.sm.ValidTransitions(current)
			return fmt.Errorf("%w: %s → %s (allowed: %v)", ErrInvalidTransition, current, newStatus, allowed)
		}

		// Snapshot build_log for finished states so a concurrent reader never
		// observes status=running/failed with a stale build_log (the
		// TestStaticSidecar_BuildAndStart flake: status and build_log were
		// written in two separate statements).
		var buildLog string
		if d.sm.IsTerminal(newStatus) {
			buildLog = strings.Join(d.getDeployLogs(id), "\n")
		}

		updated, err := d.compareAndSwapStatus(ctx, id, current, newStatus, buildLog)
		if err != nil {
			return err
		}
		if !updated {
			// Lost the race. Re-read on the next attempt: the deployment may
			// now already be in newStatus (idempotent) or in a state that
			// rejects this transition (deterministic error).
			continue
		}

		transition := StatusTransition{
			From:      current,
			To:        newStatus,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		// Serialize the status_history read-modify-write. Only a CAS winner
		// reaches this point, so history can never be appended twice.
		d.mu.Lock()
		d.appendStatusHistory(ctx, id, transition)
		d.mu.Unlock()

		// Emit event (nil-safe), outside the lock: handlers may call back.
		if d.eventBus != nil {
			siteID, orgID := d.deploymentOrg(ctx, id)
			_ = d.eventBus.Emit(kernel.Event{
				Name: "deploy.state_changed",
				Data: map[string]any{
					"deployment_id": id,
					"site_id":       siteID,
					"org_id":        orgID,
					"from_state":    current,
					"to_state":      newStatus,
					"timestamp":     transition.Timestamp,
				},
			}, &kernel.Context{})
			if newStatus == string(StateFailed) && current != string(StateFailed) {
				d.emitDeployFailed(ctx, id)
			}
		}

		return nil
	}

	return fmt.Errorf("%w: deployment %s", ErrTransitionConflict, id)
}

// compareAndSwapStatus updates status (and build_log when non-empty) only if
// the row is still in the expected source state. It reports whether the write
// applied; callers must treat false as a lost race.
func (d *Deploy) compareAndSwapStatus(ctx context.Context, id, from, to, buildLog string) (bool, error) {
	var (
		res sql.Result
		err error
	)
	if buildLog != "" {
		res, err = d.db.ExecContext(ctx,
			"UPDATE deployments SET status = ?, build_log = ? WHERE id = ? AND status = ?",
			to, buildLog, id, from)
	} else {
		res, err = d.db.ExecContext(ctx,
			"UPDATE deployments SET status = ? WHERE id = ? AND status = ?",
			to, id, from)
	}
	if err != nil {
		return false, fmt.Errorf("update deployment status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return affected == 1, nil
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
	if _, err := d.db.ExecContext(ctx,
		"UPDATE deployments SET status_history = ? WHERE id = ?", string(data), id); err != nil {
		d.logger.Warn("append status history failed", "deployment_id", id, "error", err)
	}
}
func (d *Deploy) updateStatus(id, status string) {
	_ = d.TransitionState(context.Background(), id, status)
}
