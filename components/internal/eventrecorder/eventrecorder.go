package eventrecorder

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const maxEvents = 5000

// RecordedEvent is a persisted bus event.
type RecordedEvent struct {
	ID        string         `json:"id"`
	Hook      string         `json:"hook"`
	SiteID    string         `json:"site_id,omitempty"`
	Data      map[string]any `json:"data"`
	Timestamp string         `json:"timestamp"`
}

// Filter selects recorded events.
type Filter struct {
	SiteID      string
	WindowStart time.Time
	WindowEnd   time.Time
	Hooks       []string
	Limit       int
}

// Recorder persists bus events with FIFO cap.
type Recorder struct {
	db kernel.DBer
}

func New(db kernel.DBer) (*Recorder, error) {
	r := &Recorder{db: db}
	if err := r.migrate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Recorder) migrate() error {
	if err := r.db.Migrate(`CREATE TABLE IF NOT EXISTS monitoring_events (
		id TEXT PRIMARY KEY,
		hook TEXT NOT NULL,
		site_id TEXT NOT NULL DEFAULT '',
		data TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate monitoring_events: %w", err)
	}
	_, _ = r.db.ExecContext(context.Background(),
		`CREATE INDEX IF NOT EXISTS idx_monitoring_events_site_created ON monitoring_events(site_id, created_at DESC)`)
	return nil
}

func (r *Recorder) Record(ctx context.Context, hook, siteID string, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	id, err := newID()
	if err != nil {
		return err
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO monitoring_events (id, hook, site_id, data, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, hook, siteID, string(payload), ts); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitoring_events`).Scan(&count); err != nil {
		return nil
	}
	if count > maxEvents {
		excess := count - maxEvents
		_, _ = r.db.ExecContext(ctx,
			`DELETE FROM monitoring_events WHERE id IN (
				SELECT id FROM monitoring_events ORDER BY created_at ASC LIMIT ?
			)`, excess)
	}
	return nil
}

func (r *Recorder) Query(ctx context.Context, f Filter) ([]RecordedEvent, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, hook, site_id, data, created_at FROM monitoring_events WHERE created_at >= ? AND created_at <= ?`
	args := []any{f.WindowStart.UTC().Format(time.RFC3339Nano), f.WindowEnd.UTC().Format(time.RFC3339Nano)}
	if f.SiteID != "" {
		query += ` AND site_id = ?`
		args = append(args, f.SiteID)
	}
	if len(f.Hooks) > 0 {
		placeholders := make([]string, len(f.Hooks))
		for i, h := range f.Hooks {
			placeholders[i] = "?"
			args = append(args, h)
		}
		query += ` AND hook IN (` + strings.Join(placeholders, ",") + `)`
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []RecordedEvent
	for rows.Next() {
		var ev RecordedEvent
		var dataJSON string
		if err := rows.Scan(&ev.ID, &ev.Hook, &ev.SiteID, &dataJSON, &ev.Timestamp); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(dataJSON), &ev.Data)
		out = append(out, ev)
	}
	return out, nil
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
