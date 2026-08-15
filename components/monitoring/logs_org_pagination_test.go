package monitoring_test

// e86s01 (keyset pagination) + e86s03 (org-scoped logs) coverage.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/kernel"
)

func setupLogsMonitoring(t *testing.T) (http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	m := monitoring.New(monitoring.Options{DB: d, Logger: logger})
	k.Register(d)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return m.Handler(), d
}

func withOrgLogs(orgID int64, r *http.Request) *http.Request {
	return r.WithContext(kernel.WithOrgID(r.Context(), orgID))
}

// seedLog inserts a monitoring_logs row directly with a controlled, ordered id
// (zero-padded so string DESC order == insertion order — the keyset invariant).
func seedLog(t *testing.T, d *db.DB, seq int, orgID int64, level, message string) string {
	t.Helper()
	id := fmt.Sprintf("%015d", seq)
	if _, err := d.ExecContext(context.Background(),
		"INSERT INTO monitoring_logs (id, level, message, org_id, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
		id, level, message, orgID); err != nil {
		t.Fatalf("seed log seq=%d: %v", seq, err)
	}
	return id
}

func TestMigrate(t *testing.T) {
	_, d := setupLogsMonitoring(t)
	rows, err := d.QueryContext(context.Background(), "PRAGMA table_info(monitoring_logs)")
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer func() { _ = rows.Close() }()
	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == "org_id" {
			found = true
		}
	}
	if !found {
		t.Error("monitoring_logs must have an org_id column after migration")
	}
}

func TestLogCreateOrg(t *testing.T) {
	h, d := setupLogsMonitoring(t)

	// No org context → 401.
	body, _ := json.Marshal(map[string]string{"level": "info", "message": "no-org"})
	req := httptest.NewRequest("POST", "/api/monitoring/logs", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("create without org: got %d, want 401", w.Code)
	}

	// With org context → 201, and the row carries that org_id.
	body2, _ := json.Marshal(map[string]string{"level": "error", "message": "boom"})
	req2 := httptest.NewRequest("POST", "/api/monitoring/logs", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, withOrgLogs(7, req2))
	if w2.Code != http.StatusCreated {
		t.Fatalf("create with org: got %d, want 201: %s", w2.Code, w2.Body.String())
	}
	var created struct{ ID string }
	_ = json.Unmarshal(w2.Body.Bytes(), &created)
	var gotOrg int64
	if err := d.QueryRowContext(context.Background(),
		"SELECT org_id FROM monitoring_logs WHERE id = ?", created.ID).Scan(&gotOrg); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if gotOrg != 7 {
		t.Errorf("stored org_id = %d, want 7", gotOrg)
	}
}

type logSearchResp struct {
	Data       []monitoring.LogEntry `json:"data"`
	NextCursor string                `json:"next_cursor"`
	HasMore    bool                  `json:"has_more"`
}

func searchLogs(t *testing.T, h http.Handler, orgID int64, query string) logSearchResp {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/monitoring/logs"+query, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, withOrgLogs(orgID, req))
	if w.Code != http.StatusOK {
		t.Fatalf("search %q: got %d, want 200: %s", query, w.Code, w.Body.String())
	}
	var resp logSearchResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal search resp: %v (%s)", err, w.Body.String())
	}
	return resp
}

func TestOrgIsolationLogs(t *testing.T) {
	h, d := setupLogsMonitoring(t)
	idA := seedLog(t, d, 1, 1, "info", "org1-secret")
	seedLog(t, d, 2, 2, "info", "org2-data")
	seedLog(t, d, 3, 0, "info", "nullorg-leftover") // org_id=0 (legacy) invisible to tenants

	// Org 2 search sees only its own row.
	resp := searchLogs(t, h, 2, "")
	if len(resp.Data) != 1 || resp.Data[0].Message != "org2-data" {
		t.Fatalf("org2 search = %+v, want exactly org2-data", resp.Data)
	}

	// Org 2 cannot read org 1's log by id.
	req := httptest.NewRequest("GET", "/api/monitoring/logs/"+idA, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, withOrgLogs(2, req))
	if w.Code != http.StatusNotFound {
		t.Errorf("cross-org byID: got %d, want 404", w.Code)
	}

	// Org 1 can read its own log by id.
	req2 := httptest.NewRequest("GET", "/api/monitoring/logs/"+idA, nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, withOrgLogs(1, req2))
	if w2.Code != http.StatusOK {
		t.Errorf("same-org byID: got %d, want 200", w2.Code)
	}
}

func TestLogPagination(t *testing.T) {
	h, d := setupLogsMonitoring(t)
	for i := 0; i < 250; i++ {
		seedLog(t, d, i, 5, "info", fmt.Sprintf("msg-%03d", i))
	}

	// Page 1: default limit 100.
	p1 := searchLogs(t, h, 5, "?limit=100")
	if len(p1.Data) != 100 || !p1.HasMore || p1.NextCursor == "" {
		t.Fatalf("page1: len=%d has_more=%v cursor=%q, want 100/true/nonempty", len(p1.Data), p1.HasMore, p1.NextCursor)
	}
	// Page 2.
	p2 := searchLogs(t, h, 5, "?limit=100&cursor="+p1.NextCursor)
	if len(p2.Data) != 100 || !p2.HasMore {
		t.Fatalf("page2: len=%d has_more=%v, want 100/true", len(p2.Data), p2.HasMore)
	}
	// Page 3 (last).
	p3 := searchLogs(t, h, 5, "?limit=100&cursor="+p2.NextCursor)
	if len(p3.Data) != 50 || p3.HasMore {
		t.Fatalf("page3: len=%d has_more=%v, want 50/false", len(p3.Data), p3.HasMore)
	}

	// Newest-first ordering across the boundary.
	if p1.Data[0].Message != "msg-249" {
		t.Errorf("page1 first = %q, want msg-249 (newest first)", p1.Data[0].Message)
	}
}

func TestLogSearch(t *testing.T) {
	h, d := setupLogsMonitoring(t)
	seedLog(t, d, 1, 5, "info", "alpha")
	seedLog(t, d, 2, 5, "info", "beta")
	seedLog(t, d, 3, 5, "info", "alphabet")
	seedLog(t, d, 4, 9, "info", "alpha-other-org") // different org, must not appear

	resp := searchLogs(t, h, 5, "?q=alpha")
	if len(resp.Data) != 2 {
		t.Fatalf("q=alpha for org5: got %d rows, want 2 (%+v)", len(resp.Data), resp.Data)
	}
	for _, l := range resp.Data {
		if l.Message == "alpha-other-org" {
			t.Error("search leaked another org's row")
		}
	}

	// limit clamp: limit=0 falls back to default (still returns rows).
	if got := searchLogs(t, h, 5, "?limit=0"); len(got.Data) == 0 {
		t.Error("limit=0 should clamp to default, not return empty")
	}
}
