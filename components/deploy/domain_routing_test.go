package deploy_test

import (
	"context"
	"sync"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

// mockHostRouter records registered/unregistered hosts for verification.
type mockHostRouter struct {
	mu      sync.Mutex
	hosts   map[string]int // host → port
	siteIDs map[string]string
}

func newMockHostRouter() *mockHostRouter {
	return &mockHostRouter{
		hosts:   make(map[string]int),
		siteIDs: make(map[string]string),
	}
}

func (m *mockHostRouter) RegisterDeploymentHost(host string, port int, siteID string, _ []string, _ map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hosts[host] = port
	m.siteIDs[host] = siteID
	return nil
}

func (m *mockHostRouter) UnregisterDeploymentHost(host string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hosts, host)
	delete(m.siteIDs, host)
}

func (m *mockHostRouter) getPort(host string) (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.hosts[host]
	return p, ok
}

// migrateSiteDomains creates the site_domains table (matching the sites component schema).
func migrateSiteDomains(t *testing.T, database *db.DB) {
	t.Helper()
	_, err := database.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS site_domains (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		domain TEXT NOT NULL UNIQUE,
		verify_token TEXT NOT NULL,
		verified_at TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("migrate site_domains: %v", err)
	}
}

func TestCustomDomainRouting(t *testing.T) {
	t.Run("registers_verified_custom_domain_as_proxy_host", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		router := newMockHostRouter()
		dep := deploy.New(deploy.Options{
			DB:         database,
			Logger:     logger,
			HostRouter: router,
		})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		// Create the site_domains table and add a verified custom domain.
		migrateSiteDomains(t, database)
		now := "2026-06-26T12:00:00Z"
		_, err := database.ExecContext(context.Background(),
			`INSERT INTO site_domains (id, site_id, domain, verify_token, verified_at, created_at)
			 VALUES ('dom-1', 'site-1', 'myapp.com', 'tok', ?, ?)`, now, now)
		if err != nil {
			t.Fatalf("insert verified domain: %v", err)
		}

		// Register custom domain hosts for this site.
		err = dep.RegisterCustomDomainHosts(context.Background(), "site-1", 9999)
		if err != nil {
			t.Fatalf("RegisterCustomDomainHosts: %v", err)
		}

		// Verify the custom domain is registered.
		port, ok := router.getPort("myapp.com")
		if !ok {
			t.Fatal("expected custom domain myapp.com to be registered as host")
		}
		if port != 9999 {
			t.Fatalf("expected port 9999, got %d", port)
		}
	})

	t.Run("skips_unverified_custom_domains", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		router := newMockHostRouter()
		dep := deploy.New(deploy.Options{
			DB:         database,
			Logger:     logger,
			HostRouter: router,
		})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		migrateSiteDomains(t, database)

		// Insert an UNVERIFIED domain (verified_at IS NULL).
		_, err := database.ExecContext(context.Background(),
			`INSERT INTO site_domains (id, site_id, domain, verify_token, created_at)
			 VALUES ('dom-2', 'site-2', 'unverified.com', 'tok', '2026-06-26T12:00:00Z')`)
		if err != nil {
			t.Fatalf("insert unverified domain: %v", err)
		}

		err = dep.RegisterCustomDomainHosts(context.Background(), "site-2", 9999)
		if err != nil {
			t.Fatalf("RegisterCustomDomainHosts: %v", err)
		}

		// Verify unverified domain is NOT registered.
		_, ok := router.getPort("unverified.com")
		if ok {
			t.Fatal("expected unverified custom domain to NOT be registered")
		}
	})

	t.Run("handles_no_custom_domains_gracefully", func(t *testing.T) {
		logger := testLogger{}
		database := db.New(db.Options{Path: ":memory:", Logger: logger})
		_ = database.Start(&kernel.Context{})
		defer func() { _ = database.Stop(&kernel.Context{}) }()

		router := newMockHostRouter()
		dep := deploy.New(deploy.Options{
			DB:         database,
			Logger:     logger,
			HostRouter: router,
		})
		_ = dep.Start(&kernel.Context{})
		defer func() { _ = dep.Stop(&kernel.Context{}) }()

		migrateSiteDomains(t, database)

		// Site with no custom domains.
		err := dep.RegisterCustomDomainHosts(context.Background(), "site-no-domains", 9999)
		if err != nil {
			t.Fatalf("expected no error for site without domains, got: %v", err)
		}
	})
}
