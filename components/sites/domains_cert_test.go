package sites_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

// certMaxAttempts mirrors the unexported budget in the sites package. Keeping
// the literal here makes the test an independent oracle for the contract.
const certMaxAttempts = 5

func setupCertTest(t *testing.T, info sites.CertInfoFunc) (*sites.Sites, http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger, Dir: t.TempDir()})
	s := sites.New(sites.Options{DB: d, Logger: logger, CertInfo: info})

	k.Register(d)
	k.Register(g)
	k.Register(s)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return s, s.Handler(), d
}

// certSeed describes the persisted attempt metadata for a seeded domain.
type certSeed struct {
	attempts      int
	startedAt     *time.Time
	lastAttemptAt *time.Time
	retryAfter    *time.Time
	lastError     string
}

func timePtr(t time.Time) *time.Time { return &t }

// seedDomain inserts an already-verified custom domain with explicit cert
// attempt metadata so status transitions can be exercised without DNS or a
// real ACME server.
func seedDomain(t *testing.T, d *db.DB, siteID, domain string, seed certSeed) {
	t.Helper()
	now := time.Now().UTC()
	started := now
	if seed.startedAt != nil {
		started = *seed.startedAt
	}
	var lastAttempt, retryAfter any
	if seed.lastAttemptAt != nil {
		lastAttempt = seed.lastAttemptAt.UTC().Format(time.RFC3339)
	}
	if seed.retryAfter != nil {
		retryAfter = seed.retryAfter.UTC().Format(time.RFC3339)
	}
	_, err := d.Exec(
		`INSERT INTO site_domains
			(id, site_id, domain, verify_token, verified_at, created_at,
			 cert_attempts, cert_started_at, cert_last_attempt_at, cert_last_error, cert_retry_after)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"dom-"+domain, siteID, domain, "token", now.Format(time.RFC3339), now.Format(time.RFC3339),
		seed.attempts, started.Format(time.RFC3339), lastAttempt, seed.lastError, retryAfter)
	if err != nil {
		t.Fatalf("seed domain %s: %v", domain, err)
	}
}

func listDomain(t *testing.T, h http.Handler, siteID, domain string) map[string]any {
	t.Helper()
	req := authedRequest(http.MethodGet, "/api/sites/"+siteID+"/domains", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list domains: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	for _, d := range resp.Data {
		if d["domain"] == domain {
			return d
		}
	}
	t.Fatalf("domain %q not found in list: %s", domain, w.Body.String())
	return nil
}

func TestCertStatusAging(t *testing.T) {
	now := time.Now().UTC()
	noCert := func(context.Context, string) (time.Time, bool) { return time.Time{}, false }
	futureCert := func(context.Context, string) (time.Time, bool) {
		return now.Add(90 * 24 * time.Hour), true
	}
	expiredCert := func(context.Context, string) (time.Time, bool) {
		return now.Add(-time.Hour), true
	}

	tests := []struct {
		name           string
		certInfo       sites.CertInfoFunc
		seed           certSeed
		wantStatus     string
		wantErrContain string
	}{
		{
			name:       "fresh provisioning stays provisioning",
			certInfo:   noCert,
			seed:       certSeed{attempts: 1, startedAt: timePtr(now)},
			wantStatus: "provisioning",
		},
		{
			name:     "fails once the attempt budget is exhausted",
			certInfo: noCert,
			seed: certSeed{
				attempts:  certMaxAttempts,
				startedAt: timePtr(now),
				lastError: "acme: dns-01 challenge failed",
			},
			wantStatus:     "failed",
			wantErrContain: "dns-01",
		},
		{
			name:     "fails once provisioning is stale",
			certInfo: noCert,
			seed: certSeed{
				attempts:  1,
				startedAt: timePtr(now.Add(-2 * time.Hour)),
			},
			wantStatus:     "failed",
			wantErrContain: "timed out",
		},
		{
			name:       "valid certificate is unchanged",
			certInfo:   futureCert,
			seed:       certSeed{},
			wantStatus: "valid",
		},
		{
			name:       "expired certificate is unchanged",
			certInfo:   expiredCert,
			seed:       certSeed{},
			wantStatus: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, h, d := setupCertTest(t, tt.certInfo)
			siteID := createTestSite(t, h, d)
			domain := "cert-" + strings.ReplaceAll(tt.name, " ", "-") + ".example.com"
			seedDomain(t, d, siteID, domain, tt.seed)

			got := listDomain(t, h, siteID, domain)
			if got["cert_status"] != tt.wantStatus {
				t.Errorf("cert_status = %v, want %s", got["cert_status"], tt.wantStatus)
			}
			errText, _ := got["cert_error"].(string)
			if tt.wantErrContain == "" && errText != "" {
				t.Errorf("cert_error = %q, want empty", errText)
			}
			if tt.wantErrContain != "" && !strings.Contains(errText, tt.wantErrContain) {
				t.Errorf("cert_error = %q, want substring %q", errText, tt.wantErrContain)
			}
		})
	}
}

func TestRetryDomainCertResetsAttempts(t *testing.T) {
	now := time.Now().UTC()
	noCert := func(context.Context, string) (time.Time, bool) { return time.Time{}, false }

	_, h, d := setupCertTest(t, noCert)
	siteID := createTestSite(t, h, d)
	domain := "retry.example.com"
	seedDomain(t, d, siteID, domain, certSeed{
		attempts:      certMaxAttempts,
		startedAt:     timePtr(now.Add(-2 * time.Hour)),
		lastAttemptAt: timePtr(now.Add(-2 * time.Hour)),
		lastError:     "acme: rate limited",
	})

	// Pre-condition: the domain is already failed before the retry.
	if before := listDomain(t, h, siteID, domain); before["cert_status"] != "failed" {
		t.Fatalf("precondition: cert_status = %v, want failed", before["cert_status"])
	}

	req := authedRequest(http.MethodPost,
		"/api/sites/"+siteID+"/domains/"+domain+"/cert/retry", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("retry: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Domain     string `json:"domain"`
		CertStatus string `json:"cert_status"`
		Attempts   int    `json:"attempts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode retry result: %v", err)
	}
	if res.Domain != domain {
		t.Errorf("retry domain = %q, want %q", res.Domain, domain)
	}
	if res.CertStatus != "provisioning" {
		t.Errorf("retry cert_status = %q, want provisioning", res.CertStatus)
	}
	if res.Attempts != 0 {
		t.Errorf("retry attempts = %d, want 0", res.Attempts)
	}

	var attempts int
	var lastError string
	if err := d.QueryRow(
		`SELECT cert_attempts, cert_last_error FROM site_domains WHERE site_id = ? AND domain = ?`,
		siteID, domain).Scan(&attempts, &lastError); err != nil {
		t.Fatalf("read attempt metadata: %v", err)
	}
	if attempts != 0 {
		t.Errorf("persisted attempts = %d, want 0", attempts)
	}
	if lastError != "" {
		t.Errorf("persisted last_error = %q, want empty", lastError)
	}

	after := listDomain(t, h, siteID, domain)
	if after["cert_status"] != "provisioning" {
		t.Errorf("post-retry cert_status = %v, want provisioning", after["cert_status"])
	}
	if errText, _ := after["cert_error"].(string); errText != "" {
		t.Errorf("post-retry cert_error = %q, want empty", errText)
	}
}

func TestRetryDomainCertRequiresVerifiedDomain(t *testing.T) {
	noCert := func(context.Context, string) (time.Time, bool) { return time.Time{}, false }

	_, h, d := setupCertTest(t, noCert)
	siteID := createTestSite(t, h, d)

	body := `{"domain":"unverified.example.com"}`
	req := authedRequest(http.MethodPost, "/api/sites/"+siteID+"/domains", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = authedRequest(http.MethodPost,
		"/api/sites/"+siteID+"/domains/unverified.example.com/cert/retry", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("retry unverified: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRetryDomainCertUnknownDomain(t *testing.T) {
	noCert := func(context.Context, string) (time.Time, bool) { return time.Time{}, false }

	_, h, d := setupCertTest(t, noCert)
	siteID := createTestSite(t, h, d)

	req := authedRequest(http.MethodPost,
		fmt.Sprintf("/api/sites/%s/domains/missing.example.com/cert/retry", siteID), nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("retry unknown: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
