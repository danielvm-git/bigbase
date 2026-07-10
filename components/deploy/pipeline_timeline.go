package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PipelineTimeline records per-stage timestamps for a deployment pipeline.
// Orthogonal to DeploymentState — see ADR 0007.
// Pointer fields ensure unset stages are omitted from JSON (time.Time zero values are not omitempty-safe).
type PipelineTimeline struct {
	CloneStart  *time.Time `json:"clone_start,omitempty"`
	CloneEnd    *time.Time `json:"clone_end,omitempty"`
	BuildStart  *time.Time `json:"build_start,omitempty"`
	BuildEnd    *time.Time `json:"build_end,omitempty"`
	StartStart  *time.Time `json:"start_start,omitempty"`
	StartEnd    *time.Time `json:"start_end,omitempty"`
	HealthStart *time.Time `json:"health_start,omitempty"`
	HealthEnd   *time.Time `json:"health_end,omitempty"`
}

func timelineNow() *time.Time {
	t := time.Now()
	return &t
}

func (d *Deploy) ensurePipelineTimelineColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN pipeline_timeline TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if isDuplicateColumnError(err) {
		return nil
	}
	return fmt.Errorf("add pipeline_timeline column: %w", err)
}

func (d *Deploy) persistPipelineTimeline(id string, tl *PipelineTimeline) {
	if tl == nil {
		return
	}
	data, err := json.Marshal(tl)
	if err != nil {
		d.logger.Error("marshal pipeline timeline", "id", id, "error", err)
		return
	}
	_, err = d.db.ExecContext(context.Background(),
		"UPDATE deployments SET pipeline_timeline = ? WHERE id = ?", string(data), id)
	if err != nil {
		d.logger.Error("persist pipeline timeline", "id", id, "error", err)
	}
}

func parsePipelineTimeline(raw string) *PipelineTimeline {
	if raw == "" {
		return nil
	}
	var tl PipelineTimeline
	if err := json.Unmarshal([]byte(raw), &tl); err != nil {
		return nil
	}
	return &tl
}
