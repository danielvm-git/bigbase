package secrets_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) Debug(string, ...any) {}

// testRootKey is a canonical base64 32-byte root key shared by the fixtures.
func testRootKey(t *testing.T) []byte {
	t.Helper()
	raw := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	key, err := secrets.ParseRootKey(raw)
	if err != nil {
		t.Fatalf("parse root key: %v", err)
	}
	return key
}

// startKernel registers db, projects, and secrets in dependency order and
// starts them. dsn selects the SQLite target (":memory:" or a file URI).
func startKernel(t *testing.T, dsn string) (*db.DB, *secrets.Secrets, *projects.Projects) {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: dsn, Logger: logger})
	s, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: testRootKey(t)})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	p := projects.New(projects.Options{DB: d, Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(p)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("start kernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return d, s, p
}

// secretsFixture uses an in-memory database.
func secretsFixture(t *testing.T) (*db.DB, *secrets.Secrets, *projects.Projects) {
	t.Helper()
	return startKernel(t, ":memory:")
}

// fileFixture uses a file-backed database with foreign keys and a busy
// timeout enabled, which is required by the schema-constraint and race tests.
func fileFixture(t *testing.T) (*db.DB, *secrets.Secrets, *projects.Projects, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secrets.db")
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)"
	d, s, p := startKernel(t, dsn)
	return d, s, p, path
}

// createScope creates a project, production environment, and folder owned by
// orgID, returning the authenticated context to use against them.
func createScope(t *testing.T, p *projects.Projects, s *secrets.Secrets, orgID int64) (context.Context, string, string, string) {
	t.Helper()
	ctx := auth.WithOrgID(context.Background(), orgID)
	proj, err := p.CreateProject(ctx, "Payments")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	env, err := p.CreateEnvironment(ctx, proj.ID, "production", "Production")
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	folder, err := s.EnsureFolder(ctx, proj.ID, env.ID, "default")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	return ctx, proj.ID, env.ID, folder.ID
}

func TestSecretsImplementsPublicSeams(t *testing.T) {
	var _ kernel.Component = (*secrets.Secrets)(nil)
	var _ secrets.SecretManager = (*secrets.Secrets)(nil)
	// The optional transaction seam is implemented by both drivers.
	var _ kernel.TxBeginner = (*db.DB)(nil)
	var _ kernel.TxBeginner = (*db.PostgresDB)(nil)
}

func TestSecretsRequiresValidRootKey(t *testing.T) {
	d, _, _ := secretsFixture(t)
	for _, tt := range []struct {
		name string
		key  []byte
	}{
		{name: "nil root key", key: nil},
		{name: "16-byte root key", key: bytes.Repeat([]byte{0x11}, 16)},
		{name: "33-byte root key", key: bytes.Repeat([]byte{0x11}, 33)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := secrets.New(secrets.Options{DB: d, Logger: testLogger{}, RootKey: tt.key}); err == nil {
				t.Fatalf("New accepted an invalid root key")
			}
		})
	}
}

func TestSecretsParseRootKeyContract(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, 32))
	key, err := secrets.ParseRootKey(raw)
	if err != nil || len(key) != 32 {
		t.Fatalf("valid canonical root key rejected: len=%d err=%v", len(key), err)
	}
	for _, bad := range []string{"", "not-base64", "AA=="} {
		if _, err := secrets.ParseRootKey(bad); err == nil {
			t.Fatalf("ParseRootKey(%q) accepted invalid configuration", bad)
		}
	}
}

func TestSecretsStartupDependencyOnProjects(t *testing.T) {
	// secrets declares a kernel dependency on projects; the kernel refuses to
	// start when projects is not registered, so Secret tables can never be
	// created before Project tables.
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	s, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: testRootKey(t)})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	k := kernel.New(logger)
	k.Register(d)
	k.Register(s)
	if err := k.Start(); err == nil {
		t.Fatal("kernel started secrets without the projects dependency")
	}
}
