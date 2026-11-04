package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Non-terminal state sets. They mirror the authoritative state machines in
// components/deploy/state_machine.go (IsActive) and components/cici/state.go
// (queued/running, with legacy "pending" aliased to queued). Monitoring owns
// the observability copy because querying those tables must not import the
// deploy/cici packages, which would invert the component dependency graph.
var stateMachineSpecs = []stateMachineSpec{
	{
		Name:            "deployment",
		Table:           "deployments",
		TimestampColumn: "created_at",
		States:          []string{"pending", "building", "deploying", "draining", "running"},
	},
	{
		Name:            "cici_run",
		Table:           "cici_runs",
		TimestampColumn: "started_at",
		States:          []string{"queued", "running"},
		StatusAliases:   map[string]string{"pending": "queued"},
	},
}

const stateAgeQueryTimeout = 2 * time.Second

// stateMachineSpec describes one table whose rows have a non-terminal
// lifecycle state and a timestamp from which staleness can be derived.
type stateMachineSpec struct {
	Name            string
	Table           string
	TimestampColumn string
	States          []string
	// StatusAliases maps legacy status spellings onto a canonical state.
	StatusAliases map[string]string
}

// StateAgeReading is the staleness of one machine/state pair. AgeSeconds is
// the age of the oldest row in that state; Rows is the number of non-terminal
// rows observed.
type StateAgeReading struct {
	Machine    string  `json:"machine"`
	State      string  `json:"state"`
	AgeSeconds float64 `json:"age_seconds"`
	Rows       int     `json:"rows"`
	OldestID   string  `json:"oldest_id,omitempty"`
}

// StateAgeReport is the derived view of rows stuck in non-terminal states.
// Worst identifies the single oldest row across every machine/state, so
// operators can jump straight to the offender. Unavailable lists machines whose
// table could not be queried (for example, a deployment-free installation).
type StateAgeReport struct {
	Worst       *StateAgeReading  `json:"worst,omitempty"`
	ByState     []StateAgeReading `json:"by_state"`
	Unavailable []string          `json:"unavailable,omitempty"`
}

// now returns the injected clock when set, otherwise wall time. Injecting a
// clock keeps state-age assertions deterministic.
func (m *Monitoring) now() time.Time {
	if m != nil && m.nowFn != nil {
		return m.nowFn()
	}
	return time.Now()
}

// stateAgeReport queries every configured state machine and returns the
// staleness of each non-terminal state.
func (m *Monitoring) stateAgeReport(ctx context.Context) StateAgeReport {
	report := StateAgeReport{ByState: []StateAgeReading{}}
	if m == nil || m.db == nil {
		for _, spec := range stateMachineSpecs {
			report.Unavailable = append(report.Unavailable, spec.Name)
		}
		return report
	}

	now := m.now()
	for _, spec := range stateMachineSpecs {
		readings, err := m.readStateAges(ctx, spec, now)
		if err != nil {
			report.Unavailable = append(report.Unavailable, spec.Name)
			continue
		}
		report.ByState = append(report.ByState, readings...)
	}

	for i := range report.ByState {
		r := &report.ByState[i]
		if r.Rows == 0 {
			continue
		}
		if report.Worst == nil || r.AgeSeconds > report.Worst.AgeSeconds {
			worst := *r
			report.Worst = &worst
		}
	}
	return report
}

// readStateAges returns one reading per declared state, in declaration order.
// The state set is static so a state with zero rows still surfaces as a zero
// gauge instead of disappearing from the metric.
func (m *Monitoring) readStateAges(ctx context.Context, spec stateMachineSpec, now time.Time) ([]StateAgeReading, error) {
	readings := make([]StateAgeReading, 0, len(spec.States))
	index := make(map[string]int, len(spec.States))
	for _, state := range spec.States {
		index[state] = len(readings)
		readings = append(readings, StateAgeReading{Machine: spec.Name, State: state})
	}

	queryStates := append([]string{}, spec.States...)
	for alias := range spec.StatusAliases {
		queryStates = append(queryStates, alias)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(queryStates)), ",")
	query := fmt.Sprintf("SELECT id, status, %s FROM %s WHERE status IN (%s)",
		spec.TimestampColumn, spec.Table, placeholders)
	args := make([]any, len(queryStates))
	for i, state := range queryStates {
		args[i] = state
	}

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id, status string
		var rawTimestamp sql.NullString
		if err := rows.Scan(&id, &status, &rawTimestamp); err != nil {
			return nil, err
		}
		canonical := strings.ToLower(strings.TrimSpace(status))
		if alias, ok := spec.StatusAliases[canonical]; ok {
			canonical = alias
		}
		pos, ok := index[canonical]
		if !ok {
			continue
		}
		reading := &readings[pos]
		reading.Rows++

		observedAt, err := parseStateAgeTimestamp(rawTimestamp.String)
		if err != nil {
			continue
		}
		age := now.Sub(observedAt).Seconds()
		if age < 0 {
			age = 0
		}
		if age > reading.AgeSeconds || reading.OldestID == "" {
			reading.AgeSeconds = age
			reading.OldestID = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return readings, nil
}

// parseStateAgeTimestamp accepts both the RFC3339 timestamps written by the
// deploy/cici components and the "YYYY-MM-DD HH:MM:SS" form produced by
// SQLite's datetime('now') default.
func parseStateAgeTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	} {
		if layout == "2006-01-02 15:04:05" || layout == "2006-01-02T15:04:05" {
			if parsed, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
				return parsed, nil
			}
			continue
		}
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", raw)
}

// machineAvailable reports whether a machine's table was queried successfully.
func (r StateAgeReport) machineAvailable(machine string) bool {
	for _, unavailable := range r.Unavailable {
		if unavailable == machine {
			return false
		}
	}
	return true
}

// maxAge returns the oldest reading for a machine (or across all machines when
// machine is empty). The second return value is false when no machine could be
// queried, so callers can fail closed instead of alerting on an unknown value.
func (r StateAgeReport) maxAge(machine string) (float64, bool) {
	available := false
	max := 0.0
	for _, reading := range r.ByState {
		if machine != "" && reading.Machine != machine {
			continue
		}
		available = true
		if reading.AgeSeconds > max {
			max = reading.AgeSeconds
		}
	}
	if available {
		return max, true
	}
	for _, spec := range stateMachineSpecs {
		if (machine == "" || spec.Name == machine) && r.machineAvailable(spec.Name) {
			return 0, true
		}
	}
	return 0, false
}

// stateAgeMetricValue resolves alert-rule metric names for state staleness.
// The bool pair is (value, ok), matching metricValue; the third return value
// reports whether the name belongs to the state-age family at all, so
// metricValueWithSite can fall through to the host/process metrics.
//
// Supported names:
//   - state_age_seconds                        — oldest row across all machines
//   - deployment_state_age_seconds             — oldest deployment
//   - deployment_pending_age_seconds           — oldest deployment in one state
//   - cici_run_state_age_seconds               — oldest CICI run
//   - cici_run_running_age_seconds             — oldest CICI run in one state
//   - cici_state_age_seconds / cici_queued_age_seconds (short aliases)
func (m *Monitoring) stateAgeMetricValue(metric string) (float64, bool, bool) {
	return stateAgeMetricValueFromReport(metric, m.stateAgeReportForAlerts())
}

func stateAgeMetricValueFromReport(metric string, report StateAgeReport) (float64, bool, bool) {
	if metric == "state_age_seconds" {
		value, ok := report.maxAge("")
		return value, ok, true
	}

	for _, spec := range stateMachineSpecs {
		prefixes := []string{spec.Name, strings.TrimSuffix(spec.Name, "_run")}
		for _, prefix := range prefixes {
			if metric == prefix+"_state_age_seconds" {
				value, ok := report.maxAge(spec.Name)
				return value, ok, true
			}
			for _, state := range spec.States {
				if metric == prefix+"_"+state+"_age_seconds" {
					return report.stateAgeFor(spec.Name, state)
				}
			}
		}
	}
	return 0, false, false
}

// stateAgeFor returns the age of one machine/state pair. An unavailable table
// or an unknown state is reported as unknown (ok=false).
func (r StateAgeReport) stateAgeFor(machine, state string) (float64, bool, bool) {
	if !r.machineAvailable(machine) {
		return 0, false, true
	}
	for _, reading := range r.ByState {
		if reading.Machine == machine && reading.State == state {
			return reading.AgeSeconds, true, true
		}
	}
	return 0, false, true
}

// writeStateAgeGauges emits the derived staleness gauge for every available
// machine/state pair. Unavailable tables are skipped so a deployment-free
// installation does not advertise a misleading 0.
func writeStateAgeGauges(w http.ResponseWriter, report StateAgeReport) {
	if len(report.ByState) == 0 {
		return
	}
	writePromLine(w, "# HELP bigbase_state_age_seconds Age of the oldest row in a non-terminal state, in seconds\n")
	writePromLine(w, "# TYPE bigbase_state_age_seconds gauge\n")
	for _, reading := range report.ByState {
		if !report.machineAvailable(reading.Machine) {
			continue
		}
		writePromLinef(w, "bigbase_state_age_seconds{machine=\"%s\",state=\"%s\"} %.2f\n",
			reading.Machine, reading.State, reading.AgeSeconds)
	}
	writePromLine(w, "\n")
}

// stateAgeReportForAlerts computes a fresh report for the alert checker.
func (m *Monitoring) stateAgeReportForAlerts() StateAgeReport {
	if m == nil || m.db == nil {
		return StateAgeReport{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), stateAgeQueryTimeout)
	defer cancel()
	return m.stateAgeReport(ctx)
}
