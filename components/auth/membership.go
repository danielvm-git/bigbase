package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// OrgInvite represents a pending invitation to join an organization.
type OrgInvite struct {
	ID            int64   `json:"id"`
	OrgID         int64   `json:"org_id"`
	InviteeEmail  string  `json:"invitee_email"`
	Role          string  `json:"role"`
	Token         string  `json:"token"`
	AcceptedAt    *string `json:"accepted_at,omitempty"`
	ExpiresAt     string  `json:"expires_at"`
}

// OrgMember represents a user who has accepted membership in an org.
type OrgMember struct {
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	JoinedAt  string `json:"joined_at"`
}

// validMemberRoles lists the allowed roles for org membership.
var validMemberRoles = map[string]bool{
	"admin":  true,
	"member": true,
}

// migrateInvitesTable creates the org_invites and org_members tables if they don't exist.
func (a *Auth) migrateInvitesTable() error {
	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS org_invites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id INTEGER NOT NULL,
		invitee_email TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'member',
		token TEXT NOT NULL UNIQUE,
		accepted_at TEXT,
		expires_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("migrate org_invites: %w", err)
	}
	// Partial unique index: only one pending invite per (org, email)
	if err := a.db.Migrate(`CREATE UNIQUE INDEX IF NOT EXISTS idx_org_invites_pending
		ON org_invites (org_id, invitee_email)
		WHERE accepted_at IS NULL`); err != nil {
		return fmt.Errorf("migrate org_invites index: %w", err)
	}
	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS org_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL DEFAULT 'member',
		joined_at TEXT NOT NULL,
		UNIQUE(org_id, user_id)
	)`); err != nil {
		return fmt.Errorf("migrate org_members: %w", err)
	}
	return nil
}

// CreateInvite inserts a new invite into org_invites and returns it.
// Returns an error with "UNIQUE constraint" if a pending invite already exists for that email+org.
func (a *Auth) CreateInvite(ctx context.Context, orgID int64, inviteeEmail, role string) (*OrgInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	token, err := generateInviteToken()
	if err != nil {
		return nil, fmt.Errorf("generate invite token: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour).Format(time.RFC3339)

	res, err := a.db.ExecContext(ctx,
		`INSERT INTO org_invites (org_id, invitee_email, role, token, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		orgID, inviteeEmail, role, token, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("invite last insert id: %w", err)
	}

	return &OrgInvite{
		ID:           id,
		OrgID:        orgID,
		InviteeEmail: inviteeEmail,
		Role:         role,
		Token:        token,
		ExpiresAt:    expiresAt,
	}, nil
}

// GetInviteByToken fetches a pending (not yet accepted) invite by token.
// Returns nil if not found or already accepted.
func (a *Auth) GetInviteByToken(ctx context.Context, orgID int64, token string) (*OrgInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var invite OrgInvite
	var acceptedAt sql.NullString
	err := a.db.QueryRowContext(ctx,
		`SELECT id, org_id, invitee_email, role, token, accepted_at, expires_at
		 FROM org_invites WHERE org_id = ? AND token = ? AND accepted_at IS NULL`,
		orgID, token,
	).Scan(&invite.ID, &invite.OrgID, &invite.InviteeEmail, &invite.Role,
		&invite.Token, &acceptedAt, &invite.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	if acceptedAt.Valid {
		invite.AcceptedAt = &acceptedAt.String
	}
	return &invite, nil
}

// AcceptInvite marks the invite as accepted and adds the user to org_members.
func (a *Auth) AcceptInvite(ctx context.Context, invite *OrgInvite, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339)

	_, err := a.db.ExecContext(ctx,
		`UPDATE org_invites SET accepted_at = ? WHERE id = ?`,
		now, invite.ID)
	if err != nil {
		return fmt.Errorf("mark invite accepted: %w", err)
	}

	_, err = a.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO org_members (org_id, user_id, role, joined_at)
		 VALUES (?, ?, ?, ?)`,
		invite.OrgID, userID, invite.Role, now)
	if err != nil {
		return fmt.Errorf("insert org member: %w", err)
	}

	return nil
}

// ListOrgMembers returns all accepted members of an org.
func (a *Auth) ListOrgMembers(ctx context.Context, orgID int64) ([]OrgMember, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := a.db.QueryContext(ctx,
		`SELECT m.user_id, u.email, m.role, m.joined_at
		 FROM org_members m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.org_id = ?
		 ORDER BY m.joined_at`,
		orgID)
	if err != nil {
		return nil, fmt.Errorf("list org members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	members := make([]OrgMember, 0)
	for rows.Next() {
		var m OrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan org member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// generateInviteToken creates a 32-byte hex-encoded random token.
func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
