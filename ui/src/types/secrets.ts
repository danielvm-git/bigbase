// Project secret REST types (e89s04/e89s05).
//
// SecretMetadata and SecretVersionMetadata are metadata-only projections:
// they deliberately contain NO plaintext or ciphertext fields. List state,
// mutation forms, and version history render exclusively from these types.
// The plaintext-bearing SecretValue type is separate and is used only by the
// explicit /value reveal path, so a value response can never leak into list
// state or mutation rendering.

export interface Project {
  id: string
  org_id: number
  name: string
  created_at: string
  updated_at: string
}

export interface ProjectEnvironment {
  id: string
  project_id: string
  slug: string
  name: string
  created_at: string
  updated_at: string
}

/** Metadata-only secret projection returned by list/get/create/update. */
export interface SecretMetadata {
  id: string
  project_id: string
  environment_id: string
  folder_id: string
  key: string
  current_version: number
  /** Masked preview computed server-side; never a plaintext value. */
  value_preview: string
  created_at: string
  updated_at: string
}

/** Metadata-only immutable-version projection. Never carries a value. */
export interface SecretVersionMetadata {
  id: string
  version: number
  key_id: string
  algorithm: string
  created_at: string
}

/**
 * Explicit value-read response type. Intentionally distinct from
 * SecretMetadata so a reveal response is never reused as list state. Held in
 * the UI only while the reveal modal is open and cleared on close/unmount.
 */
export interface SecretValue {
  secret_id: string
  key: string
  version: number
  value: string
  key_id: string
  algorithm: string
}
