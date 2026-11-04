package sites

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// site_domains uses a composite UNIQUE(site_id, domain): a domain is scoped to a
// site rather than globally exclusive. Registration alone never routes traffic
// (only verified domains are registered with the proxy), so DNS ownership — not
// first-to-register — is the source of truth. This removes the squatting vector
// where an unverified registration could globally block a domain.
const domainsMigration = `CREATE TABLE IF NOT EXISTS site_domains (
	id TEXT PRIMARY KEY,
	site_id TEXT NOT NULL,
	domain TEXT NOT NULL,
	verify_token TEXT NOT NULL,
	verified_at TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(site_id, domain)
)`

// domainRE validates a fully-qualified domain name (one or more labels). Mirrors
// the client-side check in SiteDomainsTab.tsx so the API is not dependent on the
// UI for input validation.
var domainRE = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)

// certWarnWindow is how far before expiry a certificate is flagged "expiring_soon".
const certWarnWindow = 30 * 24 * time.Hour

// Certificate attempt budget. A verified domain with no cached certificate is
// "provisioning"; it ages out to "failed" once either limit is reached so a
// broken ACME issuance cannot stay provisioning forever.
const (
	// certMaxAttempts is the number of provisioning observations tolerated
	// before the domain is reported as failed.
	certMaxAttempts = 5
	// certStaleAfter is the wall-clock age, measured from the start of the
	// provisioning window, after which the domain is reported as failed.
	certStaleAfter = 15 * time.Minute
	// certRetryAfter is the minimum interval between recorded provisioning
	// observations and the backoff advertised after a manual retry.
	certRetryAfter = 2 * time.Minute
)

// certState is the persisted attempt metadata for one domain. Nil timestamps
// mean "not recorded yet"; the verified_at column is used as a fallback anchor
// so domains verified before this metadata existed still age out.
type certState struct {
	Attempts      int
	LastAttemptAt *time.Time
	LastError     string
	RetryAfter    *time.Time
	StartedAt     *time.Time
}

type SiteDomain struct {
	ID          string  `json:"id"`
	SiteID      string  `json:"site_id"`
	Domain      string  `json:"domain"`
	VerifyToken string  `json:"verify_token"`
	VerifiedAt  *string `json:"verified_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	// Computed display fields — populated from the proxy's certificate cache,
	// never stored. Empty when the domain is unverified or HTTPS is disabled.
	CertStatus    string  `json:"cert_status,omitempty"`
	CertExpiresAt *string `json:"cert_expires_at,omitempty"`
	// CertError carries the stored failure reason when cert_status=failed.
	CertError string `json:"cert_error,omitempty"`
	// CertAttempts and CertRetryAfter expose the persisted attempt budget so
	// operators (and the UI) can see how close a domain is to failing.
	CertAttempts   int     `json:"cert_attempts"`
	CertRetryAfter *string `json:"cert_retry_after,omitempty"`
}

// CertRetryResult is the typed response returned by the manual retry endpoint.
type CertRetryResult struct {
	Domain     string `json:"domain"`
	CertStatus string `json:"cert_status"`
	Attempts   int    `json:"attempts"`
	RetryAfter string `json:"retry_after,omitempty"`
}

// verifyLimiter throttles the (DNS-lookup-backed) verify endpoint per site+domain
// so an authenticated caller cannot drive unbounded outbound DNS traffic.
type verifyLimiter struct {
	mu       sync.Mutex
	last     map[string]time.Time
	interval time.Duration
}

func newVerifyLimiter(interval time.Duration) *verifyLimiter {
	return &verifyLimiter{last: make(map[string]time.Time), interval: interval}
}

func (v *verifyLimiter) allow(key string, now time.Time) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if t, ok := v.last[key]; ok && now.Sub(t) < v.interval {
		return false
	}
	v.last[key] = now
	return true
}

func (s *Sites) migrateDomains() error {
	if err := s.db.Migrate(domainsMigration); err != nil {
		return err
	}
	// Persisted certificate attempt metadata. ALTER TABLE ADD COLUMN is not
	// idempotent, so each statement is attempted and duplicate-column errors
	// ignored. Any unexpected database failure is returned wrapped.
	// The DDL is deliberately dialect-neutral (TEXT / INTEGER, no datetime('now')
	// defaults) so it also applies cleanly on PostgreSQL.
	for _, stmt := range []string{
		`ALTER TABLE site_domains ADD COLUMN cert_attempts INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE site_domains ADD COLUMN cert_started_at TEXT`,
		`ALTER TABLE site_domains ADD COLUMN cert_last_attempt_at TEXT`,
		`ALTER TABLE site_domains ADD COLUMN cert_last_error TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE site_domains ADD COLUMN cert_retry_after TEXT`,
	} {
		if err := s.db.Migrate(stmt); err != nil && !isDuplicateColumnError(err) {
			return fmt.Errorf("migrate site_domains column: %w", err)
		}
	}
	return nil
}

func (s *Sites) handleDomains(w http.ResponseWriter, r *http.Request) {
	// Path: /api/sites/{id}/domains[/{domain}[/verify]]
	path := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// parts[0] = site id, parts[1] = "domains", parts[2] = domain, parts[3] = "verify"
	if len(parts) < 2 || parts[1] != "domains" {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	siteID := parts[0]

	if !s.requireSiteOwnership(r.Context(), w, siteID) {
		return
	}

	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		s.listDomains(w, r, siteID)
	case len(parts) == 2 && r.Method == http.MethodPost:
		s.registerDomain(w, r, siteID)
	case len(parts) == 3 && r.Method == http.MethodDelete:
		s.deleteDomain(w, r, siteID, parts[2])
	case len(parts) == 4 && parts[3] == "verify" && r.Method == http.MethodPost:
		s.verifyDomain(w, r, siteID, parts[2])
	case len(parts) == 5 && parts[3] == "cert" && parts[4] == "retry" && r.Method == http.MethodPost:
		s.retryDomainCert(w, r, siteID, parts[2])
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Sites) listDomains(w http.ResponseWriter, r *http.Request, siteID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, site_id, domain, verify_token, verified_at, created_at,
		        cert_attempts, cert_started_at, cert_last_attempt_at, cert_last_error, cert_retry_after
		 FROM site_domains WHERE site_id = ? ORDER BY created_at`, siteID)
	if err != nil {
		s.logger.Error("list domains", "site_id", siteID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	domains := make([]SiteDomain, 0)
	states := make([]certState, 0)
	for rows.Next() {
		var d SiteDomain
		var verifiedAt, startedAt, lastAttemptAt, retryAfter *string
		var lastError string
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Domain, &d.VerifyToken, &verifiedAt, &d.CreatedAt,
			&d.CertAttempts, &startedAt, &lastAttemptAt, &lastError, &retryAfter); err != nil {
			s.logger.Error("scan domain", "error", err)
			continue
		}
		d.VerifiedAt = verifiedAt
		domains = append(domains, d)
		states = append(states, certState{
			Attempts:      d.CertAttempts,
			StartedAt:     parseRFC3339Ptr(startedAt),
			LastAttemptAt: parseRFC3339Ptr(lastAttemptAt),
			LastError:     lastError,
			RetryAfter:    parseRFC3339Ptr(retryAfter),
		})
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate domains", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Annotate verified domains with live certificate status from the proxy,
	// falling back to the persisted attempt budget when no certificate is cached.
	for i := range domains {
		if domains[i].VerifiedAt == nil {
			continue
		}
		st := &states[i]
		status, expiresAt, reason := s.certStatusFor(ctx, siteID, domains[i].Domain, domains[i].VerifiedAt, st)
		domains[i].CertStatus = status
		domains[i].CertExpiresAt = expiresAt
		domains[i].CertError = reason
		domains[i].CertAttempts = st.Attempts
		if st.RetryAfter != nil {
			iso := st.RetryAfter.UTC().Format(time.RFC3339)
			domains[i].CertRetryAfter = &iso
		}
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": domains})
}

// certStatusFor resolves the certificate status for a verified domain via the
// optional CertInfo hook, then applies the persisted attempt budget when no
// certificate is cached yet. It returns the status, ISO expiry (when present),
// and a human-readable failure reason (only when status is "failed").
//
// While a domain is still provisioning, this records at most one attempt
// observation per certRetryAfter window. That advances the N-attempt budget
// without requiring a live ACME callback inside the sites component.
func (s *Sites) certStatusFor(ctx context.Context, siteID, domain string, verifiedAt *string, st *certState) (string, *string, string) {
	if s.certInfo == nil {
		return "", nil, ""
	}
	now := time.Now().UTC()
	if notAfter, ok := s.certInfo(ctx, domain); ok {
		status, expiresAt := certStatus(notAfter, true, now)
		return status, expiresAt, ""
	}

	verified := parseRFC3339Ptr(verifiedAt)
	if failed, reason := evaluateCertAttempts(now, *st, verified); failed {
		return "failed", nil, reason
	}

	if st.RetryAfter == nil || !now.Before(*st.RetryAfter) {
		next := now.Add(certRetryAfter)
		recorded, err := s.recordCertAttempt(ctx, siteID, domain, now, next)
		if err != nil {
			s.logger.Warn("record cert attempt", "site_id", siteID, "domain", domain, "error", err)
		} else if recorded {
			st.Attempts++
			st.LastAttemptAt = &now
			st.RetryAfter = &next
			if st.StartedAt == nil {
				st.StartedAt = &now
			}
			if failed, reason := evaluateCertAttempts(now, *st, verified); failed {
				return "failed", nil, reason
			}
		}
	}
	return "provisioning", nil, ""
}

// evaluateCertAttempts reports whether a provisioning domain has exhausted its
// retry budget (N attempts) or its provisioning window (T time).
func evaluateCertAttempts(now time.Time, st certState, verifiedAt *time.Time) (bool, string) {
	if st.Attempts >= certMaxAttempts {
		return true, certFailureReason(st.LastError,
			fmt.Sprintf("certificate provisioning failed after %d attempts", st.Attempts))
	}
	anchor := st.StartedAt
	if anchor == nil {
		anchor = verifiedAt
	}
	if anchor != nil && now.Sub(*anchor) >= certStaleAfter {
		return true, certFailureReason(st.LastError,
			fmt.Sprintf("certificate provisioning timed out after %s", certStaleAfter))
	}
	return false, ""
}

func certFailureReason(lastError, fallback string) string {
	if strings.TrimSpace(lastError) != "" {
		return lastError
	}
	return fallback
}

// recordCertAttempt persists one provisioning observation and advances the
// retry window. The WHERE clause makes the observation atomic across concurrent
// list calls, so only one caller can consume a retry window. cert_started_at is
// only set once so the T timeout is measured from the beginning of the
// provisioning window, not from the latest poll. It reports whether a row was
// actually updated.
func (s *Sites) recordCertAttempt(ctx context.Context, siteID, domain string, now, next time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE site_domains
		    SET cert_attempts = cert_attempts + 1,
		        cert_last_attempt_at = ?,
		        cert_retry_after = ?,
		        cert_started_at = COALESCE(cert_started_at, ?)
		  WHERE site_id = ? AND domain = ?
		    AND (cert_retry_after IS NULL OR cert_retry_after <= ?)`,
		now.Format(time.RFC3339), next.Format(time.RFC3339), now.Format(time.RFC3339),
		siteID, domain, now.Format(time.RFC3339))
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func parseRFC3339Ptr(v *string) *time.Time {
	if v == nil || *v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil {
		return nil
	}
	return &t
}

// certStatus maps a certificate expiry to a display status + ISO expiry string.
func certStatus(notAfter time.Time, haveCert bool, now time.Time) (string, *string) {
	if !haveCert {
		return "provisioning", nil
	}
	iso := notAfter.UTC().Format(time.RFC3339)
	switch {
	case now.After(notAfter):
		return "expired", &iso
	case notAfter.Sub(now) < certWarnWindow:
		return "expiring_soon", &iso
	default:
		return "valid", &iso
	}
}

func (s *Sites) registerDomain(w http.ResponseWriter, r *http.Request, siteID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "domain is required"})
		return
	}
	if !domainRE.MatchString(domain) {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain format"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Guard against attaching domains to non-existent sites (no FK on the table).
	if !s.siteExists(ctx, siteID) {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}

	id, err := kernel.GenerateID()
	if err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	verifyToken := hex.EncodeToString(tokenBytes)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO site_domains (id, site_id, domain, verify_token, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, siteID, domain, verifyToken, now)
	if err != nil {
		if isUniqueViolation(err) {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "domain already registered for this site"})
			return
		}
		s.logger.Error("insert domain", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, SiteDomain{
		ID:          id,
		SiteID:      siteID,
		Domain:      domain,
		VerifyToken: verifyToken,
		CreatedAt:   now,
	})
}

func (s *Sites) siteExists(ctx context.Context, siteID string) bool {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sites WHERE id = ?`, siteID).Scan(&one)
	return err == nil
}

func (s *Sites) deleteDomain(w http.ResponseWriter, r *http.Request, siteID, domain string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.db.ExecContext(ctx,
		`DELETE FROM site_domains WHERE site_id = ? AND domain = ?`,
		siteID, domain)
	if err != nil {
		s.logger.Error("delete domain", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	// Stop routing live traffic for the removed domain. Without this the proxy
	// keeps the host registered (and serving its cert) until the next redeploy.
	if s.unregisterHost != nil {
		s.unregisterHost(domain)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Sites) verifyDomain(w http.ResponseWriter, r *http.Request, siteID, domain string) {
	if !s.verifyLim.allow(siteID+"|"+domain, time.Now()) {
		w.Header().Set("Retry-After", "5")
		kernel.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many verification attempts, try again shortly"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var verifyToken string
	err := s.db.QueryRowContext(ctx,
		`SELECT verify_token FROM site_domains WHERE site_id = ? AND domain = ?`,
		siteID, domain).Scan(&verifyToken)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "domain not registered"})
		return
	}

	expected := fmt.Sprintf("bigbase-verify=%s", verifyToken)
	verified := checkDNSTXT(ctx, domain, expected)

	if verified {
		now := time.Now().UTC().Format(time.RFC3339)
		_, _ = s.db.ExecContext(ctx,
			`UPDATE site_domains
			    SET verified_at = ?,
			        cert_started_at = COALESCE(cert_started_at, ?)
			  WHERE site_id = ? AND domain = ?`,
			now, now, siteID, domain)
		// Activate routing immediately rather than waiting for the next deploy.
		if s.activateDomain != nil {
			if err := s.activateDomain(ctx, siteID, domain); err != nil {
				s.logger.Warn("activate custom domain", "domain", domain, "error", err)
			}
		}
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"domain":       domain,
		"verified":     verified,
		"verify_token": verifyToken,
		"txt_record":   expected,
	})
}

// retryDomainCert resets the certificate attempt budget for a verified domain
// and asks the proxy to re-register the host so issuance can be retried. It is
// the manual escape hatch for a domain stuck in cert_status=failed.
func (s *Sites) retryDomainCert(w http.ResponseWriter, r *http.Request, siteID, domain string) {
	if !s.verifyLim.allow("cert|"+siteID+"|"+domain, time.Now()) {
		w.Header().Set("Retry-After", "5")
		kernel.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many retry attempts, try again shortly"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var verifiedAt *string
	err := s.db.QueryRowContext(ctx,
		`SELECT verified_at FROM site_domains WHERE site_id = ? AND domain = ?`,
		siteID, domain).Scan(&verifiedAt)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "domain not registered"})
		return
	}
	if verifiedAt == nil {
		kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "domain is not verified"})
		return
	}

	now := time.Now().UTC()
	retryAfter := now.Add(certRetryAfter)
	_, err = s.db.ExecContext(ctx,
		`UPDATE site_domains
		    SET cert_attempts = 0,
		        cert_started_at = ?,
		        cert_last_attempt_at = ?,
		        cert_last_error = '',
		        cert_retry_after = ?
		  WHERE site_id = ? AND domain = ?`,
		now.Format(time.RFC3339), now.Format(time.RFC3339), retryAfter.Format(time.RFC3339), siteID, domain)
	if err != nil {
		s.logger.Error("reset cert attempts", "site_id", siteID, "domain", domain, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Re-register the host with the proxy so a previously failed domain is
	// served again and autocert gets a fresh issuance opportunity.
	if s.activateDomain != nil {
		if err := s.activateDomain(ctx, siteID, domain); err != nil {
			s.logger.Warn("activate custom domain after cert retry", "domain", domain, "error", err)
		}
	}

	status := "provisioning"
	if s.certInfo != nil {
		if notAfter, ok := s.certInfo(ctx, domain); ok {
			status, _ = certStatus(notAfter, true, now)
		}
	}

	kernel.WriteJSON(w, http.StatusOK, CertRetryResult{
		Domain:     domain,
		CertStatus: status,
		Attempts:   0,
		RetryAfter: retryAfter.Format(time.RFC3339),
	})
}

// checkDNSTXT looks up TXT records for the domain and returns true if expected is found.
func checkDNSTXT(ctx context.Context, domain, expected string) bool {
	records, err := net.DefaultResolver.LookupTXT(ctx, domain)
	if err != nil {
		return false
	}
	for _, r := range records {
		if r == expected {
			return true
		}
	}
	return false
}

// isUniqueViolation returns true if err is a SQLite UNIQUE constraint error.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
