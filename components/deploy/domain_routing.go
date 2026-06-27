package deploy

import (
	"context"
	"fmt"
	"time"
)

// RegisterCustomDomainHosts queries verified custom domains for the given site
// and registers each one as an additional proxy host pointing to the deployment port.
// This allows custom domains (e.g. myapp.com) to route to the same deployment
// as the default host (e.g. myapp.bigbase.click).
//
// Unverified domains (verified_at IS NULL) are skipped — they must pass DNS
// verification via the sites component's /api/sites/{id}/domains/{domain}/verify
// endpoint before they can route traffic.
//
// Returns nil when there are no custom domains or no host router is configured.
func (d *Deploy) RegisterCustomDomainHosts(ctx context.Context, siteID string, port int) error {
	if d.hostRouter == nil || siteID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(ctx,
		`SELECT domain FROM site_domains WHERE site_id = ? AND verified_at IS NOT NULL`,
		siteID)
	if err != nil {
		// Table may not exist yet — treat as empty.
		if isNoSuchTable(err) {
			return nil
		}
		return fmt.Errorf("query custom domains: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var count int
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			d.logger.Warn("scan domain", "error", err)
			continue
		}
		if err := d.hostRouter.RegisterDeploymentHost(domain, port, siteID, nil, nil); err != nil {
			d.logger.Warn("register custom domain host", "domain", domain, "error", err)
			continue
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate domains: %w", err)
	}

	if count > 0 {
		d.logger.Info("registered custom domain hosts", "site_id", siteID, "count", count)
	}
	return nil
}
