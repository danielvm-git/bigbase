package secrets_test

import (
	"context"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

func TestSchemaCreatesSecretTables(t *testing.T) {
	d, _, _ := secretsFixture(t)
	for _, table := range []string{"secret_folders", "project_key_records", "secrets", "secret_versions"} {
		var name string
		if err := d.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
		if name != table {
			t.Fatalf("unexpected table name %q", name)
		}
	}
	for _, idx := range []string{
		"idx_project_key_records_active",
		"idx_project_key_records_rotating",
		"idx_secrets_folder",
		"idx_secret_versions_secret",
	} {
		var name string
		if err := d.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?`, idx).Scan(&name); err != nil {
			t.Fatalf("index %s missing: %v", idx, err)
		}
	}
}

func TestSchemaMigrationIsIdempotent(t *testing.T) {
	d, s, _ := secretsFixture(t)
	// Re-running Start must be a no-op (CREATE IF NOT EXISTS).
	if err := s.Start(&kernel.Context{Logger: testLogger{}}); err != nil {
		t.Fatalf("restart secrets: %v", err)
	}
	if err := s.Start(&kernel.Context{Logger: testLogger{}}); err != nil {
		t.Fatalf("restart secrets again: %v", err)
	}
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n); err != nil {
		t.Fatalf("projects table missing after restart: %v", err)
	}
}

func TestSchemaIsCiphertextOnly(t *testing.T) {
	d, _, _ := secretsFixture(t)
	// secret_versions must hold exactly the envelope columns; no plaintext
	// storage is possible because no such column exists.
	var columns []string
	rows, err := d.Query(`PRAGMA table_info(secret_versions)`)
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt *string
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		columns = append(columns, name)
	}
	_ = rows.Close()
	want := map[string]bool{
		"id": true, "secret_id": true, "version_no": true, "key_record_id": true,
		"key_id": true, "algorithm": true, "nonce": true, "ciphertext": true, "created_at": true,
	}
	if len(columns) != len(want) {
		t.Fatalf("unexpected secret_versions columns: %v", columns)
	}
	for _, c := range columns {
		if !want[c] {
			t.Fatalf("unexpected secret_versions column %q", c)
		}
		if strings.Contains(strings.ToLower(c), "value") || strings.Contains(strings.ToLower(c), "plain") {
			t.Fatalf("plaintext-capable column %q found in secret_versions", c)
		}
	}
}

func TestSchemaForeignKeyConstraints(t *testing.T) {
	d, s, p, _ := fileFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	_ = ctx
	_ = folderID

	assertFails := func(label, query string, args ...any) {
		t.Helper()
		if _, err := d.Exec(query, args...); err == nil {
			t.Fatalf("%s: expected foreign key violation", label)
		}
	}

	// A folder must belong to an existing project/environment.
	assertFails("folder bad project", `INSERT INTO secret_folders (id, project_id, environment_id, name) VALUES ('f-bad', 'missing-project', ?, 'x')`, envID)
	assertFails("folder bad environment", `INSERT INTO secret_folders (id, project_id, environment_id, name) VALUES ('f-bad2', ?, 'missing-env', 'x')`, projectID)
	// A secret must belong to an existing folder.
	assertFails("secret bad folder", `INSERT INTO secrets (id, project_id, environment_id, folder_id, key, current_version) VALUES ('s-bad', ?, ?, 'missing-folder', 'K', 1)`, projectID, envID)
	// A version must belong to an existing secret and key record.
	assertFails("version bad secret", `INSERT INTO secret_versions (id, secret_id, version_no, key_record_id, key_id, algorithm, nonce, ciphertext) VALUES ('v-bad', 'missing-secret', 1, 'missing-key-record', 'k', 'aes-256-gcm', 'n', 'c')`)
	// A key record must belong to an existing project.
	assertFails("key record bad project", `INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key) VALUES ('kr-bad', 'missing-project', 'k', 'aes-256-gcm', 'enc')`)
}

func TestSchemaUniqueConstraints(t *testing.T) {
	d, s, p, _ := fileFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)

	// Folder names are unique per project environment.
	if _, err := s.EnsureFolder(ctx, projectID, envID, "payments"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if _, err := s.EnsureFolder(ctx, projectID, envID, "payments"); err != nil {
		t.Fatalf("EnsureFolder must be idempotent: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO secret_folders (id, project_id, environment_id, name) VALUES ('f-dup', ?, ?, 'payments')`, projectID, envID); err == nil {
		t.Fatal("duplicate folder name accepted")
	}

	// Secret keys are unique per folder.
	if _, err := s.CreateSecret(ctx, projectID, envID, folderID, "TOKEN", "one"); err != nil {
		t.Fatalf("create secret: %v", err)
	}
	if _, err := s.CreateSecret(ctx, projectID, envID, folderID, "TOKEN", "two"); err == nil {
		t.Fatal("duplicate secret key accepted")
	}

	// Version numbers are unique per secret.
	var secretID string
	if err := d.QueryRow(`SELECT id FROM secrets WHERE key = 'TOKEN'`).Scan(&secretID); err != nil {
		t.Fatalf("lookup secret: %v", err)
	}
	var keyRecordID string
	if err := d.QueryRow(`SELECT id FROM project_key_records WHERE project_id = ? AND state = 'active'`, projectID).Scan(&keyRecordID); err != nil {
		t.Fatalf("lookup key record: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO secret_versions (id, secret_id, version_no, key_record_id, key_id, algorithm, nonce, ciphertext) VALUES ('v-dup', ?, 1, ?, 'k', 'aes-256-gcm', 'n', 'c')`, secretID, keyRecordID); err == nil {
		t.Fatal("duplicate version number accepted")
	}

	// Exactly one active and one rotating key may exist per project.
	if _, err := d.Exec(`INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key) VALUES ('kr-2', ?, 'key-2', 'aes-256-gcm', 'enc')`, projectID); err == nil {
		t.Fatal("second active key accepted")
	}
	if _, err := d.Exec(`INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key, state) VALUES ('kr-3', ?, 'key-3', 'aes-256-gcm', 'enc', 'rotating')`, projectID); err != nil {
		t.Fatalf("first rotating key rejected: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key, state) VALUES ('kr-4', ?, 'key-4', 'aes-256-gcm', 'enc', 'rotating')`, projectID); err == nil {
		t.Fatal("second rotating key accepted")
	}
	// Retired keys may coexist with an active key (rotation history).
	if _, err := d.Exec(`INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key, state, retired_at) VALUES ('kr-5', ?, 'key-5', 'aes-256-gcm', 'enc', 'retired', '2026-01-01T00:00:00Z')`, projectID); err != nil {
		t.Fatalf("retired key rejected: %v", err)
	}
}

func TestSchemaKeyStateConstraint(t *testing.T) {
	d, _, p, _ := fileFixture(t)
	ctx := auth.WithOrgID(context.Background(), 1)
	proj, err := p.CreateProject(ctx, "StateCheck")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key, state) VALUES ('kr-bad-state', ?, 'k', 'aes-256-gcm', 'enc', 'invalid')`, proj.ID); err == nil {
		t.Fatal("invalid key state accepted")
	}
}
