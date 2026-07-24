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
	return s.db.Migrate(domainsMigration)
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
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Sites) listDomains(w http.ResponseWriter, r *http.Request, siteID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, site_id, domain, verify_token, verified_at, created_at
		 FROM site_domains WHERE site_id = ? ORDER BY created_at`, siteID)
	if err != nil {
		s.logger.Error("list domains", "site_id", siteID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	domains := make([]SiteDomain, 0)
	for rows.Next() {
		var d SiteDomain
		var verifiedAt *string
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Domain, &d.VerifyToken, &verifiedAt, &d.CreatedAt); err != nil {
			s.logger.Error("scan domain", "error", err)
			continue
		}
		d.VerifiedAt = verifiedAt
		domains = append(domains, d)
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate domains", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Annotate verified domains with live certificate status from the proxy.
	for i := range domains {
		if domains[i].VerifiedAt == nil {
			continue
		}
		status, expiresAt := s.certStatusFor(ctx, domains[i].Domain)
		domains[i].CertStatus = status
		domains[i].CertExpiresAt = expiresAt
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": domains})
}

// certStatusFor resolves the certificate status for a verified domain via the
// optional CertInfo hook. Returns "provisioning" when no cert is cached yet.
func (s *Sites) certStatusFor(ctx context.Context, domain string) (string, *string) {
	if s.certInfo == nil {
		return "", nil
	}
	notAfter, ok := s.certInfo(ctx, domain)
	return certStatus(notAfter, ok, time.Now())
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
			`UPDATE site_domains SET verified_at = ? WHERE site_id = ? AND domain = ?`,
			now, siteID, domain)
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
