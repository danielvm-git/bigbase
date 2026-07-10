package kernel

import (
	"context"
	"testing"
)

// TestProjectIDContextRoundTrip verifies that WithProjectID and
// ProjectIDFromContext form a correct round-trip.
func TestProjectIDContextRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		projectID int64
	}{
		{"positive", 42},
		{"zero", 0},
		{"negative", -1},
		{"large", 1<<62 - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithProjectID(ctx, tt.projectID)
			got, ok := ProjectIDFromContext(ctx)

			if !ok {
				t.Errorf("ProjectIDFromContext() ok = false, want true")
			}
			if got != tt.projectID {
				t.Errorf("ProjectIDFromContext() = %d, want %d", got, tt.projectID)
			}
		})
	}
}

// TestProjectIDContextMissing verifies that ProjectIDFromContext returns
// (0, false) when no project ID has been set in the context.
func TestProjectIDContextMissing(t *testing.T) {
	ctx := context.Background()
	got, ok := ProjectIDFromContext(ctx)

	if ok {
		t.Errorf("ProjectIDFromContext() ok = true, want false")
	}
	if got != 0 {
		t.Errorf("ProjectIDFromContext() = %d, want 0", got)
	}
}

// TestProjectIDContextTypeIsolation verifies that an untyped string key
// with value "project_id" does NOT satisfy the typed key lookup.
// This is the critical security property — prevents context key spoofing.
func TestProjectIDContextTypeIsolation(t *testing.T) {
	// Set a string key "project_id" in context (simulating an attacker)
	//nolint:staticcheck // deliberate use of string key to test type-safety isolation
	ctx := context.WithValue(context.Background(), "project_id", int64(999))
	got, ok := ProjectIDFromContext(ctx)

	if ok {
		t.Errorf("ProjectIDFromContext() ok = true, want false — typed key prevented spoofing")
	}
	if got != 0 {
		t.Errorf("ProjectIDFromContext() = %d, want 0", got)
	}
}

// TestProjectIDContextMultiple verifies that calling WithProjectID
// multiple times chains correctly (last value wins, per context semantics).
func TestProjectIDContextMultiple(t *testing.T) {
	ctx := context.Background()
	ctx = WithProjectID(ctx, 1)
	ctx = WithProjectID(ctx, 2)
	ctx = WithProjectID(ctx, 3)

	got, ok := ProjectIDFromContext(ctx)
	if !ok {
		t.Errorf("ProjectIDFromContext() ok = false, want true")
	}
	if got != 3 {
		t.Errorf("ProjectIDFromContext() = %d, want 3 (last value wins)", got)
	}
}

func TestSiteIDContextRoundTrip(t *testing.T) {
	ctx := WithSiteID(context.Background(), "site-abc")
	got, ok := SiteIDFromContext(ctx)
	if !ok || got != "site-abc" {
		t.Fatalf("SiteIDFromContext() = %q, %v; want site-abc, true", got, ok)
	}
}
