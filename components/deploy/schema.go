package deploy

import (
	"context"
	"fmt"
	"strings"
)

func (d *Deploy) ensureSiteIDColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN site_id TEXT NOT NULL DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add site_id column: %w", err)
}
func (d *Deploy) ensureErrorMessageColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN error_message TEXT NOT NULL DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add error_message column: %w", err)
}
func (d *Deploy) ensureBuildLogColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN build_log TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add build_log column: %w", err)
}
func (d *Deploy) ensurePassthroughPathsColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN passthrough_paths TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add passthrough_paths column: %w", err)
}
func (d *Deploy) ensurePIDColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN pid INTEGER NOT NULL DEFAULT 0`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add pid column: %w", err)
}
func (d *Deploy) ensureStatusHistoryColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN status_history TEXT DEFAULT '[]'`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add status_history column: %w", err)
}
func (d *Deploy) ensureManifestPathColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN manifest_path TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add manifest_path column: %w", err)
}
func (d *Deploy) ensureHealthSummaryColumn() error {
	_, err := d.db.ExecContext(context.Background(),
		`ALTER TABLE deployments ADD COLUMN health_summary TEXT DEFAULT ''`)
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return fmt.Errorf("add health_summary column: %w", err)
}
func (d *Deploy) ensureDeploymentsRepoCreatedIndex() error {
	_, err := d.db.ExecContext(context.Background(),
		`CREATE INDEX IF NOT EXISTS idx_deployments_repo_created ON deployments(repo_id, created_at DESC)`)
	if err != nil {
		return fmt.Errorf("create deployments repo_created index: %w", err)
	}
	return nil
}
