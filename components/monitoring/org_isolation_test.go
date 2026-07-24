package monitoring_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/kernel"
)

func TestMonitoringAlertOrgIsolation(t *testing.T) {
	t.Run("alerts_scoped_by_org_id", func(t *testing.T) {
		// Setup: single DB, two handlers for different orgs
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

		base := m.Handler()
		withOrg := func(orgID int64, r *http.Request) *http.Request {
			return r.WithContext(kernel.WithOrgID(r.Context(), orgID))
		}

		// Org 1 creates an alert
		alert1 := map[string]any{
			"name":      "org1-high-errors",
			"metric":    "error_rate",
			"threshold": 5.0,
			"operator":  "gt",
			"enabled":   true,
		}
		body1, _ := json.Marshal(alert1)
		req1 := httptest.NewRequest("POST", "/api/monitoring/alerts", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		base.ServeHTTP(w1, withOrg(1, req1))
		if w1.Code != http.StatusCreated {
			t.Fatalf("org1 create: expected 201, got %d: %s", w1.Code, w1.Body.String())
		}

		// Org 2 creates an alert
		alert2 := map[string]any{
			"name":      "org2-high-latency",
			"metric":    "latency_p99",
			"threshold": 500.0,
			"operator":  "gt",
			"enabled":   true,
		}
		body2, _ := json.Marshal(alert2)
		req2 := httptest.NewRequest("POST", "/api/monitoring/alerts", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		base.ServeHTTP(w2, withOrg(2, req2))
		if w2.Code != http.StatusCreated {
			t.Fatalf("org2 create: expected 201, got %d: %s", w2.Code, w2.Body.String())
		}

		// Org 1 lists alerts — should only see org1's alert
		listReq1 := httptest.NewRequest("GET", "/api/monitoring/alerts", nil)
		lw1 := httptest.NewRecorder()
		base.ServeHTTP(lw1, withOrg(1, listReq1))
		if lw1.Code != http.StatusOK {
			t.Fatalf("org1 list: expected 200, got %d: %s", lw1.Code, lw1.Body.String())
		}
		var result1 map[string]any
		if err := json.NewDecoder(lw1.Body).Decode(&result1); err != nil {
			t.Fatalf("decode: %v", err)
		}
		alerts1, ok := result1["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got: %v", result1)
		}
		if len(alerts1) != 1 {
			t.Fatalf("org1 should see 1 alert, got %d", len(alerts1))
		}
		alert1Data := alerts1[0].(map[string]any)
		if alert1Data["name"] != "org1-high-errors" {
			t.Fatalf("org1 should see 'org1-high-errors', got '%v'", alert1Data["name"])
		}

		// Org 2 lists alerts — should only see org2's alert
		listReq2 := httptest.NewRequest("GET", "/api/monitoring/alerts", nil)
		lw2 := httptest.NewRecorder()
		base.ServeHTTP(lw2, withOrg(2, listReq2))
		if lw2.Code != http.StatusOK {
			t.Fatalf("org2 list: expected 200, got %d: %s", lw2.Code, lw2.Body.String())
		}
		var result2 map[string]any
		if err := json.NewDecoder(lw2.Body).Decode(&result2); err != nil {
			t.Fatalf("decode: %v", err)
		}
		alerts2, ok := result2["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got: %v", result2)
		}
		if len(alerts2) != 1 {
			t.Fatalf("org2 should see 1 alert, got %d", len(alerts2))
		}
		alert2Data := alerts2[0].(map[string]any)
		if alert2Data["name"] != "org2-high-latency" {
			t.Fatalf("org2 should see 'org2-high-latency', got '%v'", alert2Data["name"])
		}
	})

	t.Run("alert_create_requires_org_id", func(t *testing.T) {
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

		base := m.Handler()

		// Request WITHOUT org_id in context — should fail
		alert := map[string]any{
			"name":      "no-org-alert",
			"metric":    "cpu",
			"threshold": 90.0,
			"operator":  "gt",
			"enabled":   true,
		}
		body, _ := json.Marshal(alert)
		req := httptest.NewRequest("POST", "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		base.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without org_id, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestMonitoringIncidentOrgIsolation(t *testing.T) {
	t.Run("incidents_scoped_by_org_id", func(t *testing.T) {
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

		base := m.Handler()
		withOrg := func(orgID int64, r *http.Request) *http.Request {
			return r.WithContext(kernel.WithOrgID(r.Context(), orgID))
		}

		// Manually insert incidents for different orgs
		_, err := d.ExecContext(context.Background(),
			`INSERT INTO monitoring_alert_incidents (id, rule_id, metric, value, threshold, operator, org_id)
			 VALUES ('inc-1', 'rule-1', 'cpu', 95.0, 90.0, 'gt', 1)`)
		if err != nil {
			t.Fatalf("insert incident org1: %v", err)
		}
		_, err = d.ExecContext(context.Background(),
			`INSERT INTO monitoring_alert_incidents (id, rule_id, metric, value, threshold, operator, org_id)
			 VALUES ('inc-2', 'rule-2', 'memory', 85.0, 80.0, 'gt', 2)`)
		if err != nil {
			t.Fatalf("insert incident org2: %v", err)
		}

		// Org 1 lists incidents — should only see inc-1
		listReq1 := httptest.NewRequest("GET", "/api/monitoring/incidents", nil)
		lw1 := httptest.NewRecorder()
		base.ServeHTTP(lw1, withOrg(1, listReq1))
		if lw1.Code != http.StatusOK {
			t.Fatalf("org1 list: expected 200, got %d: %s", lw1.Code, lw1.Body.String())
		}
		var result1 map[string]any
		if err := json.NewDecoder(lw1.Body).Decode(&result1); err != nil {
			t.Fatalf("decode: %v", err)
		}
		incidents1, ok := result1["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got: %v", result1)
		}
		if len(incidents1) != 1 {
			t.Fatalf("org1 should see 1 incident, got %d", len(incidents1))
		}
		inc1Data := incidents1[0].(map[string]any)
		if inc1Data["id"] != "inc-1" {
			t.Fatalf("org1 should see 'inc-1', got '%v'", inc1Data["id"])
		}

		// Org 2 lists incidents — should only see inc-2
		listReq2 := httptest.NewRequest("GET", "/api/monitoring/incidents", nil)
		lw2 := httptest.NewRecorder()
		base.ServeHTTP(lw2, withOrg(2, listReq2))
		if lw2.Code != http.StatusOK {
			t.Fatalf("org2 list: expected 200, got %d: %s", lw2.Code, lw2.Body.String())
		}
		var result2 map[string]any
		if err := json.NewDecoder(lw2.Body).Decode(&result2); err != nil {
			t.Fatalf("decode: %v", err)
		}
		incidents2, ok := result2["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got: %v", result2)
		}
		if len(incidents2) != 1 {
			t.Fatalf("org2 should see 1 incident, got %d", len(incidents2))
		}
		inc2Data := incidents2[0].(map[string]any)
		if inc2Data["id"] != "inc-2" {
			t.Fatalf("org2 should see 'inc-2', got '%v'", inc2Data["id"])
		}
	})

	t.Run("incidents_require_org_id", func(t *testing.T) {
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

		base := m.Handler()

		// Request WITHOUT org_id — should fail
		req := httptest.NewRequest("GET", "/api/monitoring/incidents", nil)
		w := httptest.NewRecorder()
		base.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without org_id, got %d: %s", w.Code, w.Body.String())
		}
	})
}
