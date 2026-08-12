import type {
  Project,
  ProjectEnvironment,
  SecretMetadata,
  SecretValue,
  SecretVersionMetadata,
} from '../types/secrets'

// secretsData.ts — fetch helpers for the e89s04 Project secret REST surface.
//
// List/mutation/version helpers return metadata-only projections; the only
// helper that carries a plaintext value is readSecretValue, which targets the
// explicit /value route and returns the response status so callers can render
// value-free 401/403 errors.

/** Security-bound batch limit for a single .env import (e89s04 §13). */
export const MAX_SECRET_BATCH_KEYS = 1000
/** Server-side key/value bounds (e89s04 §13), mirrored for client validation. */
export const MAX_SECRET_KEY_BYTES = 128
export const MAX_SECRET_VALUE_BYTES = 64 * 1024

const SECRET_KEY_RE = /^[A-Z][A-Z0-9_]*$/

async function fetchJSON<T>(url: string): Promise<{ ok: boolean; status: number; data: T }> {
  try {
    const res = await fetch(url)
    const data = (await res.json()) as T
    return { ok: res.ok, status: res.status, data }
  } catch {
    return { ok: false, status: 0, data: {} as T }
  }
}

// --- Project / Environment navigation (e89s02) ---

export async function listProjects(): Promise<Project[]> {
  const { ok, data } = await fetchJSON<{ data: Project[] }>('/api/projects')
  if (ok && data && Array.isArray(data.data)) return data.data
  return []
}

export async function getProject(projectId: string): Promise<Project | null> {
  const { ok, data } = await fetchJSON<{ data: Project }>(`/api/projects/${encodeURIComponent(projectId)}`)
  if (ok && data && data.data) return data.data
  return null
}

export async function listProjectEnvironments(projectId: string): Promise<ProjectEnvironment[]> {
  const { ok, data } = await fetchJSON<{ data: ProjectEnvironment[] }>(
    `/api/projects/${encodeURIComponent(projectId)}/environments`,
  )
  if (ok && data && Array.isArray(data.data)) return data.data
  return []
}

// --- Secret metadata (list/get/mutations) — never carries a value ---

export async function listSecrets(projectId: string, envId: string): Promise<SecretMetadata[]> {
  const { ok, data } = await fetchJSON<{ data: SecretMetadata[] }>(
    `/api/projects/${encodeURIComponent(projectId)}/environments/${encodeURIComponent(envId)}/secrets`,
  )
  if (ok && data && Array.isArray(data.data)) return data.data
  return []
}

export async function createSecret(
  projectId: string,
  envId: string,
  payload: { key: string; value: string },
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(
      `/api/projects/${encodeURIComponent(projectId)}/environments/${encodeURIComponent(envId)}/secrets`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      },
    )
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
    if (!res.ok) return { ok: false, error: valueFreeError(res.status, body.error) }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function updateSecret(
  projectId: string,
  envId: string,
  key: string,
  payload: { value: string },
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(
      `/api/projects/${encodeURIComponent(projectId)}/environments/${encodeURIComponent(envId)}/secrets/${encodeURIComponent(key)}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      },
    )
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
    if (!res.ok) return { ok: false, error: valueFreeError(res.status, body.error) }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function deleteSecret(
  projectId: string,
  envId: string,
  key: string,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(
      `/api/projects/${encodeURIComponent(projectId)}/environments/${encodeURIComponent(envId)}/secrets/${encodeURIComponent(key)}`,
      { method: 'DELETE' },
    )
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: valueFreeError(res.status, body.error) }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

// --- Version history (metadata only) ---

export async function listSecretVersions(
  projectId: string,
  envId: string,
  key: string,
): Promise<SecretVersionMetadata[]> {
  const { ok, data } = await fetchJSON<{ data: SecretVersionMetadata[] }>(
    `/api/projects/${encodeURIComponent(projectId)}/environments/${encodeURIComponent(envId)}/secrets/${encodeURIComponent(key)}/versions`,
  )
  if (ok && data && Array.isArray(data.data)) return data.data
  return []
}

// --- Explicit value read — the ONLY path that returns a plaintext value ---

export interface ReadSecretValueResult {
  ok: boolean
  data?: SecretValue
  status?: number
  error?: string
}

export async function readSecretValue(
  projectId: string,
  envId: string,
  key: string,
): Promise<ReadSecretValueResult> {
  try {
    const res = await fetch(
      `/api/projects/${encodeURIComponent(projectId)}/environments/${encodeURIComponent(envId)}/secrets/${encodeURIComponent(key)}/value`,
    )
    const body = await res.json().catch(() => ({ error: '' })) as { data?: SecretValue; error?: string }
    if (!res.ok) {
      // 401/403 map to value-free messages; the server error text is a fixed
      // generic string and is never echoed with submitted values.
      return { ok: false, status: res.status, error: valueFreeError(res.status, body.error) }
    }
    if (!body.data || typeof body.data.value !== 'string') {
      return { ok: false, status: 500, error: 'Unexpected response' }
    }
    return { ok: true, status: res.status, data: body.data }
  } catch {
    return { ok: false, status: 0, error: 'Network error' }
  }
}

/**
 * Maps an API error to a value-free message. Forbidden/unauthorized responses
 * always render a fixed message; anything else falls back to the server text,
 * which is a fixed generic string and never contains submitted values.
 */
function valueFreeError(status: number | undefined, serverError: string | undefined): string {
  if (status === 401) return 'You must be signed in to perform this action.'
  if (status === 403) return "You don't have permission to perform this action."
  if (status === 404) return 'Not found.'
  return serverError || `Failed (HTTP ${status ?? 0})`
}

// --- Safe .env import (e89s05 §13) ---

export interface EnvPair {
  key: string
  value: string
}

/**
 * Parses dotenv-style text into key/value pairs. Lines that are blank,
 * comments, or lack a KEY=VALUE shape are skipped. Quoted values are
 * unquoted. This is the same shape as the legacy Site import parser.
 */
export function parseEnvFile(text: string): EnvPair[] {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(line => line && !line.startsWith('#'))
    .flatMap(line => {
      const eq = line.indexOf('=')
      if (eq < 1) return []
      const key = line.slice(0, eq).trim()
      const raw = line.slice(eq + 1).trim()
      const value = raw.startsWith('"') && raw.endsWith('"') ? raw.slice(1, -1) : raw
      return [{ key, value }]
    })
}

function isValidSecretKey(key: string): boolean {
  if (!key || key.length > MAX_SECRET_KEY_BYTES) return false
  return SECRET_KEY_RE.test(key)
}

export interface ImportSecretsResult {
  created: number
  updated: number
  /** Keys that failed to import, by name only — values are never reported. */
  failedKeys: string[]
  /** Keys beyond the batch limit that were not attempted. */
  skipped: number
}

/**
 * Imports .env pairs as individual validated writes, bounded by
 * MAX_SECRET_BATCH_KEYS. Existing keys are updated; new keys are created.
 * Failures are reported by key name only — submitted values are never echoed
 * in the result or any error message (SC-e89s05-P1-05).
 */
export async function importEnvSecrets(
  projectId: string,
  envId: string,
  text: string,
  existingKeys: ReadonlySet<string>,
): Promise<ImportSecretsResult> {
  const pairs = parseEnvFile(text)
  const result: ImportSecretsResult = { created: 0, updated: 0, failedKeys: [], skipped: 0 }

  const bounded = pairs.slice(0, MAX_SECRET_BATCH_KEYS)
  result.skipped = Math.max(0, pairs.length - MAX_SECRET_BATCH_KEYS)

  for (const pair of bounded) {
    // Client-side validation mirrors the server bounds so invalid keys are
    // reported by name without issuing a write or echoing the value.
    if (!isValidSecretKey(pair.key) || pair.value.length > MAX_SECRET_VALUE_BYTES) {
      result.failedKeys.push(pair.key)
      continue
    }
    const res = existingKeys.has(pair.key)
      ? await updateSecret(projectId, envId, pair.key, { value: pair.value })
      : await createSecret(projectId, envId, { key: pair.key, value: pair.value })
    if (!res.ok) {
      result.failedKeys.push(pair.key)
      continue
    }
    if (existingKeys.has(pair.key)) result.updated += 1
    else result.created += 1
  }

  return result
}
