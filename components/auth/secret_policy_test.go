package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/kernel"
)

// policyFixture boots db + projects + auth so the projects, orgs, and
// org_members tables exist. auth.Start also creates the audit table used by
// the REST adapter, and the secrets schema is not needed for policy tests.
func policyFixture(t *testing.T) (*db.DB, *auth.Auth, *projects.Projects) {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	p := projects.New(projects.Options{DB: d, Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(p)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("start kernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return d, a, p
}

// seedOrg creates an organization owned by ownerID and one project in it.
func seedOrg(t *testing.T, a *auth.Auth, p *projects.Projects, name string, ownerID int64) (int64, string) {
	t.Helper()
	org, err := a.CreateOrg(context.Background(), name, name, ownerID)
	if err != nil {
		t.Fatalf("create org %s: %v", name, err)
	}
	ctx := auth.WithOrgID(context.Background(), org.ID)
	proj, err := p.CreateProject(ctx, "Payments")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return org.ID, proj.ID
}

func TestSecretPolicyAuthorization(t *testing.T) {
	d, a, p := policyFixture(t)
	orgID, projectID := seedOrg(t, a, p, "acme", 42)
	base := auth.WithOrgID(auth.WithUserID(context.Background(), 42), orgID)
	policy := auth.NewSecretPolicy(d)

	cases := []struct {
		name   string
		role   auth.ProjectRole
		action auth.SecretAction
		deny   bool
	}{
		// org_admin: full matrix.
		{name: "org_admin describe", role: auth.RoleOrgAdmin, action: auth.SecretActionDescribe},
		{name: "org_admin read value", role: auth.RoleOrgAdmin, action: auth.SecretActionReadValue},
		{name: "org_admin create", role: auth.RoleOrgAdmin, action: auth.SecretActionCreate},
		{name: "org_admin update", role: auth.RoleOrgAdmin, action: auth.SecretActionUpdate},
		{name: "org_admin delete", role: auth.RoleOrgAdmin, action: auth.SecretActionDelete},
		{name: "org_admin versions", role: auth.RoleOrgAdmin, action: auth.SecretActionListVersions},
		// project_operator: full matrix.
		{name: "operator describe", role: auth.RoleProjectOperator, action: auth.SecretActionDescribe},
		{name: "operator read value", role: auth.RoleProjectOperator, action: auth.SecretActionReadValue},
		{name: "operator create", role: auth.RoleProjectOperator, action: auth.SecretActionCreate},
		{name: "operator update", role: auth.RoleProjectOperator, action: auth.SecretActionUpdate},
		{name: "operator delete", role: auth.RoleProjectOperator, action: auth.SecretActionDelete},
		{name: "operator versions", role: auth.RoleProjectOperator, action: auth.SecretActionListVersions},
		// project_member: describe and versions only.
		{name: "member describe", role: auth.RoleProjectMember, action: auth.SecretActionDescribe},
		{name: "member versions", role: auth.RoleProjectMember, action: auth.SecretActionListVersions},
		{name: "member cannot read value", role: auth.RoleProjectMember, action: auth.SecretActionReadValue, deny: true},
		{name: "member cannot create", role: auth.RoleProjectMember, action: auth.SecretActionCreate, deny: true},
		{name: "member cannot update", role: auth.RoleProjectMember, action: auth.SecretActionUpdate, deny: true},
		{name: "member cannot delete", role: auth.RoleProjectMember, action: auth.SecretActionDelete, deny: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := auth.WithSecretProjectRole(base, tt.role)
			err := policy.Authorize(ctx, projectID, tt.action)
			if tt.deny {
				if !errors.Is(err, auth.ErrSecretActionForbidden) {
					t.Fatalf("expected forbidden, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected allow, got %v", err)
			}
		})
	}
}

func TestSecretPolicyCrossOrgNonDisclosing(t *testing.T) {
	d, a, p := policyFixture(t)
	_, projectA := seedOrg(t, a, p, "acme", 42)
	orgB, _ := seedOrg(t, a, p, "globex", 43)
	ctxB := auth.WithOrgID(auth.WithUserID(context.Background(), 43), orgB)
	policy := auth.NewSecretPolicy(d)

	// A caller from org B targeting org A's project must see exactly the same
	// sentinel as a genuinely missing project — existence is not disclosed.
	if err := policy.Authorize(ctxB, projectA, auth.SecretActionDescribe); !errors.Is(err, auth.ErrSecretProjectNotFound) {
		t.Fatalf("cross-org: expected ErrSecretProjectNotFound, got %v", err)
	}
	if err := policy.Authorize(ctxB, "missing-project", auth.SecretActionDescribe); !errors.Is(err, auth.ErrSecretProjectNotFound) {
		t.Fatalf("missing: expected ErrSecretProjectNotFound, got %v", err)
	}
	// The same applies to value reads and mutations.
	if err := policy.Authorize(ctxB, projectA, auth.SecretActionReadValue); !errors.Is(err, auth.ErrSecretProjectNotFound) {
		t.Fatalf("cross-org value read: expected ErrSecretProjectNotFound, got %v", err)
	}
	if err := policy.Authorize(ctxB, projectA, auth.SecretActionCreate); !errors.Is(err, auth.ErrSecretProjectNotFound) {
		t.Fatalf("cross-org create: expected ErrSecretProjectNotFound, got %v", err)
	}
}

func TestSecretPolicyRequiresOrganization(t *testing.T) {
	d, a, p := policyFixture(t)
	orgID, projectID := seedOrg(t, a, p, "acme", 42)
	policy := auth.NewSecretPolicy(d)

	// Authenticated user without an organization: organization required.
	ctxNoOrg := auth.WithUserID(context.Background(), 42)
	if err := policy.Authorize(ctxNoOrg, projectID, auth.SecretActionDescribe); !errors.Is(err, auth.ErrSecretOrganizationRequired) {
		t.Fatalf("expected ErrSecretOrganizationRequired, got %v", err)
	}

	// Owner with org in context succeeds.
	ctx := auth.WithOrgID(auth.WithUserID(context.Background(), 42), orgID)
	if err := policy.Authorize(ctx, projectID, auth.SecretActionDescribe); err != nil {
		t.Fatalf("owner describe: expected allow, got %v", err)
	}
}

func TestSecretPolicyOrgMembershipResolution(t *testing.T) {
	d, a, p := policyFixture(t)
	orgID, projectID := seedOrg(t, a, p, "acme", 42)

	// Org owner resolves to org_admin: can read values.
	owner := auth.WithOrgID(auth.WithUserID(context.Background(), 42), orgID)
	policy := auth.NewSecretPolicy(d)
	if err := policy.Authorize(owner, projectID, auth.SecretActionReadValue); err != nil {
		t.Fatalf("owner read value: expected allow, got %v", err)
	}

	// An org member with role "member" resolves to project_operator: full.
	memberID := int64(100)
	insertOrgMember(t, d, orgID, memberID, "member")
	member := auth.WithOrgID(auth.WithUserID(context.Background(), memberID), orgID)
	if err := policy.Authorize(member, projectID, auth.SecretActionReadValue); err != nil {
		t.Fatalf("member read value: expected allow, got %v", err)
	}
	if err := policy.Authorize(member, projectID, auth.SecretActionCreate); err != nil {
		t.Fatalf("member create: expected allow, got %v", err)
	}

	// An org member with role "admin" resolves to org_admin: full.
	adminID := int64(101)
	insertOrgMember(t, d, orgID, adminID, "admin")
	admin := auth.WithOrgID(auth.WithUserID(context.Background(), adminID), orgID)
	if err := policy.Authorize(admin, projectID, auth.SecretActionReadValue); err != nil {
		t.Fatalf("admin member read value: expected allow, got %v", err)
	}

	// A user with no membership row is denied, not disclosed.
	stranger := auth.WithOrgID(auth.WithUserID(context.Background(), 999), orgID)
	if err := policy.Authorize(stranger, projectID, auth.SecretActionDescribe); !errors.Is(err, auth.ErrSecretActionForbidden) {
		t.Fatalf("non-member: expected forbidden, got %v", err)
	}

	// An org API key (org in context, no user) resolves to project_operator.
	keyCtx := auth.WithOrgID(context.Background(), orgID)
	if err := policy.Authorize(keyCtx, projectID, auth.SecretActionReadValue); err != nil {
		t.Fatalf("org key read value: expected allow, got %v", err)
	}
}

func TestSecretPolicyUnknownRoleFailsClosed(t *testing.T) {
	d, a, p := policyFixture(t)
	orgID, projectID := seedOrg(t, a, p, "acme", 42)
	base := auth.WithOrgID(auth.WithUserID(context.Background(), 42), orgID)
	policy := auth.NewSecretPolicy(d)

	ctx := auth.WithSecretProjectRole(base, auth.ProjectRole("superadmin"))
	if err := policy.Authorize(ctx, projectID, auth.SecretActionDescribe); !errors.Is(err, auth.ErrSecretActionForbidden) {
		t.Fatalf("unknown role: expected forbidden, got %v", err)
	}
}

func TestSecretPolicyNilPolicyFailsClosed(t *testing.T) {
	_, a, p := policyFixture(t)
	orgID, projectID := seedOrg(t, a, p, "acme", 42)
	ctx := auth.WithOrgID(auth.WithUserID(context.Background(), 42), orgID)

	var policy *auth.SecretPolicy
	if err := policy.Authorize(ctx, projectID, auth.SecretActionDescribe); !errors.Is(err, auth.ErrSecretActionForbidden) {
		t.Fatalf("nil policy: expected forbidden, got %v", err)
	}
}

func insertOrgMember(t *testing.T, d *db.DB, orgID, userID int64, role string) {
	t.Helper()
	if _, err := d.Exec(`INSERT INTO org_members (org_id, user_id, role, joined_at) VALUES (?, ?, ?, datetime('now'))`,
		orgID, userID, role); err != nil {
		t.Fatalf("insert org member: %v", err)
	}
}
