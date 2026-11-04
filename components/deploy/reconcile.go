package deploy

import (
	"context"
	"strings"
	"time"
)

// defaultReconcileStaleAfter is how old an in-flight deployment must be before
// boot reconciliation treats it as orphaned. Kept conservative so a deployment
// started immediately before a restart is not killed by a slow boot.
const defaultReconcileStaleAfter = 30 * time.Minute

const (
	reconcileInFlightMessage = "reconciliation: orphaned by process restart"
	reconcileRunningMessage  = "reconciliation: running deployment has no port"
)

// reconcileOrphanedDeployments marks deployments left in flight by a previous
// process as failed. It runs once at boot, before new goroutines start, so any
// pending/building/deploying row older than the configured threshold is
// unrecoverable. A running row without a port is always unrecoverable,
// regardless of age.
//
// Every transition goes through TransitionState, so status_history and
// deploy.state_changed events are recorded. The error_message is persisted
// before the transition, mirroring failDeployment, so a concurrent reader never
// observes status=failed with an empty reason.
//
// The function is idempotent: once a row is failed it no longer matches the
// candidate query. It returns the number of rows reconciled.
func (d *Deploy) reconcileOrphanedDeployments(ctx context.Context) int {
	threshold := d.reconcileStaleAfter
	if threshold <= 0 {
		threshold = defaultReconcileStaleAfter
	}
	cutoff := time.Now().UTC().Add(-threshold)

	rows, err := d.db.QueryContext(ctx,
		`SELECT id, status, COALESCE(created_at, ''), COALESCE(port, 0)
		 FROM deployments
		 WHERE status IN ('pending', 'building', 'deploying', 'draining')
		    OR (status = 'running' AND COALESCE(port, 0) <= 0)`)
	if err != nil {
		d.logger.Warn("reconcile orphaned deployments: query", "error", err)
		return 0
	}

	type candidate struct {
		id     string
		reason string
	}
	var candidates []candidate
	for rows.Next() {
		var id, status, createdAt string
		var port int
		if err := rows.Scan(&id, &status, &createdAt, &port); err != nil {
			continue
		}
		if status == string(StateRunning) {
			// port <= 0 is a broken running row whether or not it is fresh.
			candidates = append(candidates, candidate{id: id, reason: reconcileRunningMessage})
			continue
		}
		if isStaleDeploymentTime(createdAt, cutoff) {
			candidates = append(candidates, candidate{id: id, reason: reconcileInFlightMessage})
		}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		d.logger.Warn("reconcile orphaned deployments: iterate", "error", err)
		return 0
	}

	reconciled := 0
	for _, c := range candidates {
		d.mu.Lock()
		_, err := d.db.ExecContext(ctx,
			"UPDATE deployments SET error_message = ? WHERE id = ?", c.reason, c.id)
		d.mu.Unlock()
		if err != nil {
			d.logger.Warn("reconcile orphaned deployments: set error_message", "id", c.id, "error", err)
			continue
		}
		if err := d.TransitionState(ctx, c.id, string(StateFailed)); err != nil {
			d.logger.Warn("reconcile orphaned deployments: transition failed", "id", c.id, "error", err)
			continue
		}
		reconciled++
	}

	d.logger.Info("reconciled orphaned deployments", "count", reconciled)
	return reconciled
}

// isStaleDeploymentTime reports whether a persisted created_at is older than
// cutoff. Unknown or unparsable timestamps are treated as stale so a corrupt
// row cannot pin a site in a non-terminal state forever.
func isStaleDeploymentTime(createdAt string, cutoff time.Time) bool {
	ts, ok := parseDeploymentTime(createdAt)
	if !ok {
		return true
	}
	return ts.Before(cutoff)
}

// parseDeploymentTime accepts both timestamp formats found in the deployments
// table: RFC3339 (written by the engine) and SQLite's datetime() default
// (written by the initial migration default).
func parseDeploymentTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
	} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}
