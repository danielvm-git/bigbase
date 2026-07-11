package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// OrgAPIKey holds metadata about an org-scoped API key.
// The raw key is only available at creation time; stored as a hash.
type OrgAPIKey struct {
	ID         int64    `json:"id"`
	OrgID      int64    `json:"org_id"`
	Name       string   `json:"name"`
	Scopes     []string `json:"scopes"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

// OrgAPIKeyCreated is the response for a new key — includes the raw key once.
type OrgAPIKeyCreated struct {
	OrgAPIKey
	Key string `json:"key"`
}

// migrateAPIKeysTable ensures the org_api_keys table exists.
func (a *Auth) migrateAPIKeysTable() error {
	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS org_api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id INTEGER NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		scopes TEXT NOT NULL DEFAULT '',
		last_used_at TEXT,
		created_at TEXT NOT NULL
	)`); err != nil {
		return err
	}
	return a.ensureSiteKeyColumns()
}

func (a *Auth) ensureSiteKeyColumns() error {
	for _, stmt := range []string{
		`ALTER TABLE org_api_keys ADD COLUMN site_id TEXT NULL`,
		`ALTER TABLE org_api_keys ADD COLUMN revoked INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE org_api_keys ADD COLUMN prefix TEXT DEFAULT ''`,
	} {
		if err := a.db.Migrate(stmt); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				continue
			}
			return err
		}
	}
	return nil
}

// CreateAPIKey generates a new API key for the org, stores its hash, and returns the raw key once.
func (a *Auth) CreateAPIKey(ctx context.Context, orgID int64, name string, scopes []string) (*OrgAPIKeyCreated, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	raw, err := generateRawAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	keyHash := hashAPIKey(raw)
	scopesStr := joinScopes(scopes)
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := a.db.ExecContext(ctx,
		`INSERT INTO org_api_keys (org_id, key_hash, name, scopes, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		orgID, keyHash, name, scopesStr, now)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("api key last insert id: %w", err)
	}

	return &OrgAPIKeyCreated{
		OrgAPIKey: OrgAPIKey{
			ID:        id,
			OrgID:     orgID,
			Name:      name,
			Scopes:    scopes,
			CreatedAt: now,
		},
		Key: raw,
	}, nil
}

// ListAPIKeys returns all API keys for an org (without raw keys).
func (a *Auth) ListAPIKeys(ctx context.Context, orgID int64) ([]OrgAPIKey, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := a.db.QueryContext(ctx,
		`SELECT id, org_id, name, scopes, last_used_at, created_at
		 FROM org_api_keys WHERE org_id = ? ORDER BY id`,
		orgID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	keys := make([]OrgAPIKey, 0)
	for rows.Next() {
		var k OrgAPIKey
		var scopesStr string
		var lastUsedAt sql.NullString
		if err := rows.Scan(&k.ID, &k.OrgID, &k.Name, &scopesStr, &lastUsedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		k.Scopes = splitScopes(scopesStr)
		if lastUsedAt.Valid {
			k.LastUsedAt = &lastUsedAt.String
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeleteAPIKey removes an API key by ID, scoped to the org.
func (a *Auth) DeleteAPIKey(ctx context.Context, orgID, keyID int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := a.db.ExecContext(ctx,
		`DELETE FROM org_api_keys WHERE id = ? AND org_id = ?`,
		keyID, orgID)
	return err
}

// ResolveOrgKey resolves a bb_ organization API key to org_id and scopes.
// Site deploy keys (bb_dep_) and revoked keys are rejected.
func (a *Auth) ResolveOrgKey(rawKey string) (int64, []string, error) {
	if !strings.HasPrefix(rawKey, "bb_") || strings.HasPrefix(rawKey, "bb_dep_") {
		return 0, nil, fmt.Errorf("invalid org api key")
	}

	keyHash := hashAPIKey(rawKey)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var orgID int64
	var keyID int64
	var scopesStr string
	err := a.db.QueryRowContext(ctx,
		`SELECT id, org_id, scopes FROM org_api_keys
		 WHERE key_hash = ? AND site_id IS NULL AND revoked = 0`,
		keyHash,
	).Scan(&keyID, &orgID, &scopesStr)
	if err == sql.ErrNoRows {
		return 0, nil, fmt.Errorf("api key not found")
	}
	if err != nil {
		return 0, nil, fmt.Errorf("resolve org key: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = a.db.ExecContext(context.Background(),
		`UPDATE org_api_keys SET last_used_at = ? WHERE id = ?`, now, keyID)

	return orgID, splitScopes(scopesStr), nil
}

// ResolveAPIKey looks up the org_id for a raw API key. Returns an error when not found.
func (a *Auth) ResolveAPIKey(rawKey string) (int64, error) {
	keyHash := hashAPIKey(rawKey)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var orgID int64
	var keyID int64
	err := a.db.QueryRowContext(ctx,
		`SELECT id, org_id FROM org_api_keys WHERE key_hash = ? AND revoked = 0`, keyHash,
	).Scan(&keyID, &orgID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("api key not found")
	}
	if err != nil {
		return 0, fmt.Errorf("resolve api key: %w", err)
	}

	// Update last_used_at asynchronously (best-effort, no error propagation)
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = a.db.ExecContext(context.Background(),
		`UPDATE org_api_keys SET last_used_at = ? WHERE id = ?`, now, keyID)

	return orgID, nil
}

// SiteKeyCreated is the response for a new site-scoped deployment key.
type SiteKeyCreated struct {
	ID     int64  `json:"id"`
	SiteID string `json:"site_id"`
	Name   string `json:"name"`
	Key    string `json:"-"`
}

// SiteKeyEntry is metadata for a site-scoped deploy key (no raw secret).
type SiteKeyEntry struct {
	KeyID      string  `json:"key_id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	CreatedAt  string  `json:"created_at"`
	Revoked    bool    `json:"revoked"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

// CreateSiteKey creates a site-scoped deployment key — returns raw token once.
func (a *Auth) CreateSiteKey(ctx context.Context, siteID, name string, scopes []string) (rawToken, keyID string, err error) {
	created, err := a.createSiteKeyRecord(ctx, siteID, name, scopes)
	if err != nil {
		return "", "", err
	}
	return created.Key, fmt.Sprintf("%d", created.ID), nil
}

// createSiteKeyRecord generates a site-scoped deployment key (bb_dep_ prefix).
func (a *Auth) createSiteKeyRecord(ctx context.Context, siteID, name string, scopes []string) (*SiteKeyCreated, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if siteID == "" {
		return nil, fmt.Errorf("site_id is required")
	}
	if name == "" {
		name = "ci-bot"
	}

	var exists int
	if err := a.db.QueryRowContext(ctx, "SELECT 1 FROM sites WHERE id = ?", siteID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("site %q not found", siteID)
		}
		return nil, fmt.Errorf("validate site: %w", err)
	}

	raw, err := generateSiteKey()
	if err != nil {
		return nil, fmt.Errorf("generate site key: %w", err)
	}

	prefix := rawPrefix(raw)
	keyHash := hashAPIKey(raw)
	scopesStr := joinScopes(scopes)
	if scopesStr == "" {
		scopesStr = "deploy"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := a.db.ExecContext(ctx,
		`INSERT INTO org_api_keys (org_id, site_id, key_hash, name, scopes, created_at, revoked, prefix)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		0, siteID, keyHash, name, scopesStr, now, prefix)
	if err != nil {
		return nil, fmt.Errorf("insert site key: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("site key last insert id: %w", err)
	}

	a.logger.Info("site key created", "key_id", id, "site_id", siteID)

	return &SiteKeyCreated{
		ID:     id,
		SiteID: siteID,
		Name:   name,
		Key:    raw,
	}, nil
}

// ListSiteKeys returns metadata for site-scoped keys (no raw secrets).
func (a *Auth) ListSiteKeys(ctx context.Context, siteID string) ([]SiteKeyEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if siteID == "" {
		return nil, fmt.Errorf("site_id is required")
	}
	var exists int
	if err := a.db.QueryRowContext(ctx, "SELECT 1 FROM sites WHERE id = ?", siteID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("site %q not found", siteID)
		}
		return nil, fmt.Errorf("validate site: %w", err)
	}

	rows, err := a.db.QueryContext(ctx,
		`SELECT id, name, prefix, created_at, revoked, last_used_at
		 FROM org_api_keys WHERE site_id = ? ORDER BY id`,
		siteID)
	if err != nil {
		return nil, fmt.Errorf("list site keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	keys := make([]SiteKeyEntry, 0)
	for rows.Next() {
		var id int64
		var name, prefix, createdAt string
		var revoked int
		var lastUsed sql.NullString
		if err := rows.Scan(&id, &name, &prefix, &createdAt, &revoked, &lastUsed); err != nil {
			return nil, fmt.Errorf("scan site key: %w", err)
		}
		entry := SiteKeyEntry{
			KeyID:     fmt.Sprintf("%d", id),
			Name:      name,
			Prefix:    prefix,
			CreatedAt: createdAt,
			Revoked:   revoked != 0,
		}
		if lastUsed.Valid {
			entry.LastUsedAt = &lastUsed.String
		}
		keys = append(keys, entry)
	}
	return keys, rows.Err()
}

// RevokeSiteKey marks a site-scoped deploy key as revoked.
func (a *Auth) RevokeSiteKey(ctx context.Context, keyID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if keyID == "" {
		return fmt.Errorf("key_id is required")
	}
	id, err := strconv.ParseInt(keyID, 10, 64)
	if err != nil {
		return fmt.Errorf("key %q not found", keyID)
	}

	res, err := a.db.ExecContext(ctx,
		`UPDATE org_api_keys SET revoked = 1 WHERE id = ? AND site_id IS NOT NULL`,
		id)
	if err != nil {
		return fmt.Errorf("revoke site key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke site key rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("key %q not found", keyID)
	}
	a.logger.Info("site key revoked", "key_id", id)
	return nil
}

// ResolveSiteKey looks up the site_id for a bb_dep_ prefixed key.
func (a *Auth) ResolveSiteKey(rawKey string) (string, error) {
	keyHash := hashAPIKey(rawKey)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var siteID sql.NullString
	var keyID int64
	err := a.db.QueryRowContext(ctx,
		`SELECT id, site_id FROM org_api_keys WHERE key_hash = ? AND revoked = 0 AND site_id IS NOT NULL`,
		keyHash,
	).Scan(&keyID, &siteID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("site key not found")
	}
	if err != nil {
		return "", fmt.Errorf("resolve site key: %w", err)
	}
	if !siteID.Valid || siteID.String == "" {
		return "", fmt.Errorf("site key not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = a.db.ExecContext(context.Background(),
		`UPDATE org_api_keys SET last_used_at = ? WHERE id = ?`, now, keyID)

	return siteID.String, nil
}

// generateSiteKey creates a site-scoped key with bb_dep_ prefix.
func generateSiteKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "bb_dep_" + hex.EncodeToString(b), nil
}

// rawPrefix returns the first 13 characters of a bb_dep_ key (prefix + first 8 hex chars).
func rawPrefix(raw string) string {
	if len(raw) > 13 {
		return raw[:13]
	}
	return raw
}

// generateRawAPIKey creates a 32-byte hex-encoded random key prefixed with "bb_".
func generateRawAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "bb_" + hex.EncodeToString(b), nil
}

// hashAPIKey produces a SHA-256 hex digest of the raw key.
func hashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// joinScopes converts a scopes slice to a comma-separated string.
func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}

// splitScopes converts a comma-separated scopes string to a slice.
func splitScopes(s string) []string {
	if s == "" {
		return []string{}
	}
	var scopes []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				scopes = append(scopes, s[start:i])
			}
			start = i + 1
		}
	}
	return scopes
}
