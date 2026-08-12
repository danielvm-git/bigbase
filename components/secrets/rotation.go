package secrets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// loadActiveKeyQuery returns the project's active data key record through any
// row queryer (pool or transaction).
func (s *Secrets) loadActiveKeyQuery(ctx context.Context, q rowQueryer, projectID string) (keyRecord, error) {
	var r keyRecord
	err := q.QueryRowContext(ctx, `
		SELECT id, project_id, key_id, algorithm, encrypted_data_key, state, created_at, COALESCE(retired_at, '')
		FROM project_key_records WHERE project_id = ? AND state = 'active'`, projectID).
		Scan(&r.ID, &r.ProjectID, &r.KeyID, &r.Algorithm, &r.EncryptedKey, &r.State, &r.CreatedAt, &r.RetiredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return keyRecord{}, fmt.Errorf("%w: project %s", ErrActiveKeyNotFound, projectID)
		}
		return keyRecord{}, fmt.Errorf("lookup active key: %w", err)
	}
	return r, nil
}

func (s *Secrets) loadActiveKey(ctx context.Context, projectID string) (keyRecord, error) {
	return s.loadActiveKeyQuery(ctx, s.db, projectID)
}

// loadRotatingKey returns the project's rotation checkpoint record.
func (s *Secrets) loadRotatingKey(ctx context.Context, projectID string) (keyRecord, error) {
	var r keyRecord
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, key_id, algorithm, encrypted_data_key, state, created_at, COALESCE(retired_at, '')
		FROM project_key_records WHERE project_id = ? AND state = 'rotating'`, projectID).
		Scan(&r.ID, &r.ProjectID, &r.KeyID, &r.Algorithm, &r.EncryptedKey, &r.State, &r.CreatedAt, &r.RetiredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return keyRecord{}, fmt.Errorf("no rotating key for project %s", projectID)
		}
		return keyRecord{}, fmt.Errorf("lookup rotating key: %w", err)
	}
	return r, nil
}

// ensureActiveKey returns the project's active data key, creating it
// atomically on first use. The partial unique index on active records is the
// hard guarantee that exactly one key exists; the INSERT runs inside a
// transaction and ignores conflicts so concurrent first writes converge on the
// same record instead of failing or duplicating keys.
func (s *Secrets) ensureActiveKey(ctx context.Context, projectID string) (keyRecord, error) {
	rec, err := s.loadActiveKey(ctx, projectID)
	if err == nil {
		return rec, nil
	}
	if !errors.Is(err, ErrActiveKeyNotFound) {
		return keyRecord{}, err
	}
	keyID, algorithm, wrapped, err := s.kh.GenerateProjectDataKey(projectID)
	if err != nil {
		return keyRecord{}, err
	}
	id, err := kernel.GenerateID()
	if err != nil {
		return keyRecord{}, fmt.Errorf("generate key record id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.beginTx(ctx)
	if err != nil {
		return keyRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key, state, created_at, retired_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, NULL)
		ON CONFLICT DO NOTHING`, id, projectID, keyID, algorithm, wrapped, now); err != nil {
		return keyRecord{}, fmt.Errorf("insert project data key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return keyRecord{}, fmt.Errorf("commit project data key: %w", err)
	}
	return s.loadActiveKey(ctx, projectID)
}

// BeginKeyRotation creates the rotating-key checkpoint for a project. It is
// idempotent and resumable: calling it again after an interruption returns the
// existing rotating record. The old active key is untouched until
// CompleteKeyRotation, so an interrupted rotation never loses decryptability.
func (s *Secrets) BeginKeyRotation(ctx context.Context, projectID string) (ProjectKeyRecord, error) {
	if err := s.requireProject(ctx, projectID); err != nil {
		return ProjectKeyRecord{}, err
	}
	if rec, err := s.loadRotatingKey(ctx, projectID); err == nil {
		return rec.public(), nil
	}
	keyID, algorithm, wrapped, err := s.kh.GenerateProjectDataKey(projectID)
	if err != nil {
		return ProjectKeyRecord{}, err
	}
	id, err := kernel.GenerateID()
	if err != nil {
		return ProjectKeyRecord{}, fmt.Errorf("generate key record id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.beginTx(ctx)
	if err != nil {
		return ProjectKeyRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO project_key_records (id, project_id, key_id, algorithm, encrypted_data_key, state, created_at, retired_at)
		VALUES (?, ?, ?, ?, ?, 'rotating', ?, NULL)
		ON CONFLICT DO NOTHING`, id, projectID, keyID, algorithm, wrapped, now); err != nil {
		return ProjectKeyRecord{}, fmt.Errorf("insert rotating key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ProjectKeyRecord{}, fmt.Errorf("commit rotating key: %w", err)
	}
	rec, err := s.loadRotatingKey(ctx, projectID)
	if err != nil {
		return ProjectKeyRecord{}, err
	}
	return rec.public(), nil
}

// RotateCurrentVersions re-encrypts every secret's current version under the
// rotating key inside one transaction. Older versions keep the old key and
// remain decryptable. The old active key is not retired by this step.
func (s *Secrets) RotateCurrentVersions(ctx context.Context, projectID string) (int, error) {
	if err := s.requireProject(ctx, projectID); err != nil {
		return 0, err
	}
	if _, err := s.loadActiveKey(ctx, projectID); err != nil {
		return 0, err
	}
	rot, err := s.loadRotatingKey(ctx, projectID)
	if err != nil {
		return 0, err
	}
	rotKey, err := s.kh.UnwrapProjectDataKey(projectID, rot.KeyID, rot.Algorithm, rot.EncryptedKey)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	tx, err := s.beginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	// Buffer the target rows and close the cursor before writing, so the
	// transaction connection is never used for two statements at once.
	type target struct {
		ver   versionRow
		scope Scope
	}
	var targets []target
	rows, err := tx.QueryContext(ctx, `
		SELECT sv.id, sv.secret_id, sv.version_no, sv.key_record_id, sv.key_id, sv.algorithm,
		       sv.nonce, sv.ciphertext, sv.created_at,
		       sec.project_id, sec.environment_id, sec.folder_id
		FROM secret_versions sv
		JOIN secrets sec ON sec.id = sv.secret_id
		WHERE sec.project_id = ? AND sv.version_no = sec.current_version
		ORDER BY sv.secret_id, sv.version_no`, projectID)
	if err != nil {
		return 0, fmt.Errorf("select current versions: %w", err)
	}
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.ver.ID, &t.ver.SecretID, &t.ver.Version, &t.ver.KeyRecordID,
			&t.ver.KeyID, &t.ver.Algorithm, &t.ver.Nonce, &t.ver.Ciphertext, &t.ver.CreatedAt,
			&t.scope.ProjectID, &t.scope.EnvironmentID, &t.scope.FolderID); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("scan current version: %w", err)
		}
		t.scope.SecretID = t.ver.SecretID
		t.scope.Version = t.ver.Version
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, fmt.Errorf("iterate current versions: %w", err)
	}
	_ = rows.Close()

	for _, t := range targets {
		plaintext, err := s.decryptVersion(ctx, tx, t.scope, t.ver)
		if err != nil {
			return 0, err
		}
		nonce, ciphertext, err := s.kh.Seal(rotKey, t.scope, plaintext)
		if err != nil {
			return 0, fmt.Errorf("re-encrypt version %d: %w", t.ver.Version, err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE secret_versions SET key_record_id = ?, key_id = ?, nonce = ?, ciphertext = ?
			WHERE id = ?`, rot.ID, rot.KeyID, nonce, ciphertext, t.ver.ID); err != nil {
			return 0, fmt.Errorf("update version %d: %w", t.ver.Version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit rotation: %w", err)
	}
	return len(targets), nil
}

// CompleteKeyRotation verifies that every current version references the
// rotating key, spot-checks decryptability, and only then retires the old key
// and promotes the rotating key to active — in that order, in one transaction.
func (s *Secrets) CompleteKeyRotation(ctx context.Context, projectID string) error {
	if err := s.requireProject(ctx, projectID); err != nil {
		return err
	}
	active, err := s.loadActiveKey(ctx, projectID)
	if err != nil {
		return err
	}
	rot, err := s.loadRotatingKey(ctx, projectID)
	if err != nil {
		return err
	}

	var secretsTotal, reencrypted int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM secrets WHERE project_id = ?`, projectID).Scan(&secretsTotal); err != nil {
		return fmt.Errorf("count secrets: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM secret_versions sv
		JOIN secrets sec ON sec.id = sv.secret_id
		WHERE sec.project_id = ? AND sv.version_no = sec.current_version AND sv.key_record_id = ?`,
		projectID, rot.ID).Scan(&reencrypted); err != nil {
		return fmt.Errorf("count re-encrypted versions: %w", err)
	}
	if secretsTotal != reencrypted {
		return fmt.Errorf("rotation incomplete: %d of %d current versions re-encrypted", reencrypted, secretsTotal)
	}

	// Spot-check: decrypt one current version under the rotating key.
	if secretsTotal > 0 {
		rows, err := s.db.QueryContext(ctx, `
			SELECT sv.id, sv.secret_id, sv.version_no, sv.key_record_id, sv.key_id, sv.algorithm,
			       sv.nonce, sv.ciphertext, sv.created_at,
			       sec.project_id, sec.environment_id, sec.folder_id
			FROM secret_versions sv
			JOIN secrets sec ON sec.id = sv.secret_id
			WHERE sec.project_id = ? AND sv.version_no = sec.current_version
			LIMIT 1`, projectID)
		if err != nil {
			return fmt.Errorf("select spot-check version: %w", err)
		}
		var v versionRow
		var scope Scope
		if !rows.Next() {
			_ = rows.Close()
			return errors.New("rotation verification: no current version found")
		}
		if err := rows.Scan(&v.ID, &v.SecretID, &v.Version, &v.KeyRecordID, &v.KeyID, &v.Algorithm,
			&v.Nonce, &v.Ciphertext, &v.CreatedAt,
			&scope.ProjectID, &scope.EnvironmentID, &scope.FolderID); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan spot-check version: %w", err)
		}
		_ = rows.Close()
		scope.SecretID = v.SecretID
		scope.Version = v.Version
		rotKey, err := s.kh.UnwrapProjectDataKey(projectID, rot.KeyID, rot.Algorithm, rot.EncryptedKey)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
		}
		if _, err := s.kh.Open(rotKey, scope, v.Nonce, v.Ciphertext); err != nil {
			return fmt.Errorf("%w: rotation spot-check: %v", ErrDecryptionFailed, err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Retire first so the partial unique index never sees two active records.
	if _, err := tx.ExecContext(ctx, `
		UPDATE project_key_records SET state = 'retired', retired_at = ? WHERE id = ?`,
		now, active.ID); err != nil {
		return fmt.Errorf("retire old key: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE project_key_records SET state = 'active', retired_at = NULL WHERE id = ?`,
		rot.ID); err != nil {
		return fmt.Errorf("promote rotating key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rotation completion: %w", err)
	}
	return nil
}
