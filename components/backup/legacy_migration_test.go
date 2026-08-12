package backup_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/backup"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/internal/envcrypto"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// migrationStack is a full db + projects + secrets stack with the legacy
// sites/site_env_vars tables available, matching the s02/s03 composition.
type migrationStack struct {
	database *db.DB
	s        *secrets.Secrets
	p        *projects.Projects
}

func legacyMigrationFixture(t *testing.T) *migrationStack {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})

	raw := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	rootKey, err := secrets.ParseRootKey(raw)
	if err != nil {
		t.Fatalf("parse root key: %v", err)
	}
	s, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: rootKey})
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

	// Legacy tables are created after Start (projects.Start tolerates their
	// absence); the sites table carries the project_id attachment column so
	// EnsureSiteProject can update it.
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL DEFAULT '',
		org_id INTEGER NOT NULL DEFAULT 0,
		project_id TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("create sites table: %v", err)
	}
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS site_env_vars (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value_encrypted TEXT NOT NULL DEFAULT '',
		is_build_time INTEGER NOT NULL DEFAULT 0,
		is_runtime INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(site_id, key)
	)`); err != nil {
		t.Fatalf("create site_env_vars table: %v", err)
	}
	return &migrationStack{database: d, s: s, p: p}
}

// seedAttachedSite creates a site row, attaches one compatibility project and
// production environment, and returns the project and environment IDs.
func seedAttachedSite(t *testing.T, st *migrationStack, siteID string, orgID int64) (projectID, envID string) {
	t.Helper()
	if _, err := st.database.Exec(
		`INSERT OR IGNORE INTO sites (id, name, git_repo_id, org_id) VALUES (?, ?, '', ?)`,
		siteID, siteID, orgID); err != nil {
		t.Fatalf("insert site %s: %v", siteID, err)
	}
	projectID, err := st.p.EnsureSiteProject(context.Background(), siteID, siteID, orgID)
	if err != nil {
		t.Fatalf("attach site %s: %v", siteID, err)
	}
	var eid string
	if err := st.database.QueryRow(
		`SELECT id FROM project_environments WHERE project_id = ? AND slug = 'production'`, projectID).
		Scan(&eid); err != nil {
		t.Fatalf("find production environment: %v", err)
	}
	return projectID, eid
}

func insertLegacyEnvVar(t *testing.T, st *migrationStack, siteID, key, value string, build, runtime bool) {
	t.Helper()
	bt, rt := 0, 0
	if build {
		bt = 1
	}
	if runtime {
		rt = 1
	}
	if _, err := st.database.Exec(
		`INSERT OR REPLACE INTO site_env_vars (id, site_id, key, value_encrypted, is_build_time, is_runtime)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("%s-%s", siteID, key), siteID, key, value, bt, rt); err != nil {
		t.Fatalf("insert legacy env var %s: %v", key, err)
	}
}

func countSecrets(t *testing.T, st *migrationStack) int {
	t.Helper()
	var n int
	if err := st.database.QueryRow(`SELECT COUNT(*) FROM secrets`).Scan(&n); err != nil {
		t.Fatalf("count secrets: %v", err)
	}
	return n
}

func checkpointState(t *testing.T, st *migrationStack, siteID string) string {
	t.Helper()
	var state string
	err := st.database.QueryRow(
		`SELECT state FROM legacy_migration_checkpoints WHERE site_id = ?`, siteID).Scan(&state)
	if err != nil {
		return ""
	}
	return state
}

func readNativeSecret(t *testing.T, st *migrationStack, orgID int64, projectID, envID, key string) (string, error) {
	t.Helper()
	ctx := auth.WithOrgID(context.Background(), orgID)
	folders, err := st.s.ListFolders(ctx, projectID, envID)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	for _, f := range folders {
		if f.Name != "default" {
			continue
		}
		val, err := st.s.ReadSecretValue(ctx, projectID, envID, f.ID, key)
		return val.Value, err
	}
	return "", errors.New("default folder missing")
}

func TestLegacyMigrationRequiresExplicitMode(t *testing.T) {
	// SC-e89s06-P0-05: without an explicit plaintext/ciphertext mode the
	// migration stops with an actionable error and changes no rows.
	st := legacyMigrationFixture(t)
	seedAttachedSite(t, st, "site-a", 1)
	insertLegacyEnvVar(t, st, "site-a", "DB_URL", "plaintext-value", false, true)

	_, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:      st.database,
		Secrets: st.s,
	})
	if !errors.Is(err, backup.ErrLegacyMigrationModeRequired) {
		t.Fatalf("expected ErrLegacyMigrationModeRequired, got %v", err)
	}
	if !strings.Contains(err.Error(), "plaintext") || !strings.Contains(err.Error(), "ciphertext") {
		t.Fatalf("error must be actionable (name the modes), got %v", err)
	}
	if n := countSecrets(t, st); n != 0 {
		t.Fatalf("migration without mode changed %d secret rows", n)
	}
}

func TestLegacyMigrationPlaintextMode(t *testing.T) {
	// SC-e89s06-P0-04: legacy values remain deployable during dual-read; the
	// explicit plaintext mode moves runtime rows into native secrets and
	// leaves build-only rows in the legacy store.
	st := legacyMigrationFixture(t)
	projectID, envID := seedAttachedSite(t, st, "site-a", 1)
	insertLegacyEnvVar(t, st, "site-a", "DB_URL", "postgres://legacy", false, true)
	insertLegacyEnvVar(t, st, "site-a", "BUILD_FLAG", "build-only", true, false)

	report, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:      st.database,
		Secrets: st.s,
		Mode:    backup.LegacyModePlaintext,
	})
	if err != nil {
		t.Fatalf("MigrateLegacyValues: %v", err)
	}
	if report.Migrated != 1 || report.BuildOnlySkipped != 1 {
		t.Fatalf("expected 1 migrated and 1 build-only skipped, got %+v", report)
	}
	if len(report.Sites) != 1 || report.Sites[0] != "site-a" {
		t.Fatalf("expected site-a in report, got %+v", report)
	}

	val, err := readNativeSecret(t, st, 1, projectID, envID, "DB_URL")
	if err != nil {
		t.Fatalf("migrated secret unreadable: %v", err)
	}
	if val != "postgres://legacy" {
		t.Fatalf("migrated value mismatch: %q", val)
	}

	// Legacy rows are never deleted — dual-read keeps them deployable.
	var legacyCount int
	if err := st.database.QueryRow(`SELECT COUNT(*) FROM site_env_vars WHERE site_id = 'site-a'`).Scan(&legacyCount); err != nil {
		t.Fatalf("count legacy rows: %v", err)
	}
	if legacyCount != 2 {
		t.Fatalf("legacy rows must not be deleted during migration, got %d", legacyCount)
	}
	if got := checkpointState(t, st, "site-a"); got != "done" {
		t.Fatalf("expected done checkpoint, got %q", got)
	}
}

func TestLegacyMigrationCiphertextMode(t *testing.T) {
	// The explicit ciphertext mode decrypts legacy envelopes under the legacy
	// hex key and stores the plaintext as a native secret.
	st := legacyMigrationFixture(t)
	projectID, envID := seedAttachedSite(t, st, "site-a", 1)

	encKey := strings.Repeat("ab", 32) // 32-byte hex key
	keyBytes, err := hex.DecodeString(encKey)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	encrypted, err := envcrypto.Encrypt(keyBytes, "cipher-secret-value")
	if err != nil {
		t.Fatalf("encrypt legacy value: %v", err)
	}
	insertLegacyEnvVar(t, st, "site-a", "API_TOKEN", encrypted, false, true)

	report, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:               st.database,
		Secrets:          st.s,
		Mode:             backup.LegacyModeCiphertext,
		EnvEncryptionKey: encKey,
	})
	if err != nil {
		t.Fatalf("MigrateLegacyValues: %v", err)
	}
	if report.Migrated != 1 {
		t.Fatalf("expected 1 migrated, got %+v", report)
	}
	val, err := readNativeSecret(t, st, 1, projectID, envID, "API_TOKEN")
	if err != nil {
		t.Fatalf("migrated secret unreadable: %v", err)
	}
	if val != "cipher-secret-value" {
		t.Fatalf("migrated value mismatch: %q", val)
	}
}

func TestLegacyMigrationUnknownFormatStops(t *testing.T) {
	// SC-e89s06-P0-05: a row that cannot be decrypted under the declared mode
	// stops the site's migration with an actionable, value-free error and
	// changes no rows for that site.
	st := legacyMigrationFixture(t)
	seedAttachedSite(t, st, "site-a", 1)
	insertLegacyEnvVar(t, st, "site-a", "GOOD", "garbage-not-ciphertext", false, true)
	insertLegacyEnvVar(t, st, "site-a", "BAD", "also-not-ciphertext", false, true)

	_, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:               st.database,
		Secrets:          st.s,
		Mode:             backup.LegacyModeCiphertext,
		EnvEncryptionKey: strings.Repeat("ab", 32),
	})
	if !errors.Is(err, backup.ErrLegacyMigrationUnknownFormat) {
		t.Fatalf("expected ErrLegacyMigrationUnknownFormat, got %v", err)
	}
	msg := err.Error()
	// Rows are validated in ORDER BY key (binary collation), so the first
	// failing key is BAD. The contract requires an actionable site+key error.
	if !strings.Contains(msg, "site-a") || !strings.Contains(msg, "BAD") {
		t.Fatalf("error must be actionable (site + key), got %v", err)
	}
	if strings.Contains(msg, "garbage") || strings.Contains(msg, "also-not-ciphertext") {
		t.Fatalf("error leaked a value: %v", err)
	}
	if n := countSecrets(t, st); n != 0 {
		t.Fatalf("unknown format changed %d secret rows", n)
	}
	if got := checkpointState(t, st, "site-a"); got == "done" {
		t.Fatal("site must not be checkpointed done after an unknown-format failure")
	}
}

func TestLegacyMigrationResumable(t *testing.T) {
	// An interrupted run resumes from the next incomplete site: a site whose
	// checkpoint is done is skipped, not rewritten.
	st := legacyMigrationFixture(t)
	seedAttachedSite(t, st, "site-a", 1)
	seedAttachedSite(t, st, "site-b", 1)
	insertLegacyEnvVar(t, st, "site-a", "A_VAR", "a-value", false, true)
	insertLegacyEnvVar(t, st, "site-b", "B_VAR", "b-value", false, true)

	// First run: only site-a.
	first, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:      st.database,
		Secrets: st.s,
		Mode:    backup.LegacyModePlaintext,
		SiteID:  "site-a",
	})
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if first.Migrated != 1 {
		t.Fatalf("expected 1 migrated in first run, got %+v", first)
	}

	// Second run over everything: site-a is resumed (skipped), site-b migrates.
	second, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:      st.database,
		Secrets: st.s,
		Mode:    backup.LegacyModePlaintext,
	})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.ResumedSites != 1 {
		t.Fatalf("expected site-a resumed, got %+v", second)
	}
	if second.Migrated != 1 || len(second.Sites) != 1 || second.Sites[0] != "site-b" {
		t.Fatalf("expected only site-b migrated, got %+v", second)
	}
	if n := countSecrets(t, st); n != 2 {
		t.Fatalf("expected 2 native secrets after resume, got %d", n)
	}
}

func TestLegacyMigrationUnattachedSite(t *testing.T) {
	// A site with legacy rows but no project attachment is reported and left
	// for a later run; its checkpoint is not marked done.
	st := legacyMigrationFixture(t)
	if _, err := st.database.Exec(
		`INSERT INTO sites (id, name, git_repo_id, org_id) VALUES ('site-orphan', 'orphan', '', 1)`); err != nil {
		t.Fatalf("insert orphan site: %v", err)
	}
	insertLegacyEnvVar(t, st, "site-orphan", "DB_URL", "legacy-value", false, true)

	report, err := backup.MigrateLegacyValues(context.Background(), backup.LegacyMigrationOptions{
		DB:      st.database,
		Secrets: st.s,
		Mode:    backup.LegacyModePlaintext,
	})
	if err != nil {
		t.Fatalf("MigrateLegacyValues: %v", err)
	}
	if len(report.UnattachedSites) != 1 || report.UnattachedSites[0] != "site-orphan" {
		t.Fatalf("expected unattached site reported, got %+v", report)
	}
	if report.Migrated != 0 {
		t.Fatalf("expected no migrations, got %+v", report)
	}
	if got := checkpointState(t, st, "site-orphan"); got == "done" {
		t.Fatal("unattached site must not be checkpointed done")
	}
}
