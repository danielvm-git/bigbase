package deploy_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// fakeSecretResolver is a minimal in-memory SecretResolver for resolver unit
// tests. It never touches a database and keeps deterministic ordering.
type fakeSecretResolver struct {
	folders []secrets.SecretFolder
	// keys maps folderID to ordered secret keys.
	keys map[string][]string
	// values maps "folderID/key" to the value result.
	values map[string]secrets.SecretValue
	// readErr, when set, is returned by every ReadSecretValue call.
	readErr error
}

func (f *fakeSecretResolver) ListFolders(context.Context, string, string) ([]secrets.SecretFolder, error) {
	return f.folders, nil
}

func (f *fakeSecretResolver) ListSecrets(_ context.Context, _, _, folderID string) ([]secrets.SecretMetadata, error) {
	var out []secrets.SecretMetadata
	for _, k := range f.keys[folderID] {
		out = append(out, secrets.SecretMetadata{Key: k})
	}
	return out, nil
}

func (f *fakeSecretResolver) ReadSecretValue(_ context.Context, _, _, folderID, key string) (secrets.SecretValue, error) {
	if f.readErr != nil {
		return secrets.SecretValue{}, f.readErr
	}
	v, ok := f.values[folderID+"/"+key]
	if !ok {
		return secrets.SecretValue{}, errors.New("secret not found")
	}
	return v, nil
}

// resolveEnvProject sets up an in-memory DB with sites/projects/environments
// tables plus the given legacy site vars, and returns a resolver backed by the
// fake secret resolver. siteProjectID attaches the site to the seeded project;
// empty leaves the site unattached (legacy-only mode).
func resolveEnvProject(t *testing.T, fake deploy.SecretResolver, siteID, siteProjectID string, vars [][4]any) *deploy.EnvResolver {
	t.Helper()
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	migrateEnvVarsTable(t, database)
	migrateProjectTables(t, database)
	if _, err := database.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, org_id, project_id) VALUES (?, 'test-site', 1, ?)`,
		siteID, sqlNullString(siteProjectID)); err != nil {
		t.Fatalf("insert site: %v", err)
	}

	for _, v := range vars {
		key, value := v[0].(string), v[1].(string)
		bt, rt := v[2].(bool), v[3].(bool)
		insertEnvVar(t, database, siteID, key, value, bt, rt)
	}

	dep := deploy.New(deploy.Options{DB: database, Logger: logger, Secrets: fake})
	_ = dep.Start(&kernel.Context{})
	t.Cleanup(func() { _ = dep.Stop(&kernel.Context{}) })
	return dep.EnvResolver()
}

// migrateProjectTables creates the s02/s03 metadata tables the resolver reads.
func migrateProjectTables(t *testing.T, database *db.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			git_repo_id TEXT NOT NULL DEFAULT '',
			org_id INTEGER NOT NULL DEFAULT 0,
			project_id TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			org_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS project_environments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, s := range stmts {
		if _, err := database.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("migrate project tables: %v", err)
		}
	}
	if _, err := database.ExecContext(context.Background(),
		`INSERT OR IGNORE INTO projects (id, org_id, name) VALUES ('proj-1', 1, 'Test Project')`); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if _, err := database.ExecContext(context.Background(),
		`INSERT OR IGNORE INTO project_environments (id, project_id, slug, name) VALUES ('env-1', 'proj-1', 'production', 'Production')`); err != nil {
		t.Fatalf("insert environment: %v", err)
	}
}

func sqlNullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func TestEnvResolverProjectAndSitePrecedence(t *testing.T) {
	// SC-e89s06-P0-01: Site compatibility values override Project values on
	// key collision, and the collision is reported without values.
	fake := &fakeSecretResolver{
		folders: []secrets.SecretFolder{{ID: "folder-1", Name: "default", ProjectID: "proj-1", EnvironmentID: "env-1"}},
		keys:    map[string][]string{"folder-1": {"DB_URL", "API_TOKEN"}},
		values: map[string]secrets.SecretValue{
			"folder-1/DB_URL":    {Key: "DB_URL", Value: "project"},
			"folder-1/API_TOKEN": {Key: "API_TOKEN", Value: "proj-token"},
		},
	}
	r := resolveEnvProject(t, fake, "site-x", "proj-1", [][4]any{
		{"DB_URL", "site", false, true},
	})

	resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeRuntime)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	m := toMap(resolved.Environ)
	if m["DB_URL"] != "site" {
		t.Fatalf("site value must override project value on collision, got %q", m["DB_URL"])
	}
	if m["API_TOKEN"] != "proj-token" {
		t.Fatalf("project-only secret missing, got %q", m["API_TOKEN"])
	}
	conflicts := resolved.Conflicts()
	if len(conflicts) != 1 || conflicts[0] != "DB_URL" {
		t.Fatalf("expected DB_URL conflict reported, got %v", conflicts)
	}
}

func TestEnvResolverFullPrecedenceChain(t *testing.T) {
	// platform baseline < manifest config < Project secret < Site
	// compatibility < reserved runtime values.
	t.Setenv("SHARED", "platform")
	fake := &fakeSecretResolver{
		folders: []secrets.SecretFolder{{ID: "folder-1", Name: "default", ProjectID: "proj-1", EnvironmentID: "env-1"}},
		keys:    map[string][]string{"folder-1": {"SHARED", "PROJECT_ONLY"}},
		values: map[string]secrets.SecretValue{
			"folder-1/SHARED":       {Key: "SHARED", Value: "project"},
			"folder-1/PROJECT_ONLY": {Key: "PROJECT_ONLY", Value: "from-project"},
		},
	}
	r := resolveEnvProject(t, fake, "site-x", "proj-1", [][4]any{
		{"SHARED", "site", false, true},
		{"SITE_ONLY", "from-site", false, true},
	})

	resolved, err := r.ResolveOptions(context.Background(), "site-x", deploy.ScopeRuntime, deploy.ResolveOptions{
		ManifestEnv: map[string]string{
			"SHARED":        "manifest",
			"MANIFEST_ONLY": "from-manifest",
		},
		Reserved: []string{"SHARED=reserved", "PORT=7777"},
	})
	if err != nil {
		t.Fatalf("ResolveOptions: %v", err)
	}
	m := toMap(resolved.Environ)
	if m["SHARED"] != "reserved" {
		t.Fatalf("reserved value must win over every layer, got %q", m["SHARED"])
	}
	if m["PORT"] != "7777" {
		t.Fatalf("reserved PORT missing, got %q", m["PORT"])
	}
	if m["SITE_ONLY"] != "from-site" {
		t.Fatalf("site layer missing, got %q", m["SITE_ONLY"])
	}
	if m["PROJECT_ONLY"] != "from-project" {
		t.Fatalf("project layer missing, got %q", m["PROJECT_ONLY"])
	}
	if m["MANIFEST_ONLY"] != "from-manifest" {
		t.Fatalf("manifest layer missing, got %q", m["MANIFEST_ONLY"])
	}
}

func TestEnvResolverManifestNeverOverridesSecret(t *testing.T) {
	// Manifest config sits below Project secrets: a secret must override a
	// manifest default for the same key.
	fake := &fakeSecretResolver{
		folders: []secrets.SecretFolder{{ID: "folder-1", Name: "default", ProjectID: "proj-1", EnvironmentID: "env-1"}},
		keys:    map[string][]string{"folder-1": {"NODE_ENV"}},
		values: map[string]secrets.SecretValue{
			"folder-1/NODE_ENV": {Key: "NODE_ENV", Value: "production"},
		},
	}
	r := resolveEnvProject(t, fake, "site-x", "proj-1", nil)
	resolved, err := r.ResolveOptions(context.Background(), "site-x", deploy.ScopeRuntime, deploy.ResolveOptions{
		ManifestEnv: map[string]string{"NODE_ENV": "development"},
	})
	if err != nil {
		t.Fatalf("ResolveOptions: %v", err)
	}
	if got := toMap(resolved.Environ)["NODE_ENV"]; got != "production" {
		t.Fatalf("project secret must override manifest config, got %q", got)
	}
}

func TestEnvResolverProjectSecretScopeAgnostic(t *testing.T) {
	// Native Project secrets carry no build/runtime flag: they appear in both
	// scopes, while legacy scope flags keep filtering the Site layer.
	fake := &fakeSecretResolver{
		folders: []secrets.SecretFolder{{ID: "folder-1", Name: "default", ProjectID: "proj-1", EnvironmentID: "env-1"}},
		keys:    map[string][]string{"folder-1": {"PROJECT_VAR"}},
		values: map[string]secrets.SecretValue{
			"folder-1/PROJECT_VAR": {Key: "PROJECT_VAR", Value: "native"},
		},
	}
	r := resolveEnvProject(t, fake, "site-x", "proj-1", [][4]any{
		{"BUILD_ONLY", "b", true, false},
		{"RUNTIME_ONLY", "r", false, true},
	})

	buildEnv, err := r.Resolve(context.Background(), "site-x", deploy.ScopeBuild)
	if err != nil {
		t.Fatalf("Resolve build: %v", err)
	}
	if _, ok := toMap(buildEnv.Environ)["RUNTIME_ONLY"]; ok {
		t.Fatal("runtime-only legacy value leaked into build scope")
	}
	if toMap(buildEnv.Environ)["PROJECT_VAR"] != "native" {
		t.Fatal("project secret missing from build scope")
	}

	rtEnv, err := r.Resolve(context.Background(), "site-x", deploy.ScopeRuntime)
	if err != nil {
		t.Fatalf("Resolve runtime: %v", err)
	}
	if _, ok := toMap(rtEnv.Environ)["BUILD_ONLY"]; ok {
		t.Fatal("build-only legacy value leaked into runtime scope")
	}
	if toMap(rtEnv.Environ)["PROJECT_VAR"] != "native" {
		t.Fatal("project secret missing from runtime scope")
	}
}

func TestEnvResolverProjectDecryptionFailureFatal(t *testing.T) {
	// A selected Project secret that cannot be decrypted stops resolution:
	// the app must never boot without a value it was supposed to receive.
	fake := &fakeSecretResolver{
		folders: []secrets.SecretFolder{{ID: "folder-1", Name: "default", ProjectID: "proj-1", EnvironmentID: "env-1"}},
		keys:    map[string][]string{"folder-1": {"BROKEN"}},
		values:  map[string]secrets.SecretValue{},
		readErr: errors.New("decryption failed"),
	}
	r := resolveEnvProject(t, fake, "site-x", "proj-1", nil)
	if _, err := r.Resolve(context.Background(), "site-x", deploy.ScopeRuntime); err == nil {
		t.Fatal("project decryption failure was silently ignored")
	}
}

func TestEnvResolverNoProjectAttachmentLegacyOnly(t *testing.T) {
	// A site without a project attachment resolves through the legacy Site
	// layer only — native dual-read must not break pre-migration sites.
	r := resolveEnvProject(t, nil, "site-legacy", "", [][4]any{
		{"LEGACY_VAR", "legacy-value", false, true},
	})
	resolved, err := r.Resolve(context.Background(), "site-legacy", deploy.ScopeRuntime)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := toMap(resolved.Environ)["LEGACY_VAR"]; got != "legacy-value" {
		t.Fatalf("legacy value missing, got %q", got)
	}
	if len(resolved.SecretKeys()) != 1 || !resolved.IsSecret("LEGACY_VAR") {
		t.Fatalf("legacy value must be treated as a secret, got %v", resolved.SecretKeys())
	}
}

func TestEnvResolverMissingProjectColumnDegrades(t *testing.T) {
	// A sites table that predates the project_id attachment column must not
	// break resolution — legacy-only behavior applies.
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	_ = database.Start(&kernel.Context{})
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	migrateEnvVarsTable(t, database)
	if _, err := database.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		org_id INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("create legacy sites table: %v", err)
	}
	if _, err := database.ExecContext(context.Background(),
		`INSERT INTO sites (id, name, org_id) VALUES ('site-old', 'old', 1)`); err != nil {
		t.Fatalf("insert legacy site: %v", err)
	}
	insertEnvVar(t, database, "site-old", "OLD_VAR", "old-value", false, true)

	dep := deploy.New(deploy.Options{DB: database, Logger: logger})
	_ = dep.Start(&kernel.Context{})
	t.Cleanup(func() { _ = dep.Stop(&kernel.Context{}) })

	resolved, err := dep.EnvResolver().Resolve(context.Background(), "site-old", deploy.ScopeRuntime)
	if err != nil {
		t.Fatalf("Resolve on pre-attachment schema: %v", err)
	}
	if got := toMap(resolved.Environ)["OLD_VAR"]; got != "old-value" {
		t.Fatalf("legacy value missing on pre-attachment schema, got %q", got)
	}
}

func TestEnvResolverProjectAndSiteRedaction(t *testing.T) {
	// Both Project and Site values must be redacted from env views and free
	// log text (SC-e89s06-P0-03).
	fake := &fakeSecretResolver{
		folders: []secrets.SecretFolder{{ID: "folder-1", Name: "default", ProjectID: "proj-1", EnvironmentID: "env-1"}},
		keys:    map[string][]string{"folder-1": {"DB_PASSWORD", "PROJECT_TOKEN"}},
		values: map[string]secrets.SecretValue{
			"folder-1/DB_PASSWORD":   {Key: "DB_PASSWORD", Value: "proj-super-secret"},
			"folder-1/PROJECT_TOKEN": {Key: "PROJECT_TOKEN", Value: "proj-token-9f3a"},
		},
	}
	r := resolveEnvProject(t, fake, "site-x", "proj-1", [][4]any{
		{"SITE_API_KEY", "site-key-hunter2", false, true},
	})

	resolved, err := r.Resolve(context.Background(), "site-x", deploy.ScopeRuntime)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	redacted := deploy.Redact(resolved.Environ, resolved.SecretKeys())
	joined := strings.Join(redacted, "\n")
	for _, secret := range []string{"proj-super-secret", "proj-token-9f3a", "site-key-hunter2"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("plaintext %q present in redacted env", secret)
		}
	}

	rawText := strings.Join(resolved.Environ, "\n") + "\n" + "tool echoed proj-super-secret and site-key-hunter2"
	logged := deploy.RedactLogText(rawText, resolved)
	if strings.Contains(logged, "proj-super-secret") || strings.Contains(logged, "site-key-hunter2") || strings.Contains(logged, "proj-token-9f3a") {
		t.Fatalf("secret leaked into redacted log text:\n%s", logged)
	}
}
