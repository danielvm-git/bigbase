package kernel

import "context"

type siteIDKeyType string

const siteIDKey siteIDKeyType = "site_id"

// WithSiteID returns a child context with the site ID set (site-scoped deploy keys).
func WithSiteID(ctx context.Context, siteID string) context.Context {
	return context.WithValue(ctx, siteIDKey, siteID)
}

// SiteIDFromContext extracts the site ID from context.
// Returns ("", false) when no site ID has been set.
func SiteIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(siteIDKey).(string)
	return id, ok
}
