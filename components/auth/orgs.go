package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Organization represents a tenant organization.
type Organization struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	OwnerID   int64  `json:"owner_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateOrg creates a new organization owned by ownerID.
// Returns the new Organization or an error if the slug already exists.
func (a *Auth) CreateOrg(ctx context.Context, name, slug string, ownerID int64) (*Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		`INSERT INTO orgs (name, slug, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		name, slug, ownerID, now, now)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return a.GetOrgByID(ctx, id)
}

// GetOrgByID returns an organization by its ID, or nil if not found.
func (a *Auth) GetOrgByID(ctx context.Context, id int64) (*Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var org Organization
	err := a.db.QueryRowContext(ctx,
		`SELECT id, name, slug, owner_id, created_at, updated_at FROM orgs WHERE id = ?`, id,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}

	return &org, nil
}

// ListOrgsByOwner returns all organizations owned by the given user.
func (a *Auth) ListOrgsByOwner(ctx context.Context, ownerID int64) ([]Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := a.db.QueryContext(ctx,
		`SELECT id, name, slug, owner_id, created_at, updated_at FROM orgs WHERE owner_id = ? ORDER BY id`,
		ownerID)
	if err != nil {
		return nil, fmt.Errorf("list orgs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	orgs := make([]Organization, 0)
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.OwnerID, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan org: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// UpdateOrg updates an organization's name and slug.
// Returns an error if the slug is already taken by another organization.
func (a *Auth) UpdateOrg(ctx context.Context, id int64, name, slug string) (*Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		`UPDATE orgs SET name = ?, slug = ?, updated_at = ? WHERE id = ?`,
		name, slug, now, id)
	if err != nil {
		return nil, fmt.Errorf("update org: %w", err)
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("org %d not found", id)
	}

	return a.GetOrgByID(ctx, id)
}

// DeleteOrg deletes an organization by ID.
func (a *Auth) DeleteOrg(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := a.db.ExecContext(ctx, `DELETE FROM orgs WHERE id = ?`, id)
	return err
}
