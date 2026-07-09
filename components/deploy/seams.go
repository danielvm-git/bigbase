package deploy

import "context"

// Diagnosis is an AI-generated deploy failure explanation.
type Diagnosis struct {
	Diagnosis string `json:"diagnosis"`
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
}

// CorrelatedEvent is a single related monitoring event.
type CorrelatedEvent struct {
	Hook      string         `json:"hook"`
	Data      map[string]any `json:"data"`
	Timestamp string         `json:"timestamp"`
}

// RelatedEvents bundles correlated events for a deployment window.
type RelatedEvents struct {
	DeployID    string                       `json:"deploy_id"`
	WindowStart string                       `json:"window_start"`
	WindowEnd   string                       `json:"window_end"`
	Events      map[string][]CorrelatedEvent `json:"events"`
}

// DeployDiagnosisReader resolves stored deploy failure diagnoses.
type DeployDiagnosisReader interface {
	GetDiagnosis(ctx context.Context, deployID string) (Diagnosis, bool, error)
}

// DeployRelatedEventsReader resolves correlated events for a deployment.
type DeployRelatedEventsReader interface {
	GetRelatedEvents(ctx context.Context, deployID string) (RelatedEvents, bool, error)
}
