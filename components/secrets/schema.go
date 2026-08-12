package secrets

// Schema migrations for the native secret manager. Every statement is
// accepted by both the SQLite and PostgreSQL DBer implementations and is
// idempotent, so Start can run it on every boot. Projects and environments
// tables are owned by the projects component; the secrets component declares
// a kernel dependency on "projects" so those tables always exist first.
//
// Storage is ciphertext-only: secret_versions holds nonce, ciphertext, and key
// identifiers but never plaintext, and secrets holds metadata only.

const foldersMigration = `CREATE TABLE IF NOT EXISTS secret_folders (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	environment_id TEXT NOT NULL,
	name TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (project_id, environment_id, name),
	FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
	FOREIGN KEY (environment_id) REFERENCES project_environments(id) ON DELETE CASCADE
)`

const keyRecordsMigration = `CREATE TABLE IF NOT EXISTS project_key_records (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	key_id TEXT NOT NULL,
	algorithm TEXT NOT NULL,
	encrypted_data_key TEXT NOT NULL,
	state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'rotating', 'retired')),
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	retired_at TEXT,
	UNIQUE (project_id, key_id),
	FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
)`

const secretsMigration = `CREATE TABLE IF NOT EXISTS secrets (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	environment_id TEXT NOT NULL,
	folder_id TEXT NOT NULL,
	key TEXT NOT NULL,
	current_version INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (folder_id, key),
	FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
	FOREIGN KEY (environment_id) REFERENCES project_environments(id) ON DELETE CASCADE,
	FOREIGN KEY (folder_id) REFERENCES secret_folders(id) ON DELETE CASCADE
)`

const versionsMigration = `CREATE TABLE IF NOT EXISTS secret_versions (
	id TEXT PRIMARY KEY,
	secret_id TEXT NOT NULL,
	version_no INTEGER NOT NULL,
	key_record_id TEXT NOT NULL,
	key_id TEXT NOT NULL,
	algorithm TEXT NOT NULL,
	nonce TEXT NOT NULL,
	ciphertext TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (secret_id, version_no),
	FOREIGN KEY (secret_id) REFERENCES secrets(id) ON DELETE CASCADE,
	FOREIGN KEY (key_record_id) REFERENCES project_key_records(id)
)`

// indexMigrations includes the partial unique indexes that serialize key
// lifecycle: at most one 'active' and one 'rotating' key per project. Both
// SQLite and PostgreSQL support partial indexes, which gives the exact
// "unique active keys" guarantee without blocking rotation checkpoints.
var indexMigrations = []string{
	`CREATE INDEX IF NOT EXISTS idx_secret_folders_project ON secret_folders(project_id)`,
	`CREATE INDEX IF NOT EXISTS idx_project_key_records_project ON project_key_records(project_id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_project_key_records_active ON project_key_records(project_id) WHERE state = 'active'`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_project_key_records_rotating ON project_key_records(project_id) WHERE state = 'rotating'`,
	`CREATE INDEX IF NOT EXISTS idx_secrets_folder ON secrets(folder_id)`,
	`CREATE INDEX IF NOT EXISTS idx_secret_versions_secret ON secret_versions(secret_id)`,
}

// schemaMigrations are applied in order by Start.
var schemaMigrations = []string{
	foldersMigration,
	keyRecordsMigration,
	secretsMigration,
	versionsMigration,
}
