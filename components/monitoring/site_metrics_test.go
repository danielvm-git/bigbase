package monitoring

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// noopLogger is a throwaway logger for unit tests.
type siteTestLogger struct{}

func (siteTestLogger) Info(string, ...any)  {}
func (siteTestLogger) Warn(string, ...any)  {}
func (siteTestLogger) Error(string, ...any) {}
func (siteTestLogger) Debug(string, ...any) {}

// TestMetricValueUnknownIsDistinguishable verifies that metricValue no longer
// silently returns 0 for an unknown/typo'd metric name. The boolean return must
// be false so the checker can fail closed and log loudly instead of firing
// forever (lt) or never (gt). (Issue #178, gap #3.)
func TestMetricValueUnknownIsDistinguishable(t *testing.T) {
	host := HostMetrics{CPUPercent: 10}
	sys := SystemMetrics{Goroutines: 5}

	// Known metrics must report ok=true.
	if v, ok := metricValue("cpu_percent", host, sys); !ok || v != 10 {
		t.Fatalf("cpu_percent: ok=%v v=%f", ok, v)
	}

	// Unknown / typo'd metric name must report ok=false (distinguishable from a
	// genuine 0 reading).
	if _, ok := metricValue("site_upp", host, sys); ok {
		t.Fatal("typo'd metric name 'site_upp' must return ok=false, not silently 0")
	}
	if _, ok := metricValue("definitely_not_a_metric", host, sys); ok {
		t.Fatal("unknown metric must return ok=false")
	}
}

// TestSiteUpMetricEvaluatesCorrectly verifies that the site_up metric (1=up,
// 0=down) is resolved from a pluggable site-status provider and can drive a
// threshold rule. (Issue #178, gap #1.)
func TestSiteUpMetricEvaluatesCorrectly(t *testing.T) {
	host := HostMetrics{}
	sys := SystemMetrics{}

	// No provider wired → unknown (so a rule never silently fires).
	if _, ok := metricValue("site_up", host, sys); ok {
		t.Fatal("site_up with no provider should be unknown (ok=false)")
	}

	// Wire a provider that reports the site as UP.
	m := &Monitoring{logger: siteTestLogger{}}
	m.SetSiteStatusProvider(func() SiteStatus {
		return SiteStatus{Up: true, HTTPStatus: 200}
	})

	if v, ok := m.metricValueWithSite("site_up", host, sys); !ok {
		t.Fatal("site_up with provider should be known (ok=true)")
	} else if v != 1 {
		t.Fatalf("site_up when UP should be 1, got %f", v)
	}

	if v, ok := m.metricValueWithSite("site_http_status", host, sys); !ok {
		t.Fatal("site_http_status with provider should be known")
	} else if v != 200 {
		t.Fatalf("site_http_status should be 200, got %f", v)
	}

	// Provider reports DOWN.
	m.SetSiteStatusProvider(func() SiteStatus {
		return SiteStatus{Up: false, HTTPStatus: 502}
	})
	if v, ok := m.metricValueWithSite("site_up", host, sys); !ok || v != 0 {
		t.Fatalf("site_up when DOWN should be 0, got ok=%v v=%f", ok, v)
	}
	if v, ok := m.metricValueWithSite("site_http_status", host, sys); !ok || v != 502 {
		t.Fatalf("site_http_status should be 502, got ok=%v v=%f", ok, v)
	}
}

// fakeNotifier records Notify calls so tests can assert the alert.triggered
// event actually reaches a notifier.
type fakeNotifier struct {
	mu       sync.Mutex
	alerts   []AlertEvent
	notifyFn func(AlertEvent) error
}

func (f *fakeNotifier) NotifyAlert(ctx context.Context, ev AlertEvent) error {
	f.mu.Lock()
	f.alerts = append(f.alerts, ev)
	f.mu.Unlock()
	if f.notifyFn != nil {
		return f.notifyFn(ev)
	}
	return nil
}

func (f *fakeNotifier) Alerts() []AlertEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]AlertEvent, len(f.alerts))
	copy(out, f.alerts)
	return out
}

// TestAlertTriggeredReachesNotifier verifies that emitting an alert.triggered
// event on the bus delivers it to a registered AlertNotifier (the missing
// delivery path). (Issue #178, gap #2.)
func TestAlertTriggeredReachesNotifier(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:", Logger: siteTestLogger{}})
	m := New(Options{DB: d, Logger: siteTestLogger{}})
	k := kernel.New(siteTestLogger{})
	k.Register(d)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	notifier := &fakeNotifier{}
	m.SetAlertNotifier(notifier)

	// Emit on the bus exactly as alert_checker.emitAlertTriggered does.
	m.emitAlertTriggered("rule-1", "site down", "site_up", 0, 1, "lt", "inc-1")

	// The subscriber launches a goroutine; wait briefly for delivery.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(notifier.Alerts()) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	got := notifier.Alerts()
	if len(got) != 1 {
		t.Fatalf("expected 1 notifier call, got %d", len(got))
	}
	if got[0].AlertID != "rule-1" || got[0].IncidentID != "inc-1" || got[0].Metric != "site_up" {
		t.Fatalf("notifier received wrong event: %+v", got[0])
	}
}
