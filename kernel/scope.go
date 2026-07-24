package kernel

import "context"

type siteIDKeyType string
type orgIDKeyType string

const siteIDKey siteIDKeyType = "site_id"
const orgIDKey orgIDKeyType = "org_id"

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

// WithOrgID returns a child context with the org ID set.
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	return context.WithValue(ctx, orgIDKey, orgID)
}

// OrgIDFromContext extracts the org ID from context.
// Returns (0, false) when no org ID has been set.
func OrgIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(orgIDKey).(int64)
	return id, ok
}
