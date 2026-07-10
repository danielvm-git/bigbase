package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// OnboardingStep represents one step in the new-user checklist.
type OnboardingStep struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Done  bool   `json:"done"`
}

// handleOnboarding serves GET /api/onboarding.
// It checks whether each checklist action has been taken and returns the result.
func (a *API) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	steps := []OnboardingStep{
		{ID: "create_database", Label: "Create database"},
		{ID: "add_collection", Label: "Add collection"},
		{ID: "deploy_function", Label: "Deploy function"},
		{ID: "invite_team_member", Label: "Invite team member"},
	}

	// "create_database" — any user-visible collection exists
	steps[0].Done = a.collectionExists(ctx)
	// "add_collection" — same signal: at least one collection created
	steps[1].Done = steps[0].Done
	// "deploy_function" — at least one row in functions table
	steps[2].Done = a.tableHasRows(ctx, "functions")
	// "invite_team_member" — more than one user exists
	steps[3].Done = a.tableHasMoreThanOneRow(ctx, "users")

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"steps": steps})
}

// collectionExists returns true if at least one user-created collection table exists.
func (a *API) collectionExists(ctx context.Context) bool {
	row := a.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	// Subtract internal tables that are always present
	internal := 0
	for range internalTables {
		internal++
	}
	return (count - internal) > 0
}

// tableHasRows returns true if the given table exists and has at least one row.
func (a *API) tableHasRows(ctx context.Context, table string) bool {
	row := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
	var exists int
	if err := row.Scan(&exists); err != nil || exists == 0 {
		return false
	}
	row = a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// tableHasMoreThanOneRow returns true if the table has more than one row.
func (a *API) tableHasMoreThanOneRow(ctx context.Context, table string) bool {
	row := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
	var exists int
	if err := row.Scan(&exists); err != nil || exists == 0 {
		return false
	}
	row = a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 1
}
