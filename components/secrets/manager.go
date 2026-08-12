package secrets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/internal/envcrypto"
	"github.com/danielvm/bigbase/kernel"
)

// Sentinel errors. Decryption and key-material failures are distinct so tests
// and adapters can distinguish "not found" from "crypto failed closed" without
// ever reading plaintext out of an error.
var (
	ErrOrganizationRequired = errors.New("organization required")
	ErrProjectNotFound      = errors.New("project not found")
	ErrEnvironmentNotFound  = errors.New("environment not found")
	ErrFolderNotFound       = errors.New("folder not found")
	ErrSecretNotFound       = errors.New("secret not found")
	ErrSecretAlreadyExists  = errors.New("secret already exists")
	ErrVersionNotFound      = errors.New("secret version not found")
	ErrActiveKeyNotFound    = errors.New("project data key not found")
	ErrInvalidKeyMaterial   = errors.New("invalid key material")
	ErrDecryptionFailed     = errors.New("secret decryption failed")
)

// SecretFolder is a named scope within a project environment.
type SecretFolder struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	Name          string `json:"name"`
	CreatedAt     string `json:"created_at"`
}

// SecretMetadata is the metadata-only projection returned by list and mutation
// operations. It NEVER contains plaintext or ciphertext; ValuePreview is a
// masked preview computed server-side.
type SecretMetadata struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	EnvironmentID  string `json:"environment_id"`
	FolderID       string `json:"folder_id"`
	Key            string `json:"key"`
	CurrentVersion int    `json:"current_version"`
	ValuePreview   string `json:"value_preview"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// SecretValue is the explicit value-read result type. It is intentionally
// distinct from SecretMetadata so no caller can mistake a list or mutation
// response for a value read.
type SecretValue struct {
	SecretID  string `json:"secret_id"`
	Key       string `json:"key"`
	Version   int    `json:"version"`
	Value     string `json:"value"`
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
}

// SecretVersionMetadata is a metadata-only version projection. It never
// carries nonce, ciphertext, or plaintext.
type SecretVersionMetadata struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	CreatedAt string `json:"created_at"`
}

// ProjectKeyRecord is the metadata-only projection of a project data key
// record. Encrypted key material never leaves the store.
type ProjectKeyRecord struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	RetiredAt string `json:"retired_at,omitempty"`
}

// SecretManager is the typed public seam for project secret storage. It is
// consumed by the REST policy adapters (e89s04), the deployment resolver
// (e89s06), and the MCP tools (e89s07). List and mutation results are
// metadata-only; plaintext is available exclusively through the explicit
// value-read methods.
type SecretManager interface {
	// EnsureFolder creates a folder if missing and returns it.
	EnsureFolder(ctx context.Context, projectID, environmentID, name string) (SecretFolder, error)
	// ListFolders lists folders in a project environment, ordered by name.
	ListFolders(ctx context.Context, projectID, environmentID string) ([]SecretFolder, error)
	// ListSecrets lists metadata (masked previews only) in a folder.
	ListSecrets(ctx context.Context, projectID, environmentID, folderID string) ([]SecretMetadata, error)
	// GetSecretMetadata returns metadata for one secret, including a masked preview.
	GetSecretMetadata(ctx context.Context, projectID, environmentID, folderID, key string) (SecretMetadata, error)
	// ListVersions lists immutable version metadata, ordered ascending.
	ListVersions(ctx context.Context, projectID, environmentID, folderID, key string) ([]SecretVersionMetadata, error)
	// CreateSecret stores the first immutable version of a secret.
	CreateSecret(ctx context.Context, projectID, environmentID, folderID, key, value string) (SecretMetadata, error)
	// UpdateSecret appends an immutable version and makes it current.
	UpdateSecret(ctx context.Context, projectID, environmentID, folderID, key, value string) (SecretMetadata, error)
	// DeleteSecret removes a secret and all its versions.
	DeleteSecret(ctx context.Context, projectID, environmentID, folderID, key string) error
	// ReadSecretValue returns the plaintext of the current version.
	ReadSecretValue(ctx context.Context, projectID, environmentID, folderID, key string) (SecretValue, error)
	// ReadSecretVersionValue returns the plaintext of a specific immutable version.
	ReadSecretVersionValue(ctx context.Context, projectID, environmentID, folderID, key string, version int) (SecretValue, error)
	// BeginKeyRotation creates the rotating-key checkpoint for a project.
	BeginKeyRotation(ctx context.Context, projectID string) (ProjectKeyRecord, error)
	// RotateCurrentVersions re-encrypts current versions under the rotating key.
	RotateCurrentVersions(ctx context.Context, projectID string) (int, error)
	// CompleteKeyRotation verifies rotation, then retires the old key.
	CompleteKeyRotation(ctx context.Context, projectID string) error
}

// keyRecord is the internal row representation of a project data key record.
// EncryptedKey is base64(nonce || sealed-data-key) wrapped under the root key.
type keyRecord struct {
	ID           string
	ProjectID    string
	KeyID        string
	Algorithm    string
	EncryptedKey string
	State        string
	CreatedAt    string
	RetiredAt    string
}

func (r keyRecord) public() ProjectKeyRecord {
	return ProjectKeyRecord{
		ID:        r.ID,
		ProjectID: r.ProjectID,
		KeyID:     r.KeyID,
		Algorithm: r.Algorithm,
		State:     r.State,
		CreatedAt: r.CreatedAt,
		RetiredAt: r.RetiredAt,
	}
}

// versionRow is the internal row representation of a secret version.
type versionRow struct {
	ID          string
	SecretID    string
	Version     int
	KeyRecordID string
	KeyID       string
	Algorithm   string
	Nonce       string
	Ciphertext  string
	CreatedAt   string
}

// rowQueryer is satisfied by kernel.DBer and *sql.Tx so version and key
// decryption can run inside transactions.
type rowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// previewDecryptError is the visible sentinel preview for a version whose
// stored value cannot be decrypted (corruption or tampering). It is distinct
// from a short-value mask so operators can tell corruption from a four-char
// secret.
const previewDecryptError = "<decrypt error>"

var validSecretKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

const (
	maxSecretKeyLen   = 128
	maxSecretValueLen = 64 * 1024
	maxFolderNameLen  = 200
)

func validateSecretKey(key string) (string, error) {
	if key == "" || len(key) > maxSecretKeyLen || !validSecretKey.MatchString(key) {
		return "", fmt.Errorf("invalid secret key")
	}
	return key, nil
}

func validateSecretValue(value string) error {
	if len(value) > maxSecretValueLen {
		return fmt.Errorf("secret value exceeds %d bytes", maxSecretValueLen)
	}
	return nil
}

func validateFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("folder name is required")
	}
	if len(name) > maxFolderNameLen {
		return "", errors.New("folder name is too long")
	}
	return name, nil
}

func orgIDFromContext(ctx context.Context) (int64, error) {
	orgID, ok := auth.OrgIDFromContext(ctx)
	if !ok || orgID <= 0 {
		return 0, ErrOrganizationRequired
	}
	return orgID, nil
}

// requireProject verifies the project exists and belongs to the authenticated
// organization. Tenant identity always comes from context, never from caller
// arguments.
func (s *Secrets) requireProject(ctx context.Context, projectID string) error {
	orgID, err := orgIDFromContext(ctx)
	if err != nil {
		return err
	}
	var projectOrg int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT org_id FROM projects WHERE id = ?`, projectID).Scan(&projectOrg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: project %s", ErrProjectNotFound, projectID)
		}
		return fmt.Errorf("lookup project: %w", err)
	}
	if projectOrg != orgID {
		return fmt.Errorf("%w: project %s", ErrProjectNotFound, projectID)
	}
	return nil
}

// verifyScope validates the project → environment → folder chain and the
// authenticated organization. An empty folderID skips the folder check.
func (s *Secrets) verifyScope(ctx context.Context, projectID, environmentID, folderID string) error {
	if err := s.requireProject(ctx, projectID); err != nil {
		return err
	}
	var envProject string
	if err := s.db.QueryRowContext(ctx,
		`SELECT project_id FROM project_environments WHERE id = ?`, environmentID).Scan(&envProject); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: environment %s", ErrEnvironmentNotFound, environmentID)
		}
		return fmt.Errorf("lookup environment: %w", err)
	}
	if envProject != projectID {
		return fmt.Errorf("%w: environment %s", ErrEnvironmentNotFound, environmentID)
	}
	if folderID == "" {
		return nil
	}
	var folderProject, folderEnv string
	if err := s.db.QueryRowContext(ctx,
		`SELECT project_id, environment_id FROM secret_folders WHERE id = ?`, folderID).
		Scan(&folderProject, &folderEnv); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: folder %s", ErrFolderNotFound, folderID)
		}
		return fmt.Errorf("lookup folder: %w", err)
	}
	if folderProject != projectID || folderEnv != environmentID {
		return fmt.Errorf("%w: folder %s", ErrFolderNotFound, folderID)
	}
	return nil
}

// ensureFolder inserts a folder when missing. The unique (project, environment,
// name) constraint makes concurrent creation converge on one row.
func (s *Secrets) ensureFolder(ctx context.Context, projectID, environmentID, name string) (SecretFolder, error) {
	id, err := kernel.GenerateID()
	if err != nil {
		return SecretFolder{}, fmt.Errorf("generate folder id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO secret_folders (id, project_id, environment_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING`, id, projectID, environmentID, name, now, now); err != nil {
		return SecretFolder{}, fmt.Errorf("create folder: %w", err)
	}
	return s.getFolder(ctx, projectID, environmentID, name)
}

func (s *Secrets) getFolder(ctx context.Context, projectID, environmentID, name string) (SecretFolder, error) {
	var f SecretFolder
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, environment_id, name, created_at
		FROM secret_folders WHERE project_id = ? AND environment_id = ? AND name = ?`,
		projectID, environmentID, name).
		Scan(&f.ID, &f.ProjectID, &f.EnvironmentID, &f.Name, &f.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SecretFolder{}, fmt.Errorf("%w: folder %s", ErrFolderNotFound, name)
		}
		return SecretFolder{}, fmt.Errorf("lookup folder: %w", err)
	}
	return f, nil
}

// loadSecret resolves a secret row by folder and key.
func (s *Secrets) loadSecret(ctx context.Context, projectID, environmentID, folderID, key string) (secretRow, error) {
	var r secretRow
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, environment_id, folder_id, key, current_version, created_at, updated_at
		FROM secrets WHERE folder_id = ? AND key = ?`, folderID, key).
		Scan(&r.ID, &r.ProjectID, &r.EnvironmentID, &r.FolderID, &r.Key, &r.CurrentVersion, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return secretRow{}, fmt.Errorf("%w: secret %s", ErrSecretNotFound, key)
		}
		return secretRow{}, fmt.Errorf("lookup secret: %w", err)
	}
	// Defensive scope check: the row must sit under the requested chain.
	if r.ProjectID != projectID || r.EnvironmentID != environmentID || r.FolderID != folderID {
		return secretRow{}, fmt.Errorf("%w: secret %s", ErrSecretNotFound, key)
	}
	return r, nil
}

type secretRow struct {
	ID             string
	ProjectID      string
	EnvironmentID  string
	FolderID       string
	Key            string
	CurrentVersion int
	CreatedAt      string
	UpdatedAt      string
}

func (s *Secrets) metadataFor(ctx context.Context, r secretRow) (SecretMetadata, error) {
	meta := SecretMetadata{
		ID:             r.ID,
		ProjectID:      r.ProjectID,
		EnvironmentID:  r.EnvironmentID,
		FolderID:       r.FolderID,
		Key:            r.Key,
		CurrentVersion: r.CurrentVersion,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	if r.CurrentVersion <= 0 {
		return meta, nil
	}
	ver, err := s.loadVersion(ctx, r.ID, r.CurrentVersion)
	if err != nil {
		s.logger.Warn("secrets: load current version for preview", "secret", r.Key, "error", err)
		meta.ValuePreview = previewDecryptError
		return meta, nil
	}
	scope := Scope{
		ProjectID:     r.ProjectID,
		EnvironmentID: r.EnvironmentID,
		FolderID:      r.FolderID,
		SecretID:      r.ID,
		Version:       ver.Version,
	}
	plaintext, err := s.decryptVersion(ctx, s.db, scope, ver)
	if err != nil {
		s.logger.Warn("secrets: decrypt for preview", "secret", r.Key, "error", err)
		meta.ValuePreview = previewDecryptError
		return meta, nil
	}
	meta.ValuePreview = envcrypto.MaskValue(plaintext)
	return meta, nil
}

// loadVersion loads a version row; version 0 resolves the current version.
func (s *Secrets) loadVersion(ctx context.Context, secretID string, version int) (versionRow, error) {
	var v versionRow
	var err error
	if version > 0 {
		err = s.db.QueryRowContext(ctx, `
			SELECT id, secret_id, version_no, key_record_id, key_id, algorithm, nonce, ciphertext, created_at
			FROM secret_versions WHERE secret_id = ? AND version_no = ?`, secretID, version).
			Scan(&v.ID, &v.SecretID, &v.Version, &v.KeyRecordID, &v.KeyID, &v.Algorithm, &v.Nonce, &v.Ciphertext, &v.CreatedAt)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT id, secret_id, version_no, key_record_id, key_id, algorithm, nonce, ciphertext, created_at
			FROM secret_versions WHERE secret_id = ? AND version_no = (SELECT current_version FROM secrets WHERE id = ?)`,
			secretID, secretID).
			Scan(&v.ID, &v.SecretID, &v.Version, &v.KeyRecordID, &v.KeyID, &v.Algorithm, &v.Nonce, &v.Ciphertext, &v.CreatedAt)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return versionRow{}, fmt.Errorf("%w: secret %s version %d", ErrVersionNotFound, secretID, version)
		}
		return versionRow{}, fmt.Errorf("lookup version: %w", err)
	}
	return v, nil
}

// loadKeyRecord loads any key record (active, rotating, or retired) by id.
// Retired records must stay readable so old versions remain decryptable.
func (s *Secrets) loadKeyRecord(ctx context.Context, q rowQueryer, id string) (keyRecord, error) {
	var r keyRecord
	err := q.QueryRowContext(ctx, `
		SELECT id, project_id, key_id, algorithm, encrypted_data_key, state, created_at, COALESCE(retired_at, '')
		FROM project_key_records WHERE id = ?`, id).
		Scan(&r.ID, &r.ProjectID, &r.KeyID, &r.Algorithm, &r.EncryptedKey, &r.State, &r.CreatedAt, &r.RetiredAt)
	if err != nil {
		return keyRecord{}, fmt.Errorf("lookup key record: %w", err)
	}
	return r, nil
}

// decryptVersion authenticates and decrypts a version against its recorded
// scope and key record. Every failure closes without exposing plaintext.
func (s *Secrets) decryptVersion(ctx context.Context, q rowQueryer, scope Scope, ver versionRow) (string, error) {
	if ver.Algorithm != AlgorithmAES256GCM {
		return "", fmt.Errorf("%w: unsupported algorithm %q", ErrInvalidKeyMaterial, ver.Algorithm)
	}
	rec, err := s.loadKeyRecord(ctx, q, ver.KeyRecordID)
	if err != nil {
		return "", fmt.Errorf("%w: key record %s", ErrInvalidKeyMaterial, ver.KeyRecordID)
	}
	if rec.KeyID != ver.KeyID {
		return "", fmt.Errorf("%w: key id mismatch", ErrInvalidKeyMaterial)
	}
	dataKey, err := s.kh.UnwrapProjectDataKey(scope.ProjectID, rec.KeyID, rec.Algorithm, rec.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	plaintext, err := s.kh.Open(dataKey, scope, ver.Nonce, ver.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

// isUniqueViolation detects duplicate-key errors across the SQLite and
// PostgreSQL drivers.
func isUniqueViolation(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}
