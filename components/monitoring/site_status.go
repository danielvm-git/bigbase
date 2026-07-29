package monitoring

import (
	"context"
	"fmt"
	"time"
)

// SiteStatus is a single liveness reading for a hosted site's public URL.
// It is produced by a SiteStatusProvider (e.g. a scheduled HTTP probe) and
// feeds the site_up / site_http_status alert metrics. (Issue #178, gap #1.)
type SiteStatus struct {
	// HasReading is false until at least one probe has completed. Before the
	// first reading, site metrics are treated as unknown (ok=false) so an
	// alert rule cannot fire on an unset value.
	HasReading bool
	// Up is true when the site responded with an acceptable HTTP status.
	Up bool
	// HTTPStatus is the last observed HTTP response code (0 if unknown).
	HTTPStatus int
}

// SiteStatusProvider returns the latest known site availability. It must be
// cheap and non-blocking; a background probe loop is expected to update the
// reading the provider closes over.
//
// TODO(issue #178): a scheduled HTTP probe of every hosted site's public URL
// should install the provider via Monitoring.SetSiteStatusProvider. The metric
// plumbing + alert wiring landed here is independent of HOW readings are
// produced, so the active probe scheduler can be added without touching the
// alert checker. Until that loop exists, site_up/site_http_status remain
// "unknown" (ok=false) and rules referencing them fail closed rather than
// firing on stale/zero data.
type SiteStatusProvider func() SiteStatus

// SetSiteStatusProvider installs the function used to resolve the site_up and
// site_http_status metrics. Pass nil to disable site metrics (they will then
// be reported as unknown by metricValueWithSite).
func (m *Monitoring) SetSiteStatusProvider(p SiteStatusProvider) {
	if m == nil {
		return
	}
	m.siteStatus = p
}

// AlertEvent is the payload delivered to an AlertNotifier when an alert rule
// breaches its threshold for the configured duration. (Issue #178, gap #2.)
type AlertEvent struct {
	AlertID    string  `json:"alert_id"`
	IncidentID string  `json:"incident_id"`
	Name       string  `json:"name"`
	Metric     string  `json:"metric"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Operator   string  `json:"operator"`
}

// AlertNotifier delivers an AlertEvent to a destination that reaches a human
// (email, Slack, PagerDuty, ...). The SMTP implementation lives in
// components/messaging. Implementations must be safe for concurrent use.
type AlertNotifier interface {
	NotifyAlert(ctx context.Context, ev AlertEvent) error
}

// SetAlertNotifier registers the notifier that receives alert.triggered
// events emitted from the bus. When set, every triggered alert is delivered
// in addition to being logged + investigated. Pass nil to disable delivery.
func (m *Monitoring) SetAlertNotifier(n AlertNotifier) {
	if m == nil {
		return
	}
	m.alertNotifier = n
}

// deliverAlert extracts the alert fields from an alert.triggered event payload
// and forwards them to the configured AlertNotifier. It is invoked from the
// alert.triggered subscriber. Errors are logged but never propagated, so a
// failing delivery channel cannot block or break the alert pipeline.
func (m *Monitoring) deliverAlert(data map[string]any) {
	if m == nil || m.alertNotifier == nil {
		return
	}
	ev := alertEventFromData(data)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := m.alertNotifier.NotifyAlert(ctx, ev); err != nil {
		m.logger.Error("deliver alert notification",
			"alert_id", ev.AlertID, "incident_id", ev.IncidentID, "error", err)
	}
}

// alertEventFromData coerces the loosely-typed event payload produced by
// emitAlertTriggered into a strongly-typed AlertEvent.
func alertEventFromData(data map[string]any) AlertEvent {
	toStr := func(k string) string {
		if v, ok := data[k].(string); ok {
			return v
		}
		return fmt.Sprintf("%v", data[k])
	}
	toFloat := func(k string) float64 {
		switch v := data[k].(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		default:
			return 0
		}
	}
	return AlertEvent{
		AlertID:    toStr("alert_id"),
		IncidentID: toStr("incident_id"),
		Name:       toStr("name"),
		Metric:     toStr("metric"),
		Value:      toFloat("value"),
		Threshold:  toFloat("threshold"),
		Operator:   toStr("operator"),
	}
}
