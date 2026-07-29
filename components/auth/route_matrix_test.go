package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

// This file implements the cross-tenant route matrix guard requested by
// issue #180 ("Multi-tenant IDOR fixes are per-handler, not structural").
//
// WHY THIS EXISTS
//
// Tenancy in BigBase is enforced per-handler, by hand, at the SQL string level
// (e.g. `WHERE s.org_id = ?` plus a requireSiteOwnership / verifySiteOwnership
// ownership probe). Two of those one-handler-at-a-time fixes already shipped
// critical production regressions (deploy-key 403, sites-invisible via
// DEFAULT 0). Every new org-scoped handler is therefore a fresh chance to
// silently reintroduce an IDOR.
//
// This test is a STRUCTURAL guard against that class. It seeds two orgs (A and
// B), each owning an identical-shaped set of tenant resources, then for every
// org-scoped route in the matrix table below it asserts two things:
//
//  1. CROSS-TENANT DENIAL — a caller authenticated as org A targeting org B's
//     resource id is denied (403 or 404). This is the IDOR guard: if a handler
//     forgets the org_id check, org A reaches org B's data and this fails.
//  2. SAME-TENANT SUCCESS — a caller authenticated as org A targeting org A's
//     own resource succeeds (2xx). This locks in that the denial is a real
//     ownership check, not an accidental blanket 403 that would regress
//     legitimate access (exactly the deploy-key 403 failure mode).
//
// The matrix is a declarative table. Adding a new org-scoped route is a
// ONE-LINE table entry — see the comment on routeMatrixCase below. The test is
// table-driven so the cost of keeping it current stays at one line per route.
//
// The test passes against current main because every route it lists already
// has an ownership check. Its value is FORWARD: any future org-scoped route
// added without an ownership check, and added to this table (which it must be,
// by convention), will fail CI. A route that touches tenant data but is missing
// from this table is a review smell — reviewers should ask "is this in the
// route matrix?".

// matrixOrgA / matrixOrgB are the two org ids used by the matrix. They are
// deliberately nonzero so they cannot be confused with the legacy org_id=0
// bootstrap carve-out (which is admin-only and out of scope here — see
// sites/org_scoping_test.go).
const (
	matrixOrgA = int64(100)
	matrixOrgB = int64(200)
)

// routeMatrixCase is one row of the cross-tenant route matrix.
//
// ADDING A NEW ORG-SCOPED ROUTE: add one row to routeMatrixCases. Fill in:
//
//   - name:         human label, used in the sub-test name.
//   - method:       HTTP method.
//   - pathTemplate: the route with the tenant path parameter replaced by %s
//                   (e.g. "/api/sites/%s" or "/api/deploy/%s/logs"). At run time
//                   %s is substituted with the target resource id.
//   - handler:      which component handler serves the route
//                   (matrixHandlerSites / matrixHandlerDeploy / matrixHandlerAuth).
//   - targetIsSite: true when %s is a SITE id, false when it is a DEPLOYMENT id
//                   (used to pick which seeded id to substitute for A and B).
//   - skipOwn:      set true ONLY for destructive/mutating routes (DELETE,
//                   POST that creates a row) where the own-org success case
//                   would mutate the shared fixture and destabilise other rows.
//                   Cross-tenant denial is still asserted. Prefer leaving false.
//
// A route that touches tenant data MUST appear in routeMatrixCases. If you
// cannot express your route here, that is a signal the matrix needs extending,
// not a reason to skip coverage.
type routeMatrixCase struct {
	name         string
	method       string
	pathTemplate string // %s = tenant resource id (site or deployment)
	handler      matrixHandler
	targetIsSite bool // true: %s is a site id; false: %s is a deployment id
	skipOwn      bool // true: skip the same-tenant success assertion (destructive)
}

// matrixHandler selects which component handler serves a given route.
type matrixHandler int

const (
	matrixHandlerSites matrixHandler = iota // sites.Sites.Handler()
	matrixHandlerDeploy                     // deploy.Deploy.Handler()
	matrixHandlerAuth                       // auth.Auth.ProtectedHandler() (deploy-key routes)
)

// routeMatrixCases is the authoritative list of org-scoped routes. New routes
// that take an org/site id path parameter and touch tenant data go here.
func routeMatrixCases() []routeMatrixCase {
	return []routeMatrixCase{
		// --- Sites component (gated by Sites.requireSiteOwnership) ---
		{
			name:         "sites/get-by-id",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s",
			handler:      matrixHandlerSites,
			targetIsSite: true,
		},
		{
			name:         "sites/get-manifest",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s/manifest",
			handler:      matrixHandlerSites,
			targetIsSite: true,
		},
		{
			name:         "sites/get-auth-policy",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s/auth-policy",
			handler:      matrixHandlerSites,
			targetIsSite: true,
		},
		{
			name:         "sites/get-deploy-defaults",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s/deploy-defaults",
			handler:      matrixHandlerSites,
			targetIsSite: true,
		},
		{
			name:         "sites/list-env-vars",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s/env-vars",
			handler:      matrixHandlerSites,
			targetIsSite: true,
		},
		{
			name:         "sites/list-domains",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s/domains",
			handler:      matrixHandlerSites,
			targetIsSite: true,
		},
		{
			name:         "sites/delete",
			method:       http.MethodDelete,
			pathTemplate: "/api/sites/%s",
			handler:      matrixHandlerSites,
			targetIsSite: true,
			skipOwn:      true, // destructive — cross-tenant denial only
		},

		// --- Deploy component (gated by verifyDeploymentOwnership / verifySiteOwnership) ---
		{
			name:         "deploy/get-by-id",
			method:       http.MethodGet,
			pathTemplate: "/api/deploy/%s",
			handler:      matrixHandlerDeploy,
			targetIsSite: false,
		},
		{
			name:         "deploy/get-logs",
			method:       http.MethodGet,
			pathTemplate: "/api/deploy/%s/logs",
			handler:      matrixHandlerDeploy,
			targetIsSite: false,
		},
		{
			name:         "deploy/get-rollback-events",
			method:       http.MethodGet,
			pathTemplate: "/api/deploy/%s/rollback-events",
			handler:      matrixHandlerDeploy,
			// rollback-events is keyed by SITE id, not deployment id.
			targetIsSite: true,
		},
		{
			name:         "deploy/delete",
			method:       http.MethodDelete,
			pathTemplate: "/api/deploy/%s",
			handler:      matrixHandlerDeploy,
			targetIsSite: false,
			skipOwn:      true, // destructive — cross-tenant denial only
		},

		// --- Auth component: deploy-key routes (gated by lookupOwnedOrg) ---
		{
			name:         "site-keys/list",
			method:       http.MethodGet,
			pathTemplate: "/api/sites/%s/deploy-keys",
			handler:      matrixHandlerAuth,
			targetIsSite: true,
		},
		{
			name:         "site-keys/create",
			method:       http.MethodPost,
			pathTemplate: "/api/sites/%s/deploy-keys",
			handler:      matrixHandlerAuth,
			targetIsSite: true,
			skipOwn:      true, // mutates key table — cross-tenant denial only
		},
		{
			name:         "site-keys/revoke",
			method:       http.MethodDelete,
			pathTemplate: "/api/sites/%s/deploy-keys/KEYID",
			handler:      matrixHandlerAuth,
			targetIsSite: true,
			skipOwn:      true, // destructive — cross-tenant denial only
		},
	}
}

// matrixEnv bundles the handlers and seeded resource ids for one matrix run.
type matrixEnv struct {
	sitesHandler  http.Handler
	deployHandler http.Handler
	authHandler   http.Handler

	siteA, siteB string // seeded site ids per org
	depA, depB   string // seeded deployment ids per org
	bKeyID       string // a deploy key id registered on org B's site (for revoke)
	tokenA       string // real JWT for org A's owner user (for deploy-key routes)
}

// handlerFor returns the component handler that serves the given matrixHandler.
func (e *matrixEnv) handlerFor(h matrixHandler) http.Handler {
	switch h {
	case matrixHandlerSites:
		return e.sitesHandler
	case matrixHandlerDeploy:
		return e.deployHandler
	case matrixHandlerAuth:
		return e.authHandler
	default:
		return nil
	}
}

// targetID picks the seeded resource id to substitute into the path template
// for the requested org, accounting for whether the path param is a site or
// deployment id.
func (e *matrixEnv) targetID(c routeMatrixCase, orgID int64) string {
	isOrgA := orgID == matrixOrgA
	switch {
	case c.targetIsSite && isOrgA:
		return e.siteA
	case c.targetIsSite:
		return e.siteB
	case isOrgA:
		return e.depA
	default:
		return e.depB
	}
}

// setupRouteMatrix boots a full auth + sites + deploy stack on one in-memory DB
// and seeds two orgs (A=100, B=200), each owning one site and one deployment,
// plus one deploy key on org B's site for the revoke cross-tenant case. All
// seeding bypasses the HTTP layer (raw SQL) so the matrix starts from a known,
// deterministic tenant layout.
func setupRouteMatrix(t *testing.T) *matrixEnv {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	a.SetOTPStore(auth.NewMapOTPStore())
	a.SetRateLimitStore(auth.NewMapRateLimitStore())
	a.SetLoginLockoutStore(auth.NewMapLoginLockoutStore())
	// Both git and sites point at the same temp dir so the manifest handler
	// (which reads <gitDir>/<repoID>.git off disk) can find the seeded repos.
	gitDir := t.TempDir()
	g := git.New(git.Options{DB: d, Logger: logger, Dir: gitDir})
	s := sites.New(sites.Options{DB: d, Logger: logger, GitDir: gitDir})
	dep := deploy.New(deploy.Options{
		DB:        d,
		Logger:    logger,
		BuildsDir: t.TempDir(),
		GitDir:    t.TempDir(),
	})

	k.Register(d)
	k.Register(a)
	k.Register(g)
	k.Register(s)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	ctx := context.Background()

	// Seed two orgs owned by two users so lookupOwnedOrg (used by deploy-key
	// routes) has a real owner to verify against.
	if _, err := d.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, role, default_org_id, created_at) VALUES
			(1001, 'a@matrix.test', 'x', 'user', ?, datetime('now')),
			(2001, 'b@matrix.test', 'x', 'user', ?, datetime('now'))`,
		matrixOrgA, matrixOrgB); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := d.ExecContext(ctx,
		`INSERT INTO orgs (id, name, slug, owner_id, created_at, updated_at) VALUES
			(?, 'Org A', 'org-a', 1001, datetime('now'), datetime('now')),
			(?, 'Org B', 'org-b', 2001, datetime('now'), datetime('now'))`,
		matrixOrgA, matrixOrgB); err != nil {
		t.Fatalf("seed orgs: %v", err)
	}

	// Seed one git repo per org (sites reference git_repos.id). Also materialise
	// the bare repo directories the manifest handler expects on disk
	// (<gitDir>/<repoID>.git) so the own-org manifest read succeeds.
	if _, err := d.ExecContext(ctx,
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES
			('repo-a', 'repo-a', 1001, 0, 'main', '', datetime('now')),
			('repo-b', 'repo-b', 2001, 0, 'main', '', datetime('now'))`); err != nil {
		t.Fatalf("seed git_repos: %v", err)
	}
	for _, repoID := range []string{"repo-a", "repo-b"} {
		if err := seedBareRepoForMatrix(t, gitDir, repoID); err != nil {
			t.Fatalf("seed bare repo %s: %v", repoID, err)
		}
	}

	siteA, siteB := "site-org-a", "site-org-b"
	if _, err := d.ExecContext(ctx,
		`INSERT INTO sites (id, name, git_repo_id, org_id) VALUES
			(?, 'Site A', 'repo-a', ?),
			(?, 'Site B', 'repo-b', ?)`,
		siteA, matrixOrgA, siteB, matrixOrgB); err != nil {
		t.Fatalf("seed sites: %v", err)
	}

	depA, depB := "dep-org-a", "dep-org-b"
	if _, err := d.ExecContext(ctx,
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, created_at) VALUES
			(?, 'repo-a', ?, 'main', 'running', datetime('now')),
			(?, 'repo-b', ?, 'main', 'running', datetime('now'))`,
		depA, siteA, depB, siteB); err != nil {
		t.Fatalf("seed deployments: %v", err)
	}

	// Seed one deploy key on org B's site for the revoke cross-tenant case.
	// Deploy keys live in org_api_keys (site-scoped via site_id). key_id is the
	// autoincrement id, which we capture so the revoke path targets it.
	var bKeyID string
	res, err := d.ExecContext(ctx,
		`INSERT INTO org_api_keys (org_id, site_id, key_hash, name, scopes, created_at, revoked, prefix)
		 VALUES (?, ?, 'hash-b', 'ci-bot', '', datetime('now'), 0, 'bb_dep_')`,
		matrixOrgB, siteB)
	if err != nil {
		t.Fatalf("seed site key on org B: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("site key last insert id: %v", err)
	}
	bKeyID = strconvI64(id)

	// Mint a real JWT for org A's owner user — the realistic production attack
	// path for deploy-key IDOR is an authenticated user, and the deploy-key
	// handlers read UserIDFromContext (set only by JWT middleware, not by the
	// raw WithOrgID test helper).
	tokenA, err := auth.CreateJWTForTesting(a, 1001, "a@matrix.test", "user", matrixOrgA)
	if err != nil {
		t.Fatalf("mint matrix JWT: %v", err)
	}

	return &matrixEnv{
		sitesHandler:  s.Handler(),
		deployHandler: dep.Handler(),
		authHandler:   a.ProtectedHandler(),
		siteA:         siteA,
		siteB:         siteB,
		depA:          depA,
		depB:          depB,
		bKeyID:        bKeyID,
		tokenA:        tokenA,
	}
}

// matrixRequest builds a request authenticated as the given org, targeting the
// given (substituted) path.
//
// For sites/deploy handlers we inject org_id directly into the context (mirrors
// the existing deploy/sites IDOR test idiom and isolates the handler-level
// ownership check — the structural target of #180). For the auth deploy-key
// handler we send a real JWT bearer so middleware populates user_id, which the
// handler needs for lookupOwnedOrg.
func matrixRequest(c routeMatrixCase, method, path, tokenA string, orgID int64) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if c.handler == matrixHandlerAuth {
		req.Header.Set("Authorization", "Bearer "+tokenA)
		return req
	}
	return req.WithContext(auth.WithOrgID(req.Context(), orgID))
}

// resolveMatrixPath substitutes the tenant resource id into the path template's
// first (and only) %s placeholder. For the revoke route, %s is the site id and
// the literal "KEYID" is then replaced with the seeded key id.
func resolveMatrixPath(c routeMatrixCase, id, bKeyID string) string {
	out := c.pathTemplate
	// Substitute the first %s manually (avoid fmt.Sprintf interpreting other %).
	for i := 0; i < len(out)-1; i++ {
		if out[i] == '%' && out[i+1] == 's' {
			out = out[:i] + id + out[i+2:]
			break
		}
	}
	return strings.ReplaceAll(out, "KEYID", bKeyID)
}

// TestRouteMatrix_CrossTenantDenial is the IDOR guard. For every org-scoped
// route in routeMatrixCases it asserts an org-A caller CANNOT reach org-B's
// resource, and (unless skipOwn) CAN reach org-A's own resource.
func TestRouteMatrix_CrossTenantDenial(t *testing.T) {
	env := setupRouteMatrix(t)

	for _, c := range routeMatrixCases() {
		c := c // capture for t.Run closure
		t.Run(c.name+"/cross_tenant_denied", func(t *testing.T) {
			targetPath := resolveMatrixPath(c, env.targetID(c, matrixOrgB), env.bKeyID)
			handler := env.handlerFor(c.handler)
			if handler == nil {
				t.Fatalf("no handler for route %s", c.name)
			}
			req := matrixRequest(c, c.method, targetPath, env.tokenA, matrixOrgA)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
				t.Fatalf("IDOR: org A (id=%d) reached org B (id=%d) resource via %s %s: expected 403/404, got %d (body=%s).\n"+
					"This route likely lacks an org_id ownership check. Add one (requireSiteOwnership / verifySiteOwnership / lookupOwnedOrg), "+
					"or, if denial is intentionally handled elsewhere, document why in the route matrix table.",
					matrixOrgA, matrixOrgB, c.method, targetPath, w.Code, w.Body.String())
			}
		})

		if c.skipOwn {
			continue
		}
		t.Run(c.name+"/same_tenant_allowed", func(t *testing.T) {
			targetPath := resolveMatrixPath(c, env.targetID(c, matrixOrgA), env.bKeyID)
			handler := env.handlerFor(c.handler)
			if handler == nil {
				t.Fatalf("no handler for route %s", c.name)
			}
			req := matrixRequest(c, c.method, targetPath, env.tokenA, matrixOrgA)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code < http.StatusOK || w.Code >= 300 {
				t.Fatalf("org A (id=%d) denied its OWN resource via %s %s: expected 2xx, got %d (body=%s).\n"+
					"A real ownership check must still allow the legitimate owner — this is the deploy-key 403 regression class.",
					matrixOrgA, c.method, targetPath, w.Code, w.Body.String())
			}
		})
	}
}

// TestRouteMatrix_DeployListScopedByOrg locks in the list-scoping path that does
// NOT take a path id: GET /api/deploy returns ONLY the caller's org
// deployments. This is a separate case because it has no path id to substitute
// and asserts on response body contents rather than a single 403.
func TestRouteMatrix_DeployListScopedByOrg(t *testing.T) {
	env := setupRouteMatrix(t)

	// Org A lists deployments.
	req := matrixRequest(routeMatrixCase{handler: matrixHandlerDeploy},
		http.MethodGet, "/api/deploy", env.tokenA, matrixOrgA)
	w := httptest.NewRecorder()
	env.deployHandler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list deployments: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	assertDeployListScoped(t, w.Body.Bytes(), env.depA, env.depB)
}

// TestRouteMatrix_SitesListScopedByOrg is the sites counterpart: GET /api/sites
// returns ONLY the caller's org sites.
func TestRouteMatrix_SitesListScopedByOrg(t *testing.T) {
	env := setupRouteMatrix(t)

	req := matrixRequest(routeMatrixCase{handler: matrixHandlerSites},
		http.MethodGet, "/api/sites", env.tokenA, matrixOrgA)
	w := httptest.NewRecorder()
	env.sitesHandler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list sites: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	assertSitesListScoped(t, w.Body.Bytes(), env.siteA, env.siteB)
}

// assertDeployListScoped fails if body contains the other-org deployment id or
// omits the caller's own.
func assertDeployListScoped(t *testing.T, body []byte, ownDep, otherDep string) {
	t.Helper()
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode deploy list: %v (body=%s)", err, body)
	}
	seen := make(map[string]bool, len(resp.Data))
	for _, d := range resp.Data {
		seen[d.ID] = true
	}
	if !seen[ownDep] {
		t.Errorf("deploy list missing caller's own deployment %q (got %v) — list scoping too strict", ownDep, seen)
	}
	if seen[otherDep] {
		t.Errorf("IDOR: deploy list leaked other-org deployment %q (got %v) — list scoping missing org_id filter", otherDep, seen)
	}
}

// assertSitesListScoped fails if body contains the other-org site id or omits
// the caller's own.
func assertSitesListScoped(t *testing.T, body []byte, ownSite, otherSite string) {
	t.Helper()
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode sites list: %v (body=%s)", err, body)
	}
	seen := make(map[string]bool, len(resp.Data))
	for _, s := range resp.Data {
		seen[s.ID] = true
	}
	if !seen[ownSite] {
		t.Errorf("sites list missing caller's own site %q (got %v) — list scoping too strict", ownSite, seen)
	}
	if seen[otherSite] {
		t.Errorf("IDOR: sites list leaked other-org site %q (got %v) — list scoping missing org_id filter", otherSite, seen)
	}
}

// seedBareRepoForMatrix creates a bare git repo at <gitDir>/<repoID>.git with
// an initial commit on `main` containing an empty bigbase.yaml. This lets the
// manifest handler read its own-org repo successfully (it shells out to
// `git show main:bigbase.yaml`). It mirrors the idiom in
// components/sites/sites_test.go (TestSiteManifestGetAndSave).
func seedBareRepoForMatrix(t *testing.T, gitDir, repoID string) error {
	t.Helper()
	barePath := filepath.Join(gitDir, repoID+".git")
	if err := os.MkdirAll(barePath, 0o755); err != nil {
		return err
	}
	if err := exec.Command("git", "init", "--bare", barePath).Run(); err != nil {
		return err
	}
	clone := t.TempDir()
	if err := exec.Command("git", "clone", barePath, clone).Run(); err != nil {
		return err
	}
	gitCfg := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = clone
		_ = c.Run()
	}
	gitCfg("config", "user.name", "Matrix Test")
	gitCfg("config", "user.email", "matrix@test")
	if err := os.WriteFile(filepath.Join(clone, "bigbase.yaml"), []byte(""), 0o644); err != nil {
		return err
	}
	gitCfg("add", "bigbase.yaml")
	gitCfg("commit", "-m", "init")
	gitCfg("branch", "-M", "main")
	gitCfg("push", "origin", "main")
	return nil
}

// strconvI64 is a thin wrapper kept local to avoid pulling in strconv at the
// call sites above. Returns the decimal string of an int64.
func strconvI64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
