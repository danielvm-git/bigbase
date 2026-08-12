import { describe, expect, it, vi, afterEach } from 'vitest'
import {
  MAX_SECRET_BATCH_KEYS,
  createSecret,
  deleteSecret,
  importEnvSecrets,
  listProjects,
  listSecrets,
  listSecretVersions,
  parseEnvFile,
  readSecretValue,
  updateSecret,
} from './secretsData'
import type { SecretMetadata, SecretValue } from '../types/secrets'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function stubFetch(impl: (url: string, init?: RequestInit) => Promise<Response>) {
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : String(input)
    return impl(url, init)
  }))
}

// Metadata fixtures must never carry a plaintext `value` property — the
// separate SecretValue type is the only plaintext-bearing shape.
const metaFixture = (key: string): SecretMetadata => ({
  id: `s-${key}`,
  project_id: 'p1',
  environment_id: 'e1',
  folder_id: 'f1',
  key,
  current_version: 1,
  value_preview: '••••abcd',
  created_at: '2026-08-01T00:00:00Z',
  updated_at: '2026-08-01T00:00:00Z',
})

describe('listProjects / listSecrets / listSecretVersions (metadata only)', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('listProjects returns projects from the API', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ data: [{ id: 'p1', org_id: 1, name: 'Demo', created_at: '', updated_at: '' }] })))
    const projects = await listProjects()
    expect(projects).toHaveLength(1)
    expect(projects[0].name).toBe('Demo')
  })

  it('listProjects returns [] on API failure', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ error: 'boom' }, 500)))
    expect(await listProjects()).toEqual([])
  })

  it('listSecrets returns metadata without a value property', async () => {
    const fixture = metaFixture('DATABASE_URL')
    expect('value' in fixture).toBe(false)
    stubFetch(() => Promise.resolve(jsonResponse({ data: [fixture] })))
    const secrets = await listSecrets('p1', 'e1')
    expect(secrets).toHaveLength(1)
    expect(secrets[0].key).toBe('DATABASE_URL')
    expect(secrets[0].value_preview).toBe('••••abcd')
    expect('value' in secrets[0]).toBe(false)
  })

  it('listSecretVersions returns metadata-only versions', async () => {
    stubFetch(() =>
      Promise.resolve(jsonResponse({ data: [{ id: 'v1', version: 2, key_id: 'k1', algorithm: 'AES-256-GCM', created_at: '' }] })),
    )
    const versions = await listSecretVersions('p1', 'e1', 'KEY')
    expect(versions).toHaveLength(1)
    expect(versions[0].version).toBe(2)
    expect('value' in versions[0]).toBe(false)
  })
})

describe('createSecret / updateSecret / deleteSecret', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('createSecret POSTs key and value and returns ok', async () => {
    let sent: unknown = null
    stubFetch(async (url, init) => {
      sent = { url, method: init?.method, body: init?.body }
      return jsonResponse({ data: metaFixture('API_KEY') }, 201)
    })
    const res = await createSecret('p1', 'e1', { key: 'API_KEY', value: 'sekrit' })
    expect(res.ok).toBe(true)
    expect(sent).toMatchObject({ url: '/api/projects/p1/environments/e1/secrets', method: 'POST' })
    expect(JSON.parse(String((sent as { body: string }).body))).toEqual({ key: 'API_KEY', value: 'sekrit' })
  })

  it('updateSecret PUTs the replacement value only', async () => {
    let sent: unknown = null
    stubFetch(async (url, init) => {
      sent = { url, method: init?.method, body: init?.body }
      return jsonResponse({ data: metaFixture('KEY') })
    })
    const res = await updateSecret('p1', 'e1', 'KEY', { value: 'new' })
    expect(res.ok).toBe(true)
    expect(sent).toMatchObject({ url: '/api/projects/p1/environments/e1/secrets/KEY', method: 'PUT' })
    expect(JSON.parse(String((sent as { body: string }).body))).toEqual({ value: 'new' })
  })

  it('createSecret maps 403 to a value-free permission message', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ error: 'forbidden' }, 403)))
    const res = await createSecret('p1', 'e1', { key: 'KEY', value: 'sekrit' })
    expect(res.ok).toBe(false)
    expect(res.error).toContain("don't have permission")
    expect(res.error).not.toContain('sekrit')
  })

  it('deleteSecret returns ok on 204', async () => {
    stubFetch(() => Promise.resolve(new Response(null, { status: 204 })))
    expect(await deleteSecret('p1', 'e1', 'KEY')).toEqual({ ok: true })
  })

  it('deleteSecret maps 401 to a value-free message', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ error: 'authorization required' }, 401)))
    const res = await deleteSecret('p1', 'e1', 'KEY')
    expect(res.ok).toBe(false)
    expect(res.error).toContain('signed in')
  })
})

describe('readSecretValue (explicit /value route)', () => {
  afterEach(() => vi.unstubAllGlobals())

  const valueFixture = (value: string): SecretValue => ({
    secret_id: 's1',
    key: 'API_KEY',
    version: 3,
    value,
    key_id: 'k1',
    algorithm: 'AES-256-GCM',
  })

  it('returns the value response on an authorized read', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ data: valueFixture('sup3r-secret') })))
    const res = await readSecretValue('p1', 'e1', 'API_KEY')
    expect(res.ok).toBe(true)
    expect(res.data?.value).toBe('sup3r-secret')
    expect(res.data?.key).toBe('API_KEY')
    expect(res.data?.version).toBe(3)
  })

  it('renders a value-free 403 error and never echoes the submitted value', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ error: 'forbidden' }, 403)))
    const res = await readSecretValue('p1', 'e1', 'API_KEY')
    expect(res.ok).toBe(false)
    expect(res.status).toBe(403)
    expect(res.error).toContain("don't have permission")
    expect(res.error).not.toContain('sup3r-secret')
    expect(res.data).toBeUndefined()
  })

  it('renders a value-free 401 error', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ error: 'authorization required' }, 401)))
    const res = await readSecretValue('p1', 'e1', 'API_KEY')
    expect(res.ok).toBe(false)
    expect(res.status).toBe(401)
    expect(res.error).toContain('signed in')
  })

  it('returns a network error without retaining a value', async () => {
    stubFetch(() => Promise.reject(new Error('down')))
    const res = await readSecretValue('p1', 'e1', 'API_KEY')
    expect(res.ok).toBe(false)
    expect(res.status).toBe(0)
    expect(res.error).toBe('Network error')
    expect(res.data).toBeUndefined()
  })
})

describe('parseEnvFile', () => {
  it('parses KEY=value pairs, skipping comments and blanks', () => {
    const pairs = parseEnvFile('# comment\n\nEMPTY=\nA=1\nB="quoted value"\n')
    expect(pairs).toEqual([
      { key: 'EMPTY', value: '' },
      { key: 'A', value: '1' },
      { key: 'B', value: 'quoted value' },
    ])
  })
})

describe('importEnvSecrets (safe .env import)', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('saves valid keys, updates existing keys, and reports invalid keys by name only', async () => {
    const calls: Array<{ url: string; method?: string; body?: unknown }> = []
    stubFetch(async (url, init) => {
      calls.push({ url, method: init?.method, body: init?.body })
      return jsonResponse({ data: metaFixture('K') })
    })

    const text = [
      'NEW_KEY=brand-new',
      'EXISTING_KEY=updated-value',
      'bad-key=lowercase-invalid',
      '123BAD=starts-with-digit',
    ].join('\n')

    const result = await importEnvSecrets('p1', 'e1', text, new Set(['EXISTING_KEY']))

    expect(result.created).toBe(1)
    expect(result.updated).toBe(1)
    expect(result.failedKeys.sort()).toEqual(['123BAD', 'bad-key'])
    expect(result.skipped).toBe(0)

    // Values are never echoed in the failure report.
    expect(result.failedKeys.join(' ')).not.toMatch(/brand-new|updated-value|lowercase|digit/)

    // Writes are per-key; only valid keys reached the API.
    expect(calls).toHaveLength(2)
    expect(calls[0].method).toBe('POST')
    expect(calls[1].method).toBe('PUT')
  })

  it('reports an over-batch key as skipped without writing it', async () => {
    const writes: string[] = []
    stubFetch(async (url, init) => {
      writes.push(`${init?.method} ${url}`)
      return jsonResponse({ data: metaFixture('K') })
    })

    const keys = Array.from({ length: MAX_SECRET_BATCH_KEYS + 3 }, (_, i) => `KEY_${i}`)
    const text = keys.map((k, i) => `${k}=v${i}`).join('\n')

    const result = await importEnvSecrets('p1', 'e1', text, new Set())
    expect(result.created).toBe(MAX_SECRET_BATCH_KEYS)
    expect(result.skipped).toBe(3)
    expect(result.failedKeys).toEqual([])
    expect(writes).toHaveLength(MAX_SECRET_BATCH_KEYS)
    expect(writes.some(w => w.includes('KEY_1000'))).toBe(false)
  })

  it('reports API failures by key name without the server error or value', async () => {
    stubFetch(() => Promise.resolve(jsonResponse({ error: 'secret already exists' }, 409)))
    const result = await importEnvSecrets('p1', 'e1', 'A=1\nB=2\n', new Set())
    expect(result.failedKeys.sort()).toEqual(['A', 'B'])
    expect(result.created).toBe(0)
    // The value characters never appear anywhere in the report.
    expect(result.failedKeys.join(' ')).not.toMatch(/[12]/)
  })
})
