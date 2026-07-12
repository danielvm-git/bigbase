package kernel

import (
	"context"
	"testing"
)

func TestSiteIDContextRoundTrip(t *testing.T) {
	ctx := WithSiteID(context.Background(), "site-abc")
	got, ok := SiteIDFromContext(ctx)
	if !ok || got != "site-abc" {
		t.Fatalf("SiteIDFromContext() = %q, %v; want site-abc, true", got, ok)
	}
}
