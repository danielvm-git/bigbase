package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// OrgUsageDay is one daily bucket of usage data for an org.
type OrgUsageDay struct {
	Date     string `json:"date"`
	Requests int64  `json:"requests"`
	Bytes    int64  `json:"bytes"`
}

// OrgUsageSummary holds the 30-day breakdown for an org.
type OrgUsageSummary struct {
	OrgID         int64         `json:"org_id"`
	TotalRequests int64         `json:"total_requests"`
	TotalBytes    int64         `json:"total_bytes"`
	Days          []OrgUsageDay `json:"days"`
}

// migrateOrgUsageTable creates the org_usage table if it doesn't exist.
func (m *Monitoring) migrateOrgUsageTable() error {
	return m.db.Migrate(`CREATE TABLE IF NOT EXISTS org_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		requests INTEGER NOT NULL DEFAULT 0,
		bytes INTEGER NOT NULL DEFAULT 0,
		UNIQUE(org_id, date)
	)`)
}

// RecordOrgRequest increments the daily request counter and bytes for an org.
// It upserts into the org_usage table using today's date as the bucket key.
func (m *Monitoring) RecordOrgRequest(orgID int64, bytesTransferred int64) {
	if m.db == nil {
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, `
		INSERT INTO org_usage (org_id, date, requests, bytes)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(org_id, date)
		DO UPDATE SET
			requests = requests + 1,
			bytes = bytes + excluded.bytes`,
		orgID, today, bytesTransferred)
	if err != nil {
		m.logger.Error("record org request", "org_id", orgID, "error", err)
	}
}

// GetOrgUsage returns the last 30 days of usage for an org.
func (m *Monitoring) GetOrgUsage(ctx context.Context, orgID int64) (*OrgUsageSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cutoff := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	rows, err := m.db.QueryContext(ctx,
		`SELECT date, requests, bytes FROM org_usage
		 WHERE org_id = ? AND date >= ?
		 ORDER BY date`,
		orgID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query org usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	summary := &OrgUsageSummary{
		OrgID: orgID,
		Days:  make([]OrgUsageDay, 0),
	}
	for rows.Next() {
		var day OrgUsageDay
		if err := rows.Scan(&day.Date, &day.Requests, &day.Bytes); err != nil {
			return nil, fmt.Errorf("scan org usage: %w", err)
		}
		summary.TotalRequests += day.Requests
		summary.TotalBytes += day.Bytes
		summary.Days = append(summary.Days, day)
	}
	return summary, rows.Err()
}

// handleOrgUsage serves GET /api/orgs/{id}/usage.
func (m *Monitoring) handleOrgUsage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	orgID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || orgID <= 0 {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid org id"})
		return
	}

	summary, err := m.GetOrgUsage(r.Context(), orgID)
	if err != nil {
		m.logger.Error("get org usage", "org_id", orgID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": summary})
}
