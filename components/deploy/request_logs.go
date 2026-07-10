package deploy

import (
	"context"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (d *Deploy) RecordRequestLog(siteID, method, path string, status int, duration time.Duration) {
	if siteID == "" {
		return
	}

	id, err := kernel.GenerateID()
	if err != nil {
		return
	}

	// Record asynchronously to avoid blocking the request
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := d.db.ExecContext(ctx,
			`INSERT INTO site_request_logs (id, site_id, method, path, status, duration_ms, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, siteID, method, path, status, duration.Milliseconds(), time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			d.logger.Warn("record request log", "error", err, "site_id", siteID)
		}
	}()
}

func (d *Deploy) StartPruningTicker() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			d.PruneRequestLogs()
		}
	}()
}

func (d *Deploy) PruneRequestLogs() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := d.db.ExecContext(ctx,
		`DELETE FROM site_request_logs WHERE created_at < datetime('now', '-7 days')`)
	if err != nil {
		d.logger.Warn("prune request logs", "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		d.logger.Info("pruned request logs", "count", n)
	}
}
