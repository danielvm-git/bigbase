package kernel

import "context"

// projectIDKeyType is a distinct, unexported type used as the context key
// for project ID values. Using an unexported type prevents external packages
// from injecting spoofed project IDs via context.WithValue.
type projectIDKeyType string

// projectIDKey is the singleton context key for storing project IDs.
const projectIDKey projectIDKeyType = "project_id"

// WithProjectID returns a child context with the given project ID set.
// Use ProjectIDFromContext to extract it.
func WithProjectID(ctx context.Context, projectID int64) context.Context {
	return context.WithValue(ctx, projectIDKey, projectID)
}

// ProjectIDFromContext extracts the project ID from context.
// Returns (0, false) when no project ID has been set.
func ProjectIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(projectIDKey).(int64)
	return id, ok
}
