package deploy_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/kernel"
)

func TestPruneRequestLogs(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	defer func() { _ = database.Stop(&kernel.Context{}) }()

	dep := deploy.New(deploy.Options{DB: database, Logger: logger})
	_ = dep.Start(&kernel.Context{})
	defer func() { _ = dep.Stop(&kernel.Context{}) }()

	// 1. Insert some old and new logs
	now := time.Now().UTC()
	oldTime := now.AddDate(0, 0, -8).Format(time.RFC3339)
	newTime := now.AddDate(0, 0, -1).Format(time.RFC3339)

	_, err := database.ExecContext(context.Background(),
		`INSERT INTO site_request_logs (id, site_id, method, path, status, duration_ms, created_at)
		 VALUES ('old', 's1', 'GET', '/', 200, 10, ?),
		        ('new', 's1', 'GET', '/', 200, 10, ?)`,
		oldTime, newTime)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// 2. Prune
	dep.PruneRequestLogs()

	// 3. Verify
	var count int
	_ = database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM site_request_logs").Scan(&count)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	var id string
	_ = database.QueryRowContext(context.Background(), "SELECT id FROM site_request_logs").Scan(&id)
	if id != "new" {
		t.Errorf("remaining id = %s, want 'new'", id)
	}
}

func TestSiteRequestLogsEndToEnd(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	defer func() { _ = database.Stop(&kernel.Context{}) }()

	// We need a real Proxy to trigger the middleware
	p := proxy.New(proxy.Options{Logger: logger})
	dep := deploy.New(deploy.Options{DB: database, Logger: logger, HostRouter: p})
	p.SetRequestLogger(dep)

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("proxy start: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	if err := dep.Start(&kernel.Context{}); err != nil {
		t.Fatalf("deploy start: %v", err)
	}
	defer func() { _ = dep.Stop(&kernel.Context{}) }()

	// 1. Setup a dummy backend to proxy to
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})
	srv := http.Server{Handler: backend}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	// 2. Register the host
	host := "logs.test"
	siteID := "s1"
	if err := p.RegisterDeploymentHost(host, port, siteID); err != nil {
		t.Fatalf("register: %v", err)
	}

	// 3. Make the request via Proxy
	req, _ := http.NewRequest("POST", "http://"+host+"/some/path", nil)
	w := httptest.NewRecorder()
	p.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("proxy failed: %d: %s", w.Code, w.Body.String())
	}

	// 4. Wait for async recording (max 1s)
	start := time.Now()
	var count int
	for time.Since(start) < 1*time.Second {
		_ = database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM site_request_logs").Scan(&count)
		if count > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if count == 0 {
		t.Fatal("request log was not recorded")
	}

	// 5. Verify the log
	var l struct {
		SiteID string
		Method string
		Path   string
		Status int
	}
	err := database.QueryRowContext(context.Background(),
		"SELECT site_id, method, path, status FROM site_request_logs WHERE site_id = ?", siteID).
		Scan(&l.SiteID, &l.Method, &l.Path, &l.Status)
	if err != nil {
		t.Fatalf("query log failed: %v", err)
	}

	if l.Method != "POST" || l.Path != "/some/path" || l.Status != http.StatusCreated {
		t.Errorf("log data mismatch: %+v", l)
	}
}
