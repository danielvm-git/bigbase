package secrets_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/backup"
	"github.com/danielvm/bigbase/components/secrets"
	_ "modernc.org/sqlite"
)

func mustCreateSecret(t *testing.T, ctx context.Context, s *secrets.Secrets, projectID, envID, folderID, key, value string) string {
	t.Helper()
	meta, err := s.CreateSecret(ctx, projectID, envID, folderID, key, value)
	if err != nil {
		t.Fatalf("CreateSecret(%s): %v", key, err)
	}
	return meta.ID
}

func TestSecretManagerCreateReadRoundTrip(t *testing.T) {
	d, s, p := secretsFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	plaintext := "postgres://user:pass@host/db"

	meta, err := s.CreateSecret(ctx, projectID, envID, folderID, "DATABASE_URL", plaintext)
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if meta.Key != "DATABASE_URL" || meta.CurrentVersion != 1 {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if meta.ValuePreview == "" || meta.ValuePreview == plaintext || strings.Contains(meta.ValuePreview, "postgres://user") {
		t.Fatalf("preview must be masked, got %q", meta.ValuePreview)
	}

	// Explicit value read returns the plaintext.
	val, err := s.ReadSecretValue(ctx, projectID, envID, folderID, "DATABASE_URL")
	if err != nil {
		t.Fatalf("ReadSecretValue: %v", err)
	}
	if val.Value != plaintext || val.Version != 1 || val.Algorithm != "aes-256-gcm" || val.KeyID == "" {
		t.Fatalf("unexpected value result: %+v", val)
	}

	// SC-e89s03-P0-01: the database contains ciphertext and no plaintext.
	var nonce, ciphertext string
	if err := d.QueryRow(`SELECT nonce, ciphertext FROM secret_versions`).Scan(&nonce, &ciphertext); err != nil {
		t.Fatalf("read version row: %v", err)
	}
	if nonce == "" || ciphertext == "" {
		t.Fatal("empty envelope in storage")
	}
	if strings.Contains(ciphertext, plaintext) || strings.Contains(nonce, plaintext) {
		t.Fatal("plaintext leaked into the stored envelope")
	}
	var total int
	if err := d.QueryRow(`SELECT COUNT(*) FROM secret_versions`).Scan(&total); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 version, got %d", total)
	}
}

func TestSecretManagerMetadataNeverLeaksValue(t *testing.T) {
	_, s, p := secretsFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	plaintext := "super-secret-plaintext-42"
	if _, err := s.CreateSecret(ctx, projectID, envID, folderID, "TOKEN", plaintext); err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}

	items, err := s.ListSecrets(ctx, projectID, envID, folderID)
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if len(items) != 1 || items[0].Key != "TOKEN" {
		t.Fatalf("unexpected list: %+v", items)
	}
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal list: %v", err)
	}
	if strings.Contains(string(raw), plaintext) || strings.Contains(string(raw), "ciphertext") {
		t.Fatalf("list response leaked secret material: %s", raw)
	}

	meta, err := s.GetSecretMetadata(ctx, projectID, envID, folderID, "TOKEN")
	if err != nil {
		t.Fatalf("GetSecretMetadata: %v", err)
	}
	rawMeta, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if strings.Contains(string(rawMeta), plaintext) {
		t.Fatalf("metadata response leaked plaintext: %s", rawMeta)
	}

	versions, err := s.ListVersions(ctx, projectID, envID, folderID, "TOKEN")
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 1 || versions[0].Version != 1 {
		t.Fatalf("unexpected versions: %+v", versions)
	}
	rawVersions, err := json.Marshal(versions)
	if err != nil {
		t.Fatalf("marshal versions: %v", err)
	}
	if strings.Contains(string(rawVersions), plaintext) || strings.Contains(string(rawVersions), "ciphertext") {
		t.Fatalf("version list leaked secret material: %s", rawVersions)
	}
}

func TestSecretManagerUpdateCreatesImmutableVersions(t *testing.T) {
	d, s, p := secretsFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	if _, err := s.CreateSecret(ctx, projectID, envID, folderID, "TOKEN", "v1"); err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}

	var v1Nonce, v1Ciphertext, v1KeyRecord string
	if err := d.QueryRow(`SELECT nonce, ciphertext, key_record_id FROM secret_versions WHERE version_no = 1`).
		Scan(&v1Nonce, &v1Ciphertext, &v1KeyRecord); err != nil {
		t.Fatalf("capture version 1 row: %v", err)
	}

	for _, v := range []string{"v2", "v3"} {
		if _, err := s.UpdateSecret(ctx, projectID, envID, folderID, "TOKEN", v); err != nil {
			t.Fatalf("UpdateSecret(%s): %v", v, err)
		}
	}

	meta, err := s.GetSecretMetadata(ctx, projectID, envID, folderID, "TOKEN")
	if err != nil {
		t.Fatalf("GetSecretMetadata: %v", err)
	}
	if meta.CurrentVersion != 3 {
		t.Fatalf("expected current_version 3, got %d", meta.CurrentVersion)
	}

	versions, err := s.ListVersions(ctx, projectID, envID, folderID, "TOKEN")
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 3 || versions[0].Version != 1 || versions[2].Version != 3 {
		t.Fatalf("unexpected version list: %+v", versions)
	}

	// SC-e89s03-P0-02: old versions remain readable and immutable.
	expect := map[int]string{1: "v1", 2: "v2", 3: "v3"}
	for version, want := range expect {
		val, err := s.ReadSecretVersionValue(ctx, projectID, envID, folderID, "TOKEN", version)
		if err != nil {
			t.Fatalf("ReadSecretVersionValue(%d): %v", version, err)
		}
		if val.Value != want {
			t.Fatalf("version %d = %q, want %q", version, val.Value, want)
		}
	}
	cur, err := s.ReadSecretValue(ctx, projectID, envID, folderID, "TOKEN")
	if err != nil {
		t.Fatalf("ReadSecretValue: %v", err)
	}
	if cur.Value != "v3" || cur.Version != 3 {
		t.Fatalf("current value = %q version %d", cur.Value, cur.Version)
	}

	// Version 1's stored envelope is byte-identical to what was created.
	var nonce, ciphertext, keyRecord string
	if err := d.QueryRow(`SELECT nonce, ciphertext, key_record_id FROM secret_versions WHERE version_no = 1`).
		Scan(&nonce, &ciphertext, &keyRecord); err != nil {
		t.Fatalf("re-read version 1 row: %v", err)
	}
	if nonce != v1Nonce || ciphertext != v1Ciphertext || keyRecord != v1KeyRecord {
		t.Fatal("version 1 row was mutated by an update")
	}
}

func TestSecretManagerWrongScopeCopyFails(t *testing.T) {
	d, s, p := secretsFixture(t)
	ctx, projectID, envID, folderA := createScope(t, p, s, 1)
	folderB, err := s.EnsureFolder(ctx, projectID, envID, "other")
	if err != nil {
		t.Fatalf("EnsureFolder: %v", err)
	}
	mustCreateSecret(t, ctx, s, projectID, envID, folderA, "ALPHA", "alpha-value")
	mustCreateSecret(t, ctx, s, projectID, envID, folderB.ID, "BETA", "beta-value")

	// SC-e89s03-P0-03: copy ALPHA's ciphertext into BETA's version row; the
	// scope-bound AAD must reject it.
	if _, err := d.Exec(`
		UPDATE secret_versions
		SET key_record_id = (SELECT key_record_id FROM secret_versions sv JOIN secrets sec ON sec.id = sv.secret_id WHERE sec.key = 'ALPHA'),
		    key_id = (SELECT key_id FROM secret_versions sv JOIN secrets sec ON sec.id = sv.secret_id WHERE sec.key = 'ALPHA'),
		    algorithm = (SELECT algorithm FROM secret_versions sv JOIN secrets sec ON sec.id = sv.secret_id WHERE sec.key = 'ALPHA'),
		    nonce = (SELECT nonce FROM secret_versions sv JOIN secrets sec ON sec.id = sv.secret_id WHERE sec.key = 'ALPHA'),
		    ciphertext = (SELECT ciphertext FROM secret_versions sv JOIN secrets sec ON sec.id = sv.secret_id WHERE sec.key = 'ALPHA')
		WHERE secret_id = (SELECT id FROM secrets WHERE key = 'BETA')`); err != nil {
		t.Fatalf("copy ciphertext: %v", err)
	}
	if _, err := s.ReadSecretValue(ctx, projectID, envID, folderB.ID, "BETA"); !errors.Is(err, secrets.ErrDecryptionFailed) {
		t.Fatalf("copied ciphertext decrypted: err=%v", err)
	}
	// The source secret is unaffected.
	val, err := s.ReadSecretValue(ctx, projectID, envID, folderA, "ALPHA")
	if err != nil || val.Value != "alpha-value" {
		t.Fatalf("source secret broken: val=%+v err=%v", val, err)
	}
}

func TestSecretManagerConcurrentFirstWrites(t *testing.T) {
	d, s, p, _ := fileFixture(t)
	// The race test needs genuinely separate connections: widen the pool the
	// component uses beyond the single default SQLite connection.
	d.DB().SetMaxOpenConns(8)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)

	const workers = 4
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("CONC_%d", n)
			value := fmt.Sprintf("value-%d", n)
			if _, err := s.CreateSecret(ctx, projectID, envID, folderID, key, value); err != nil {
				errCh <- fmt.Errorf("CreateSecret(%s): %w", key, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent create: %v", err)
	}

	// SC-e89s03-P0-04: exactly one Project Data Key exists.
	var keyCount int
	if err := d.QueryRow(`SELECT COUNT(*) FROM project_key_records WHERE project_id = ? AND state = 'active'`, projectID).Scan(&keyCount); err != nil {
		t.Fatalf("count active keys: %v", err)
	}
	if keyCount != 1 {
		t.Fatalf("expected exactly 1 active project data key, got %d", keyCount)
	}
	var totalKeys int
	if err := d.QueryRow(`SELECT COUNT(*) FROM project_key_records WHERE project_id = ?`, projectID).Scan(&totalKeys); err != nil {
		t.Fatalf("count keys: %v", err)
	}
	if totalKeys != 1 {
		t.Fatalf("expected 1 key record total, got %d", totalKeys)
	}

	// Both (all) writes decrypt correctly.
	for i := 0; i < workers; i++ {
		key := fmt.Sprintf("CONC_%d", i)
		val, err := s.ReadSecretValue(ctx, projectID, envID, folderID, key)
		if err != nil {
			t.Fatalf("read %s: %v", key, err)
		}
		if want := fmt.Sprintf("value-%d", i); val.Value != want {
			t.Fatalf("%s = %q, want %q", key, val.Value, want)
		}
	}
}

func TestSecretManagerInvalidKeyMaterialFailsClosed(t *testing.T) {
	d, s, p := secretsFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	const plaintext = "top-secret-value-xyz"

	// wrong root key: a manager bound to different key material cannot read.
	other, err := secrets.New(secrets.Options{DB: d, Logger: testLogger{}, RootKey: bytes.Repeat([]byte{0x77}, 32)})
	if err != nil {
		t.Fatalf("new other manager: %v", err)
	}
	mustCreateSecret(t, ctx, s, projectID, envID, folderID, "TOKEN", plaintext)
	_, err = other.ReadSecretValue(ctx, projectID, envID, folderID, "TOKEN")
	if !errors.Is(err, secrets.ErrDecryptionFailed) {
		t.Fatalf("wrong root key read: err=%v", err)
	}
	if strings.Contains(err.Error(), plaintext) {
		t.Fatal("decryption error leaked plaintext")
	}

	// Field tampering must fail closed without exposing plaintext.
	cases := []struct {
		name   string
		mutate func(t *testing.T, secretID string)
		want   error
	}{
		{
			name: "wrong key id",
			mutate: func(t *testing.T, secretID string) {
				if _, err := d.Exec(`UPDATE secret_versions SET key_id = 'ffffffffffffffffffffffffffffffff' WHERE secret_id = ?`, secretID); err != nil {
					t.Fatalf("tamper key id: %v", err)
				}
			},
			want: secrets.ErrInvalidKeyMaterial,
		},
		{
			name: "wrong algorithm",
			mutate: func(t *testing.T, secretID string) {
				if _, err := d.Exec(`UPDATE secret_versions SET algorithm = 'aes-128-gcm' WHERE secret_id = ?`, secretID); err != nil {
					t.Fatalf("tamper algorithm: %v", err)
				}
			},
			want: secrets.ErrInvalidKeyMaterial,
		},
		{
			name: "tampered nonce",
			mutate: func(t *testing.T, secretID string) {
				nonce := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x00}, 12))
				if _, err := d.Exec(`UPDATE secret_versions SET nonce = ? WHERE secret_id = ?`, nonce, secretID); err != nil {
					t.Fatalf("tamper nonce: %v", err)
				}
			},
			want: secrets.ErrDecryptionFailed,
		},
		{
			name: "tampered ciphertext",
			mutate: func(t *testing.T, secretID string) {
				var ct string
				if err := d.QueryRow(`SELECT ciphertext FROM secret_versions WHERE secret_id = ?`, secretID).Scan(&ct); err != nil {
					t.Fatalf("read ciphertext: %v", err)
				}
				raw, err := base64.StdEncoding.DecodeString(ct)
				if err != nil {
					t.Fatalf("decode ciphertext: %v", err)
				}
				raw[len(raw)-1] ^= 0xff
				if _, err := d.Exec(`UPDATE secret_versions SET ciphertext = ? WHERE secret_id = ?`, base64.StdEncoding.EncodeToString(raw), secretID); err != nil {
					t.Fatalf("tamper ciphertext: %v", err)
				}
			},
			want: secrets.ErrDecryptionFailed,
		},
	}
	for i, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			key := fmt.Sprintf("TAMPER_%d", i)
			secretID := mustCreateSecret(t, ctx, s, projectID, envID, folderID, key, plaintext)
			tt.mutate(t, secretID)
			_, err := s.ReadSecretValue(ctx, projectID, envID, folderID, key)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), plaintext) {
				t.Fatal("decryption error leaked plaintext")
			}
		})
	}
}

func TestSecretManagerBackupRestoreAndRotation(t *testing.T) {
	d, s, p, _ := fileFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	if _, err := s.CreateSecret(ctx, projectID, envID, folderID, "API_KEY", "v1-secret"); err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if _, err := s.UpdateSecret(ctx, projectID, envID, folderID, "API_KEY", "v2-secret"); err != nil {
		t.Fatalf("UpdateSecret: %v", err)
	}

	// SC-e89s03-P1-06: begin rotation as the checkpoint, then back up.
	rot, err := s.BeginKeyRotation(ctx, projectID)
	if err != nil {
		t.Fatalf("BeginKeyRotation: %v", err)
	}
	if rot.State != "rotating" {
		t.Fatalf("unexpected rotation record state %q", rot.State)
	}
	if err := assertKeyState(t, d, projectID, "active", 1); err != nil {
		t.Fatalf("old key not active during rotation: %v", err)
	}
	if err := assertKeyState(t, d, projectID, "rotating", 1); err != nil {
		t.Fatalf("rotating checkpoint missing: %v", err)
	}
	// Old key is NOT retired early and versions still decrypt.
	val, err := s.ReadSecretValue(ctx, projectID, envID, folderID, "API_KEY")
	if err != nil || val.Value != "v2-secret" {
		t.Fatalf("read during rotation: val=%+v err=%v", val, err)
	}

	var dump bytes.Buffer
	if err := backup.Dump(ctx, d, &dump); err != nil {
		t.Fatalf("backup dump: %v", err)
	}

	// Restore into a fresh database file. The connection is pinned so the
	// dump's PRAGMA foreign_keys=OFF applies to every replayed statement;
	// pooled connections would each re-apply the DSN pragma and re-enable
	// foreign keys mid-restore.
	restorePath := t.TempDir() + "/restored.db"
	rawDB, err := sql.Open("sqlite", "file:"+restorePath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open restore db: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	if err := backup.Restore(ctx, rawDB, dump.String()); err != nil {
		_ = rawDB.Close()
		t.Fatalf("backup restore: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close restore db: %v", err)
	}

	// Start the component stack over the restored database; migrations are
	// idempotent and recreate the partial unique indexes the dump omits.
	d2, s2, p2 := startKernel(t, "file:"+restorePath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)")
	_ = d2
	ctx2 := auth.WithOrgID(context.Background(), 1)
	if _, err := p2.GetProject(ctx2, projectID); err != nil {
		t.Fatalf("restored project missing: %v", err)
	}

	// Resume from the checkpoint: versions decrypt, old key still active.
	val, err = s2.ReadSecretValue(ctx2, projectID, envID, folderID, "API_KEY")
	if err != nil || val.Value != "v2-secret" {
		t.Fatalf("read after restore: val=%+v err=%v", val, err)
	}
	v1, err := s2.ReadSecretVersionValue(ctx2, projectID, envID, folderID, "API_KEY", 1)
	if err != nil || v1.Value != "v1-secret" {
		t.Fatalf("version 1 after restore: val=%+v err=%v", v1, err)
	}
	if err := assertKeyState(t, d2, projectID, "active", 1); err != nil {
		t.Fatalf("restored active key broken: %v", err)
	}
	if err := assertKeyState(t, d2, projectID, "rotating", 1); err != nil {
		t.Fatalf("restored rotating checkpoint broken: %v", err)
	}

	// Resume rotation and complete it; the old key retires only at the end.
	rotated, err := s2.RotateCurrentVersions(ctx2, projectID)
	if err != nil {
		t.Fatalf("RotateCurrentVersions: %v", err)
	}
	if rotated != 1 {
		t.Fatalf("expected 1 current version rotated, got %d", rotated)
	}
	if err := assertKeyState(t, d2, projectID, "active", 1); err != nil {
		t.Fatalf("old key retired before completion: %v", err)
	}
	if err := s2.CompleteKeyRotation(ctx2, projectID); err != nil {
		t.Fatalf("CompleteKeyRotation: %v", err)
	}
	if err := assertKeyState(t, d2, projectID, "active", 1); err != nil {
		t.Fatalf("rotating key not promoted: %v", err)
	}
	if err := assertKeyState(t, d2, projectID, "retired", 1); err != nil {
		t.Fatalf("old key not retired: %v", err)
	}
	if err := assertKeyState(t, d2, projectID, "rotating", 0); err != nil {
		t.Fatalf("rotating checkpoint not cleared: %v", err)
	}

	// Both versions remain decryptable after rotation completes.
	val, err = s2.ReadSecretValue(ctx2, projectID, envID, folderID, "API_KEY")
	if err != nil || val.Value != "v2-secret" {
		t.Fatalf("read after rotation: val=%+v err=%v", val, err)
	}
	v1, err = s2.ReadSecretVersionValue(ctx2, projectID, envID, folderID, "API_KEY", 1)
	if err != nil || v1.Value != "v1-secret" {
		t.Fatalf("version 1 after rotation: val=%+v err=%v", v1, err)
	}
}

func assertKeyState(t *testing.T, d interface {
	QueryRow(query string, args ...any) *sql.Row
}, projectID, state string, want int) error {
	t.Helper()
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM project_key_records WHERE project_id = ? AND state = ?`, projectID, state).Scan(&n); err != nil {
		return err
	}
	if n != want {
		return fmt.Errorf("state %q count = %d, want %d", state, n, want)
	}
	return nil
}

func TestSecretManagerDeleteSecret(t *testing.T) {
	d, s, p, _ := fileFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	mustCreateSecret(t, ctx, s, projectID, envID, folderID, "KEEP", "keep")
	delID := mustCreateSecret(t, ctx, s, projectID, envID, folderID, "DROP", "drop")

	if err := s.DeleteSecret(ctx, projectID, envID, folderID, "DROP"); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if err := s.DeleteSecret(ctx, projectID, envID, folderID, "DROP"); !errors.Is(err, secrets.ErrSecretNotFound) {
		t.Fatalf("second delete: err=%v", err)
	}
	items, err := s.ListSecrets(ctx, projectID, envID, folderID)
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if len(items) != 1 || items[0].Key != "KEEP" {
		t.Fatalf("unexpected list after delete: %+v", items)
	}
	// Cascade removed the versions too (foreign keys are enabled in fileFixture).
	var versions int
	if err := d.QueryRow(`SELECT COUNT(*) FROM secret_versions WHERE secret_id = ?`, delID).Scan(&versions); err != nil {
		t.Fatalf("count deleted versions: %v", err)
	}
	if versions != 0 {
		t.Fatalf("versions survived secret deletion: %d", versions)
	}
}

func TestSecretManagerCrossOrgIsolation(t *testing.T) {
	_, s, p := secretsFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)
	mustCreateSecret(t, ctx, s, projectID, envID, folderID, "TOKEN", "value")

	other := auth.WithOrgID(context.Background(), 2)
	if _, err := s.ReadSecretValue(other, projectID, envID, folderID, "TOKEN"); !errors.Is(err, secrets.ErrProjectNotFound) {
		t.Fatalf("cross-org read: err=%v", err)
	}
	if _, err := s.ListSecrets(other, projectID, envID, folderID); !errors.Is(err, secrets.ErrProjectNotFound) {
		t.Fatalf("cross-org list: err=%v", err)
	}
	if _, err := s.EnsureFolder(other, projectID, envID, "x"); !errors.Is(err, secrets.ErrProjectNotFound) {
		t.Fatalf("cross-org folder: err=%v", err)
	}
	if _, err := s.GetSecretMetadata(other, projectID, envID, folderID, "TOKEN"); !errors.Is(err, secrets.ErrProjectNotFound) {
		t.Fatalf("cross-org metadata: err=%v", err)
	}
	if err := s.DeleteSecret(other, projectID, envID, folderID, "TOKEN"); !errors.Is(err, secrets.ErrProjectNotFound) {
		t.Fatalf("cross-org delete: err=%v", err)
	}
	// The secret is untouched.
	val, err := s.ReadSecretValue(ctx, projectID, envID, folderID, "TOKEN")
	if err != nil || val.Value != "value" {
		t.Fatalf("secret changed by cross-org attempt: val=%+v err=%v", val, err)
	}
}

func TestSecretManagerBoundedInputs(t *testing.T) {
	_, s, p := secretsFixture(t)
	ctx, projectID, envID, folderID := createScope(t, p, s, 1)

	big := strings.Repeat("x", 64*1024+1)
	if _, err := s.CreateSecret(ctx, projectID, envID, folderID, "BIG", big); err == nil {
		t.Fatal("oversized value accepted")
	}
	if _, err := s.UpdateSecret(ctx, projectID, envID, folderID, "TOKEN", big); err == nil {
		t.Fatal("oversized update accepted")
	}
	for _, badKey := range []string{"", "lowercase", "1STARTS_WITH_DIGIT", "HAS SPACE", strings.Repeat("K", 129)} {
		if _, err := s.CreateSecret(ctx, projectID, envID, folderID, badKey, "v"); err == nil {
			t.Fatalf("invalid key %q accepted", badKey)
		}
	}
	if _, err := s.ReadSecretVersionValue(ctx, projectID, envID, folderID, "TOKEN", 0); err == nil {
		t.Fatal("version 0 accepted")
	}
	if _, err := s.ReadSecretVersionValue(ctx, projectID, envID, folderID, "TOKEN", -1); err == nil {
		t.Fatal("negative version accepted")
	}
	if _, err := s.EnsureFolder(ctx, projectID, envID, strings.Repeat("f", 201)); err == nil {
		t.Fatal("oversized folder name accepted")
	}
	if _, err := s.EnsureFolder(ctx, projectID, envID, "  "); err == nil {
		t.Fatal("blank folder name accepted")
	}
}
