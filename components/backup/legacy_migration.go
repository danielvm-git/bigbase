package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/internal/envcrypto"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// Legacy value migration (e89s06).
//
// Native-first dual-read keeps legacy site_env_vars deployable through the
// EnvResolver compatibility layer while this explicit, resumable operator
// command re-encrypts/moves legacy rows into native Project secrets. The
// operator MUST declare the storage format of the legacy rows: BigBase never
// guesses whether a value is plaintext or ciphertext (ADR 0009). A run without
// an explicit mode stops before changing any row.

// LegacyValueMode selects how stored site_env_vars values are interpreted
// during migration.
type LegacyValueMode string

const (
	// LegacyModePlaintext treats value_encrypted as plaintext (no-op-mode rows
	// written when the deploy component ran without an encryption key).
	LegacyModePlaintext LegacyValueMode = "plaintext"
	// LegacyModeCiphertext treats value_encrypted as an AES-256-GCM envelope
	// encrypted under the legacy hex env encryption key.
	LegacyModeCiphertext LegacyValueMode = "ciphertext"
)

// Valid reports whether m is one of the explicit supported modes.
func (m LegacyValueMode) Valid() bool {
	switch m {
	case LegacyModePlaintext, LegacyModeCiphertext:
		return true
	default:
		return false
	}
}

// SecretWriter is the native secret-store seam consumed by legacy migration.
// The secrets component satisfies it structurally at the composition root.
type SecretWriter interface {
	EnsureFolder(ctx context.Context, projectID, environmentID, name string) (secrets.SecretFolder, error)
	CreateSecret(ctx context.Context, projectID, environmentID, folderID, key, value string) (secrets.SecretMetadata, error)
	UpdateSecret(ctx context.Context, projectID, environmentID, folderID, key, value string) (secrets.SecretMetadata, error)
}

// LegacyMigrationOptions configures one migration run. All fields except Mode
// may be zero-valued; Mode is required.
type LegacyMigrationOptions struct {
	// DB reads legacy site_env_vars rows, site/project metadata, and the
	// migration checkpoints.
	DB kernel.DBer
	// Secrets is the native SecretManager seam used for writes.
	Secrets SecretWriter
	// EnvEncryptionKey is the legacy hex env encryption key. Required when
	// Mode is LegacyModeCiphertext.
	EnvEncryptionKey string
	// Mode declares the storage format of the legacy rows. Required.
	Mode LegacyValueMode
	// SiteID restricts the run to one site; empty migrates every site with
	// legacy rows.
	SiteID string
}

// LegacyMigrationReport is the value-free outcome of a migration run. It
// carries keys and counts only — never plaintext or ciphertext.
type LegacyMigrationReport struct {
	Mode             LegacyValueMode `json:"mode"`
	Sites            []string        `json:"sites"`
	Migrated         int             `json:"migrated"`
	Updated          int             `json:"updated"`
	BuildOnlySkipped int             `json:"build_only_skipped"`
	SkippedKeys      []string        `json:"skipped_keys,omitempty"`
	UnattachedSites  []string        `json:"unattached_sites,omitempty"`
	ResumedSites     int             `json:"resumed_sites"`
}

// Sentinel errors for the explicit migration contract.
var (
	// ErrLegacyMigrationModeRequired is returned when the operator did not
	// declare the legacy storage format. Nothing is written.
	ErrLegacyMigrationModeRequired = errors.New("legacy migration requires an explicit value format mode: plaintext or ciphertext")
	// ErrLegacyMigrationUnknownFormat is returned when a row cannot be
	// decrypted under the declared mode. The wrapped error names the site and
	// key — never the value.
	ErrLegacyMigrationUnknownFormat = errors.New("legacy value has unknown format")
	// ErrLegacyMigrationCipherKeyRequired is returned when ciphertext mode is
	// selected without the legacy env encryption key.
	ErrLegacyMigrationCipherKeyRequired = errors.New("legacy migration in ciphertext mode requires the legacy env encryption key")
)

// checkpoint state values.
const (
	checkpointInProgress = "in_progress"
	checkpointStateDone  = "done"
)

const checkpointsMigration = `CREATE TABLE IF NOT EXISTS legacy_migration_checkpoints (
	site_id TEXT PRIMARY KEY,
	mode TEXT NOT NULL,
	state TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`

// MigrateLegacyValues migrates legacy site_env_vars rows into native Project
// secrets for the selected sites. It is resumable: sites whose checkpoint is
// marked done for the same mode are skipped, so an interrupted run resumes
// from the next incomplete site. Legacy rows are never deleted — dual-read
// keeps them deployable until the operator verifies and removes them
// separately.
func MigrateLegacyValues(ctx context.Context, opts LegacyMigrationOptions) (*LegacyMigrationReport, error) {
	if opts.DB == nil {
		return nil, errors.New("backup: DB is required")
	}
	if opts.Secrets == nil {
		return nil, errors.New("backup: native secret store seam is required")
	}
	// SC-e89s06-P0-05: without an explicit format mode the migration stops
	// before changing any row.
	if !opts.Mode.Valid() {
		return nil, ErrLegacyMigrationModeRequired
	}
	var envKey []byte
	if opts.Mode == LegacyModeCiphertext {
		key, err := envcrypto.ParseKey(opts.EnvEncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrLegacyMigrationCipherKeyRequired, err)
		}
		if key == nil {
			return nil, ErrLegacyMigrationCipherKeyRequired
		}
		envKey = key
	}

	if err := opts.DB.Migrate(checkpointsMigration); err != nil {
		return nil, fmt.Errorf("ensure migration checkpoints: %w", err)
	}

	sites, err := migrationSiteIDs(ctx, opts.DB, opts.SiteID)
	if err != nil {
		return nil, err
	}

	report := &LegacyMigrationReport{Mode: opts.Mode}
	for _, siteID := range sites {
		done, err := checkpointDone(ctx, opts.DB, siteID, opts.Mode)
		if err != nil {
			return nil, err
		}
		if done {
			report.ResumedSites++
			continue
		}
		if err := migrateSite(ctx, opts, envKey, siteID, report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

// migrateSite migrates one site's legacy rows. The site is validated as a
// whole before any write: an undecryptable row aborts the site with no rows
// changed (SC-e89s06-P0-05) while earlier checkpoints stay migrated.
func migrateSite(ctx context.Context, opts LegacyMigrationOptions, envKey []byte, siteID string, report *LegacyMigrationReport) error {
	rows, err := opts.DB.QueryContext(ctx,
		`SELECT key, value_encrypted, is_runtime
		 FROM site_env_vars WHERE site_id = ? ORDER BY key`, siteID)
	if err != nil {
		if isNoSuchTable(err) {
			return nil
		}
		return fmt.Errorf("read legacy values for site %s: %w", siteID, err)
	}
	type legacyRow struct {
		key, encrypted string
		runtime        bool
	}
	var legacy []legacyRow
	for rows.Next() {
		var r legacyRow
		var rt int
		if err := rows.Scan(&r.key, &r.encrypted, &rt); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan legacy value for site %s: %w", siteID, err)
		}
		r.runtime = rt == 1
		legacy = append(legacy, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate legacy values for site %s: %w", siteID, err)
	}
	_ = rows.Close()

	if len(legacy) == 0 {
		return markCheckpoint(ctx, opts.DB, siteID, opts.Mode, checkpointStateDone)
	}

	projectID, envID, orgID, err := siteProjectEnvironment(ctx, opts.DB, siteID)
	if err != nil {
		return err
	}
	if projectID == "" {
		// No project attachment yet: dual-read keeps the site deployable.
		// Leave the checkpoint unmarked so a later run picks it up once the
		// site is attached.
		report.UnattachedSites = append(report.UnattachedSites, siteID)
		return nil
	}

	// Phase 1 — validate every runtime row before writing anything. Build-only
	// rows are not migrated: the native model has no build/runtime flag, and
	// moving a build-only value into the project environment would leak it into
	// runtime. They stay in the legacy store, which keeps supplying them at
	// build time through the compatibility layer.
	type pending struct {
		key, value string
	}
	var toWrite []pending
	for _, r := range legacy {
		if !r.runtime {
			report.BuildOnlySkipped++
			report.SkippedKeys = append(report.SkippedKeys, r.key)
			continue
		}
		value, err := envcrypto.Decrypt(envKey, r.encrypted)
		if err != nil {
			return fmt.Errorf("%w: site %s key %s cannot be decrypted as %s", ErrLegacyMigrationUnknownFormat, siteID, r.key, opts.Mode)
		}
		toWrite = append(toWrite, pending{key: r.key, value: value})
	}
	if len(toWrite) == 0 {
		return markCheckpoint(ctx, opts.DB, siteID, opts.Mode, checkpointStateDone)
	}

	if err := markCheckpoint(ctx, opts.DB, siteID, opts.Mode, checkpointInProgress); err != nil {
		return err
	}

	// Phase 2 — write. Tenant identity comes from the persisted site → project
	// chain, never from caller input.
	secCtx := auth.WithOrgID(ctx, orgID)
	folder, err := opts.Secrets.EnsureFolder(secCtx, projectID, envID, "default")
	if err != nil {
		return fmt.Errorf("ensure default folder for site %s: %w", siteID, err)
	}
	for _, p := range toWrite {
		_, err := opts.Secrets.CreateSecret(secCtx, projectID, envID, folder.ID, p.key, p.value)
		if err == nil {
			report.Migrated++
			continue
		}
		if errors.Is(err, secrets.ErrSecretAlreadyExists) {
			// Resumable idempotency: an earlier interrupted run may have
			// written this key. Update in place so the migrated value is the
			// current legacy value.
			if _, uerr := opts.Secrets.UpdateSecret(secCtx, projectID, envID, folder.ID, p.key, p.value); uerr != nil {
				return fmt.Errorf("update legacy value %s for site %s: %w", p.key, siteID, uerr)
			}
			report.Updated++
			continue
		}
		// An invalid key or oversized value cannot be stored natively. It stays
		// in the legacy store (still deployable through dual-read) and is
		// reported by name so the operator can fix it.
		report.SkippedKeys = append(report.SkippedKeys, p.key)
	}

	if err := markCheckpoint(ctx, opts.DB, siteID, opts.Mode, checkpointStateDone); err != nil {
		return err
	}
	report.Sites = append(report.Sites, siteID)
	return nil
}

// migrationSiteIDs returns the ordered set of sites with legacy rows. A
// SiteID option restricts the set to that site (even when it has no rows yet).
func migrationSiteIDs(ctx context.Context, db kernel.DBer, siteID string) ([]string, error) {
	if siteID != "" {
		return []string{siteID}, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT site_id FROM site_env_vars ORDER BY site_id`)
	if err != nil {
		if isNoSuchTable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list sites with legacy values: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan legacy site id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy site ids: %w", err)
	}
	return out, nil
}

// siteProjectEnvironment mirrors the deploy resolver's chain lookup: the
// site's project, its production environment, and the project's org. Empty
// results mean the site has no attachment yet (legacy-only mode). Missing
// tables degrade to empty.
func siteProjectEnvironment(ctx context.Context, db kernel.DBer, siteID string) (projectID, envID string, orgID int64, err error) {
	var project sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT project_id FROM sites WHERE id = ?`, siteID).Scan(&project); err != nil {
		// Older deployments may lack the sites table entirely or predate the
		// project_id attachment column — treat as unattached.
		if isNoSuchTable(err) || strings.Contains(err.Error(), "no such column: project_id") || errors.Is(err, sql.ErrNoRows) {
			return "", "", 0, nil
		}
		return "", "", 0, fmt.Errorf("resolve site project: %w", err)
	}
	if !project.Valid || project.String == "" {
		return "", "", 0, nil
	}
	projectID = project.String

	if err := db.QueryRowContext(ctx, `SELECT org_id FROM projects WHERE id = ?`, projectID).Scan(&orgID); err != nil {
		if isNoSuchTable(err) || errors.Is(err, sql.ErrNoRows) {
			return "", "", 0, nil
		}
		return "", "", 0, fmt.Errorf("resolve project org: %w", err)
	}

	if err := db.QueryRowContext(ctx,
		`SELECT id FROM project_environments WHERE project_id = ? AND slug = 'production' ORDER BY created_at, id LIMIT 1`,
		projectID).Scan(&envID); err != nil {
		if errors.Is(err, sql.ErrNoRows) || isNoSuchTable(err) {
			return projectID, "", orgID, nil
		}
		return "", "", 0, fmt.Errorf("resolve project environment: %w", err)
	}
	return projectID, envID, orgID, nil
}

// checkpointDone reports whether the site's checkpoint is marked done for the
// same mode. A checkpoint recorded under a different mode is ignored so the
// operator can re-run migration explicitly after a format change.
func checkpointDone(ctx context.Context, db kernel.DBer, siteID string, mode LegacyValueMode) (bool, error) {
	var state string
	err := db.QueryRowContext(ctx,
		`SELECT state FROM legacy_migration_checkpoints WHERE site_id = ? AND mode = ?`,
		siteID, string(mode)).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		if isNoSuchTable(err) {
			return false, nil
		}
		return false, fmt.Errorf("read migration checkpoint: %w", err)
	}
	return state == checkpointStateDone, nil
}

// markCheckpoint upserts the site's checkpoint state for the given mode.
func markCheckpoint(ctx context.Context, db kernel.DBer, siteID string, mode LegacyValueMode, state string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO legacy_migration_checkpoints (site_id, mode, state, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(site_id) DO UPDATE SET mode = excluded.mode, state = excluded.state, updated_at = excluded.updated_at`,
		siteID, string(mode), state, now); err != nil {
		return fmt.Errorf("update migration checkpoint for site %s: %w", siteID, err)
	}
	return nil
}

// isNoSuchTable detects missing-table errors across the SQLite and PostgreSQL
// drivers (local copy keeps backup dependency-light).
func isNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "does not exist")
}
