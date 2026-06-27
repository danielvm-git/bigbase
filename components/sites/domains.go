package sites

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const domainsMigration = `CREATE TABLE IF NOT EXISTS site_domains (
	id TEXT PRIMARY KEY,
	site_id TEXT NOT NULL,
	domain TEXT NOT NULL UNIQUE,
	verify_token TEXT NOT NULL,
	verified_at TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

type SiteDomain struct {
	ID          string  `json:"id"`
	SiteID      string  `json:"site_id"`
	Domain      string  `json:"domain"`
	VerifyToken string  `json:"verify_token"`
	VerifiedAt  *string `json:"verified_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (s *Sites) migrateDomains() error {
	return s.db.Migrate(domainsMigration)
}

func (s *Sites) handleDomains(w http.ResponseWriter, r *http.Request) {
	// Path: /api/sites/{id}/domains[/{domain}/verify]
	path := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	// path = "{id}/domains" or "{id}/domains/{domain}/verify"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// parts[0] = site id, parts[1] = "domains", parts[2] = domain, parts[3] = "verify"
	if len(parts) < 2 || parts[1] != "domains" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	siteID := parts[0]

	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		s.listDomains(w, r, siteID)
	case len(parts) == 2 && r.Method == http.MethodPost:
		s.registerDomain(w, r, siteID)
	case len(parts) == 3 && r.Method == http.MethodDelete:
		s.deleteDomain(w, r, siteID, parts[2])
	case len(parts) == 4 && parts[3] == "verify" && r.Method == http.MethodGet:
		s.verifyDomain(w, r, siteID, parts[2])
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": domains})
}

func (s *Sites) registerDomain(w http.ResponseWriter, r *http.Request, siteID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain is required"})
		return
	}

	id, err := generateID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	verifyToken := hex.EncodeToString(tokenBytes)
	now := time.Now().UTC().Format(time.RFC3339)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO site_domains (id, site_id, domain, verify_token, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, siteID, req.Domain, verifyToken, now)
	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "domain already registered"})
			return
		}
		s.logger.Error("insert domain", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, SiteDomain{
		ID:          id,
		SiteID:      siteID,
		Domain:      req.Domain,
		VerifyToken: verifyToken,
		CreatedAt:   now,
	})
}

func (s *Sites) deleteDomain(w http.ResponseWriter, r *http.Request, siteID, domain string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.db.ExecContext(ctx,
		`DELETE FROM site_domains WHERE site_id = ? AND domain = ?`,
		siteID, domain)
	if err != nil {
		s.logger.Error("delete domain", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Sites) verifyDomain(w http.ResponseWriter, r *http.Request, siteID, domain string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var verifyToken string
	err := s.db.QueryRowContext(ctx,
		`SELECT verify_token FROM site_domains WHERE site_id = ? AND domain = ?`,
		siteID, domain).Scan(&verifyToken)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not registered"})
		return
	}

	expected := fmt.Sprintf("bigbase-verify=%s", verifyToken)
	verified := checkDNSTXT(ctx, domain, expected)

	if verified {
		now := time.Now().UTC().Format(time.RFC3339)
		_, _ = s.db.ExecContext(ctx,
			`UPDATE site_domains SET verified_at = ? WHERE site_id = ? AND domain = ?`,
			now, siteID, domain)
	}

	writeJSON(w, http.StatusOK, map[string]any{
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
