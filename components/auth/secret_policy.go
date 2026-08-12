package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// secret_policy.go implements the e89s04 secret policy seam: one declarative
// role/action matrix shared by every secret REST route (list, value read,
// mutation, and version listing) so those routes cannot drift in
// authorization behavior. It also carries the first-release "organization
// membership plus Project role" resolution: the authenticated organization
// comes from auth context (never from caller arguments), the target Project
// must belong to that organization (a cross-org target is a non-disclosing
// not-found), and the caller's Project role is derived from org membership
// unless an explicit Project role is present in the request context (the
// future project-scoped membership seam).

// SecretAction names a discrete permission checked against the matrix.
type SecretAction string

const (
	// SecretActionDescribe covers list and single-secret metadata reads.
	SecretActionDescribe SecretAction = "describe_secret"
	// SecretActionReadValue is the explicit value-read permission. It is
	// deliberately separate from SecretActionDescribe so metadata visibility
	// never implies plaintext access.
	SecretActionReadValue SecretAction = "read_secret_value"
	SecretActionCreate    SecretAction = "create_secret"
	SecretActionUpdate    SecretAction = "update_secret"
	SecretActionDelete    SecretAction = "delete_secret"
	// SecretActionListVersions covers immutable version metadata listing. It
	// requires describe standing but is independent from value-read standing.
	SecretActionListVersions SecretAction = "list_secret_versions"
)

// ProjectRole is the caller's role within the target Project.
type ProjectRole string

const (
	RoleOrgAdmin        ProjectRole = "org_admin"
	RoleProjectOperator ProjectRole = "project_operator"
	RoleProjectMember   ProjectRole = "project_member"
)

// Sentinel errors returned by SecretPolicy. They are mapped to HTTP responses
// by the REST adapter; the not-found error is intentionally the same for
// cross-org targets and genuinely missing projects so existence is never
// disclosed (non-disclosing 404).
var (
	ErrSecretOrganizationRequired = errors.New("organization required")
	ErrSecretProjectNotFound      = errors.New("project not found")
	ErrSecretActionForbidden      = errors.New("forbidden")
)

// secretActionMatrix is the first-release policy matrix (e89s04 §13).
//
//	role              describe  read value  mutate  list versions
//	org_admin         yes       yes         yes     yes
//	project_operator  yes       yes         yes     yes
//	project_member    yes       no          no      yes
var secretActionMatrix = map[ProjectRole]map[SecretAction]bool{
	RoleOrgAdmin: {
		SecretActionDescribe:     true,
		SecretActionReadValue:    true,
		SecretActionCreate:       true,
		SecretActionUpdate:       true,
		SecretActionDelete:       true,
		SecretActionListVersions: true,
	},
	RoleProjectOperator: {
		SecretActionDescribe:     true,
		SecretActionReadValue:    true,
		SecretActionCreate:       true,
		SecretActionUpdate:       true,
		SecretActionDelete:       true,
		SecretActionListVersions: true,
	},
	RoleProjectMember: {
		SecretActionDescribe:     true,
		SecretActionListVersions: true,
	},
}

// secretActionAllowed reports whether the matrix permits role to perform
// action. Unknown roles and unknown actions fail closed.
func secretActionAllowed(role ProjectRole, action SecretAction) bool {
	byRole, ok := secretActionMatrix[role]
	if !ok {
		return false
	}
	return byRole[action]
}

type secretPolicyCtxKey string

const ctxSecretProjectRole secretPolicyCtxKey = "secret_project_role"

// WithSecretProjectRole returns a context carrying an explicit Project role.
// It is the extension seam for project-scoped membership: when present, the
// default role resolver honors it before consulting org membership. Tests use
// it to exercise the full role matrix without minting membership rows.
func WithSecretProjectRole(ctx context.Context, role ProjectRole) context.Context {
	return context.WithValue(ctx, ctxSecretProjectRole, role)
}

// SecretProjectRoleFromContext extracts an explicit Project role previously
// set by WithSecretProjectRole. Returns false when no explicit role is set.
func SecretProjectRoleFromContext(ctx context.Context) (ProjectRole, bool) {
	role, ok := ctx.Value(ctxSecretProjectRole).(ProjectRole)
	return role, ok
}

// secretRoleResolver resolves the caller's Project role. It is a function
// field so tests and future membership stores can substitute resolution.
type secretRoleResolver func(ctx context.Context, projectID string) (ProjectRole, error)

// SecretPolicy is the authorization helper shared by every secret REST route.
// Construct one with NewSecretPolicy. It is safe for concurrent use.
type SecretPolicy struct {
	db          DBer
	resolveRole secretRoleResolver
}

// NewSecretPolicy builds a policy over the shared database. The database is
// used for the organization-ownership check and org-membership role
// resolution; it must be the same database the Projects and SecretManager
// components use.
func NewSecretPolicy(db DBer) *SecretPolicy {
	if db == nil {
		return &SecretPolicy{resolveRole: denyRole}
	}
	p := &SecretPolicy{db: db}
	p.resolveRole = p.resolveOrgMembershipRole
	return p
}

// denyRole is the fail-closed resolver used when no database is configured.
func denyRole(context.Context, string) (ProjectRole, error) {
	return "", ErrSecretActionForbidden
}

// Authorize verifies the authenticated caller may perform action against the
// Project identified by projectID. The organization always comes from the
// authenticated context — never from the caller. It returns:
//
//   - ErrSecretOrganizationRequired when no organization is in context
//   - ErrSecretProjectNotFound for cross-organization or missing Projects
//     (non-disclosing: both cases look identical)
//   - ErrSecretActionForbidden when the caller's role lacks the action
//
// The error type alone is never enough to distinguish "exists but forbidden"
// from "does not exist" across organizations, which is what the REST contract
// requires.
func (p *SecretPolicy) Authorize(ctx context.Context, projectID string, action SecretAction) error {
	if p == nil || p.db == nil {
		return ErrSecretActionForbidden
	}
	orgID, ok := OrgIDFromContext(ctx)
	if !ok || orgID <= 0 {
		return ErrSecretOrganizationRequired
	}
	if err := p.requireProjectOwnership(ctx, orgID, projectID); err != nil {
		return err
	}
	role, err := p.resolveRole(ctx, projectID)
	if err != nil {
		return err
	}
	if !secretActionAllowed(role, action) {
		return fmt.Errorf("%w: action %s for role %s", ErrSecretActionForbidden, action, role)
	}
	return nil
}

// requireProjectOwnership returns ErrSecretProjectNotFound when the project
// is missing or belongs to a different organization. The response is
// intentionally identical for both so existence is not disclosed.
func (p *SecretPolicy) requireProjectOwnership(ctx context.Context, orgID int64, projectID string) error {
	var projectOrg int64
	err := p.db.QueryRowContext(ctx,
		`SELECT org_id FROM projects WHERE id = ?`, projectID).Scan(&projectOrg)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: project %s", ErrSecretProjectNotFound, projectID)
	}
	if err != nil {
		return fmt.Errorf("lookup project: %w", err)
	}
	if projectOrg != orgID {
		return fmt.Errorf("%w: project %s", ErrSecretProjectNotFound, projectID)
	}
	return nil
}

// resolveOrgMembershipRole derives the caller's Project role from the
// authenticated organization. Resolution order:
//
//  1. An explicit Project role in context (WithSecretProjectRole) wins — the
//     project-scoped membership seam.
//  2. The organization owner is org_admin.
//  3. An org_members row with role "admin" is org_admin; "member" is
//     project_operator (every org member operates the org's Projects in the
//     first release, matching the s02 project model).
//  4. An org API key (no user identity in context) is project_operator — the
//     organization's automation identity; narrow MCP scope enforcement is the
//     s07 adapter's job.
//  5. Anything else fails closed with ErrSecretActionForbidden.
func (p *SecretPolicy) resolveOrgMembershipRole(ctx context.Context, projectID string) (ProjectRole, error) {
	if role, ok := SecretProjectRoleFromContext(ctx); ok {
		return role, nil
	}
	orgID, ok := OrgIDFromContext(ctx)
	if !ok || orgID <= 0 {
		return "", ErrSecretOrganizationRequired
	}
	userID, hasUser := UserIDFromContext(ctx)
	if !hasUser {
		// Organization API key: automation identity with org-level standing.
		return RoleProjectOperator, nil
	}

	var ownerID int64
	if err := p.db.QueryRowContext(ctx,
		`SELECT owner_id FROM orgs WHERE id = ?`, orgID).Scan(&ownerID); err != nil {
		return "", fmt.Errorf("lookup organization: %w", err)
	}
	if ownerID == userID {
		return RoleOrgAdmin, nil
	}

	var membershipRole string
	err := p.db.QueryRowContext(ctx,
		`SELECT role FROM org_members WHERE org_id = ? AND user_id = ?`,
		orgID, userID).Scan(&membershipRole)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSecretActionForbidden
	}
	if err != nil {
		return "", fmt.Errorf("lookup org membership: %w", err)
	}
	switch membershipRole {
	case "admin":
		return RoleOrgAdmin, nil
	case "member":
		return RoleProjectOperator, nil
	default:
		return "", fmt.Errorf("%w: unknown membership role %q", ErrSecretActionForbidden, membershipRole)
	}
}
