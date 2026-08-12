package secrets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// EnsureFolder creates a folder if missing and returns it. The unique
// (project, environment, name) constraint makes concurrent calls converge on
// one row.
func (s *Secrets) EnsureFolder(ctx context.Context, projectID, environmentID, name string) (SecretFolder, error) {
	name, err := validateFolderName(name)
	if err != nil {
		return SecretFolder{}, err
	}
	if err := s.verifyScope(ctx, projectID, environmentID, ""); err != nil {
		return SecretFolder{}, err
	}
	return s.ensureFolder(ctx, projectID, environmentID, name)
}

// ListFolders lists folders in a project environment, ordered by name.
func (s *Secrets) ListFolders(ctx context.Context, projectID, environmentID string) ([]SecretFolder, error) {
	if err := s.verifyScope(ctx, projectID, environmentID, ""); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, environment_id, name, created_at
		FROM secret_folders WHERE project_id = ? AND environment_id = ?
		ORDER BY name`, projectID, environmentID)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []SecretFolder
	for rows.Next() {
		var f SecretFolder
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.EnvironmentID, &f.Name, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan folder: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate folders: %w", err)
	}
	return out, nil
}

// ListSecrets lists metadata for secrets in a folder. Values are masked
// server-side; plaintext and ciphertext never leave the store.
func (s *Secrets) ListSecrets(ctx context.Context, projectID, environmentID, folderID string) ([]SecretMetadata, error) {
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, environment_id, folder_id, key, current_version, created_at, updated_at
		FROM secrets WHERE folder_id = ? ORDER BY key`, folderID)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var rowsOut []secretRow
	for rows.Next() {
		var r secretRow
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.EnvironmentID, &r.FolderID, &r.Key,
			&r.CurrentVersion, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		rowsOut = append(rowsOut, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}
	// The cursor is closed before preview decryption, so the single pooled
	// connection is free for the per-row reads.
	items := make([]SecretMetadata, 0, len(rowsOut))
	for _, r := range rowsOut {
		meta, err := s.metadataFor(ctx, r)
		if err != nil {
			return nil, err
		}
		items = append(items, meta)
	}
	return items, nil
}

// GetSecretMetadata returns metadata for one secret, including a masked
// preview.
func (s *Secrets) GetSecretMetadata(ctx context.Context, projectID, environmentID, folderID, key string) (SecretMetadata, error) {
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return SecretMetadata{}, err
	}
	r, err := s.loadSecret(ctx, projectID, environmentID, folderID, key)
	if err != nil {
		return SecretMetadata{}, err
	}
	return s.metadataFor(ctx, r)
}

// ListVersions lists immutable version metadata, ordered ascending.
func (s *Secrets) ListVersions(ctx context.Context, projectID, environmentID, folderID, key string) ([]SecretVersionMetadata, error) {
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return nil, err
	}
	r, err := s.loadSecret(ctx, projectID, environmentID, folderID, key)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, version_no, key_id, algorithm, created_at
		FROM secret_versions WHERE secret_id = ? ORDER BY version_no`, r.ID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []SecretVersionMetadata
	for rows.Next() {
		var v SecretVersionMetadata
		if err := rows.Scan(&v.ID, &v.Version, &v.KeyID, &v.Algorithm, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}
	return out, nil
}

// CreateSecret stores the first immutable version of a secret under the
// project data key. The data key is created atomically on first use.
func (s *Secrets) CreateSecret(ctx context.Context, projectID, environmentID, folderID, key, value string) (SecretMetadata, error) {
	key, err := validateSecretKey(key)
	if err != nil {
		return SecretMetadata{}, err
	}
	if err := validateSecretValue(value); err != nil {
		return SecretMetadata{}, err
	}
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return SecretMetadata{}, err
	}
	rec, err := s.ensureActiveKey(ctx, projectID)
	if err != nil {
		return SecretMetadata{}, err
	}
	dataKey, err := s.kh.UnwrapProjectDataKey(projectID, rec.KeyID, rec.Algorithm, rec.EncryptedKey)
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	secretID, err := kernel.GenerateID()
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("generate secret id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	scope := Scope{
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		FolderID:      folderID,
		SecretID:      secretID,
		Version:       1,
	}
	nonce, ciphertext, err := s.kh.Seal(dataKey, scope, value)
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("encrypt secret: %w", err)
	}
	versionID, err := kernel.GenerateID()
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("generate version id: %w", err)
	}
	tx, err := s.beginTx(ctx)
	if err != nil {
		return SecretMetadata{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secrets (id, project_id, environment_id, folder_id, key, current_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
		secretID, projectID, environmentID, folderID, key, now, now); err != nil {
		if isUniqueViolation(err) {
			return SecretMetadata{}, fmt.Errorf("%w: key %s", ErrSecretAlreadyExists, key)
		}
		return SecretMetadata{}, fmt.Errorf("insert secret: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_versions (id, secret_id, version_no, key_record_id, key_id, algorithm, nonce, ciphertext, created_at)
		VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?)`,
		versionID, secretID, rec.ID, rec.KeyID, rec.Algorithm, nonce, ciphertext, now); err != nil {
		return SecretMetadata{}, fmt.Errorf("insert secret version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return SecretMetadata{}, fmt.Errorf("commit secret create: %w", err)
	}
	return s.GetSecretMetadata(ctx, projectID, environmentID, folderID, key)
}

// UpdateSecret appends an immutable version and makes it current. The row
// write happens first so concurrent updates serialize on the secrets row lock
// instead of both reading the same current_version.
func (s *Secrets) UpdateSecret(ctx context.Context, projectID, environmentID, folderID, key, value string) (SecretMetadata, error) {
	key, err := validateSecretKey(key)
	if err != nil {
		return SecretMetadata{}, err
	}
	if err := validateSecretValue(value); err != nil {
		return SecretMetadata{}, err
	}
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return SecretMetadata{}, err
	}
	secret, err := s.loadSecret(ctx, projectID, environmentID, folderID, key)
	if err != nil {
		return SecretMetadata{}, err
	}
	tx, err := s.beginTx(ctx)
	if err != nil {
		return SecretMetadata{}, err
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		UPDATE secrets SET current_version = current_version + 1, updated_at = ? WHERE id = ?`,
		now, secret.ID)
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("bump secret version: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return SecretMetadata{}, fmt.Errorf("%w: secret %s", ErrSecretNotFound, key)
	}
	var next int
	if err := tx.QueryRowContext(ctx,
		`SELECT current_version FROM secrets WHERE id = ?`, secret.ID).Scan(&next); err != nil {
		return SecretMetadata{}, fmt.Errorf("read next version: %w", err)
	}
	rec, err := s.loadActiveKeyQuery(ctx, tx, projectID)
	if err != nil {
		return SecretMetadata{}, err
	}
	dataKey, err := s.kh.UnwrapProjectDataKey(projectID, rec.KeyID, rec.Algorithm, rec.EncryptedKey)
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	scope := Scope{
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		FolderID:      folderID,
		SecretID:      secret.ID,
		Version:       next,
	}
	nonce, ciphertext, err := s.kh.Seal(dataKey, scope, value)
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("encrypt secret: %w", err)
	}
	versionID, err := kernel.GenerateID()
	if err != nil {
		return SecretMetadata{}, fmt.Errorf("generate version id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_versions (id, secret_id, version_no, key_record_id, key_id, algorithm, nonce, ciphertext, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		versionID, secret.ID, next, rec.ID, rec.KeyID, rec.Algorithm, nonce, ciphertext, now); err != nil {
		return SecretMetadata{}, fmt.Errorf("insert secret version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return SecretMetadata{}, fmt.Errorf("commit secret update: %w", err)
	}
	return s.GetSecretMetadata(ctx, projectID, environmentID, folderID, key)
}

// DeleteSecret removes a secret and all its versions.
func (s *Secrets) DeleteSecret(ctx context.Context, projectID, environmentID, folderID, key string) error {
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return err
	}
	secret, err := s.loadSecret(ctx, projectID, environmentID, folderID, key)
	if err != nil {
		return err
	}
	tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `DELETE FROM secrets WHERE id = ?`, secret.ID)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return fmt.Errorf("%w: secret %s", ErrSecretNotFound, key)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit secret delete: %w", err)
	}
	return nil
}

// ReadSecretValue returns the plaintext of the current version.
func (s *Secrets) ReadSecretValue(ctx context.Context, projectID, environmentID, folderID, key string) (SecretValue, error) {
	return s.readSecretValue(ctx, projectID, environmentID, folderID, key, 0)
}

// ReadSecretVersionValue returns the plaintext of a specific immutable version.
func (s *Secrets) ReadSecretVersionValue(ctx context.Context, projectID, environmentID, folderID, key string, version int) (SecretValue, error) {
	if version <= 0 {
		return SecretValue{}, fmt.Errorf("invalid version %d", version)
	}
	return s.readSecretValue(ctx, projectID, environmentID, folderID, key, version)
}

func (s *Secrets) readSecretValue(ctx context.Context, projectID, environmentID, folderID, key string, version int) (SecretValue, error) {
	if err := s.verifyScope(ctx, projectID, environmentID, folderID); err != nil {
		return SecretValue{}, err
	}
	secret, err := s.loadSecret(ctx, projectID, environmentID, folderID, key)
	if err != nil {
		return SecretValue{}, err
	}
	ver, err := s.loadVersion(ctx, secret.ID, version)
	if err != nil {
		return SecretValue{}, err
	}
	scope := Scope{
		ProjectID:     secret.ProjectID,
		EnvironmentID: secret.EnvironmentID,
		FolderID:      secret.FolderID,
		SecretID:      secret.ID,
		Version:       ver.Version,
	}
	plaintext, err := s.decryptVersion(ctx, s.db, scope, ver)
	if err != nil {
		return SecretValue{}, err
	}
	return SecretValue{
		SecretID:  secret.ID,
		Key:       secret.Key,
		Version:   ver.Version,
		Value:     plaintext,
		KeyID:     ver.KeyID,
		Algorithm: ver.Algorithm,
	}, nil
}

// beginTx starts a transaction through the optional kernel.TxBeginner seam.
func (s *Secrets) beginTx(ctx context.Context) (*sql.Tx, error) {
	tb, ok := s.db.(kernel.TxBeginner)
	if !ok {
		return nil, errors.New("database does not support transactions (kernel.TxBeginner)")
	}
	return tb.BeginTx(ctx, nil)
}
