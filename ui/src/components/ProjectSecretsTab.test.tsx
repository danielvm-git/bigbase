import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { ProjectSecretsTab } from './ProjectSecretsTab'
import type { SecretMetadata, SecretValue, SecretVersionMetadata } from '../types/secrets'

// vi.fn() references kept outside vi.mock so tests can override per-test.
const listSecretsMock = vi.fn()
const readSecretValueMock = vi.fn()
const listSecretVersionsMock = vi.fn()
const createSecretMock = vi.fn()
const updateSecretMock = vi.fn()
const deleteSecretMock = vi.fn()
const importEnvSecretsMock = vi.fn()

vi.mock('../lib/secretsData', () => ({
  MAX_SECRET_BATCH_KEYS: 1000,
  listSecrets: (...args: unknown[]) => listSecretsMock(...args),
  readSecretValue: (...args: unknown[]) => readSecretValueMock(...args),
  listSecretVersions: (...args: unknown[]) => listSecretVersionsMock(...args),
  createSecret: (...args: unknown[]) => createSecretMock(...args),
  updateSecret: (...args: unknown[]) => updateSecretMock(...args),
  deleteSecret: (...args: unknown[]) => deleteSecretMock(...args),
  importEnvSecrets: (...args: unknown[]) => importEnvSecretsMock(...args),
  parseEnvFile: (text: string) =>
    text
      .split('\n')
      .map(line => line.trim())
      .filter(Boolean)
      .map(line => {
        const eq = line.indexOf('=')
        return { key: line.slice(0, eq).trim(), value: line.slice(eq + 1).trim() }
      }),
}))

const meta = (key: string, overrides: Partial<SecretMetadata> = {}): SecretMetadata => ({
  id: `s-${key}`,
  project_id: 'p1',
  environment_id: 'e1',
  folder_id: 'f1',
  key,
  current_version: 2,
  // Per-key masked preview so assertions can tell rows apart.
  value_preview: `••••${key.slice(-4)}`,
  created_at: '2026-08-01T00:00:00Z',
  updated_at: '2026-08-02T00:00:00Z',
  ...overrides,
})

const valueFixture = (key: string, value: string): SecretValue => ({
  secret_id: `s-${key}`,
  key,
  version: 2,
  value,
  key_id: 'k1',
  algorithm: 'AES-256-GCM',
})

const versionFixture = (v: number): SecretVersionMetadata => ({
  id: `v-${v}`,
  version: v,
  key_id: 'k1',
  algorithm: 'AES-256-GCM',
  created_at: '2026-08-01T00:00:00Z',
})

async function seedRows(keys: string[]) {
  listSecretsMock.mockResolvedValue(keys.map(k => meta(k)))
  render(<ProjectSecretsTab projectId="p1" envId="e1" />)
  await screen.findByText(keys[0])
  return keys
}

describe('ProjectSecretsTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listSecretsMock.mockResolvedValue([])
    readSecretValueMock.mockResolvedValue({ ok: true, data: valueFixture('DB_URL', 'postgres://secret') })
    listSecretVersionsMock.mockResolvedValue([versionFixture(1), versionFixture(2)])
    createSecretMock.mockResolvedValue({ ok: true })
    updateSecretMock.mockResolvedValue({ ok: true })
    deleteSecretMock.mockResolvedValue({ ok: true })
    importEnvSecretsMock.mockResolvedValue({ created: 0, updated: 0, failedKeys: [], skipped: 0 })
    vi.stubGlobal('confirm', vi.fn(() => true))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders a masked secret table without plaintext values', async () => {
    await seedRows(['DB_URL', 'API_KEY'])

    // Keys are visible…
    expect(screen.getByText('DB_URL')).toBeInTheDocument()
    expect(screen.getByText('API_KEY')).toBeInTheDocument()
    // …and only masked previews are rendered, never a plaintext value.
    expect(screen.getByText('••••_URL')).toBeInTheDocument()
    expect(screen.getByText('••••_KEY')).toBeInTheDocument()
    expect(screen.queryByText('postgres://secret')).not.toBeInTheDocument()
    // The metadata fixture itself carries no plaintext field.
    const metadata = meta('DB_URL') as unknown as Record<string, unknown>
    expect('value' in metadata).toBe(false)
  })

  it('reveal shows the value only for the chosen secret and clears it on close', async () => {
    await seedRows(['DB_URL', 'API_KEY'])
    readSecretValueMock.mockResolvedValue({ ok: true, data: valueFixture('DB_URL', 'postgres://secret') })

    const dbRow = screen.getByText('DB_URL').closest('tr')!
    fireEvent.click(within(dbRow).getByRole('button', { name: 'Reveal' }))

    // The chosen secret's value appears; the other row stays masked.
    await screen.findByText('postgres://secret')
    expect(screen.getByText('••••_KEY')).toBeInTheDocument()

    // Closing the dialog clears the only state that held the plaintext.
    fireEvent.click(screen.getByRole('button', { name: 'Close dialog' }))
    await waitFor(() => {
      expect(screen.queryByText('postgres://secret')).not.toBeInTheDocument()
    })
    expect(readSecretValueMock).toHaveBeenCalledWith('p1', 'e1', 'DB_URL')
  })

  it('reveal failure renders a value-free 403 error and retains no value', async () => {
    await seedRows(['DB_URL'])
    readSecretValueMock.mockResolvedValue({ ok: false, status: 403, error: "You don't have permission to perform this action." })

    fireEvent.click(screen.getByRole('button', { name: 'Reveal' }))
    await screen.findByText(/don't have permission/i)

    // No plaintext and no submitted value anywhere in the dialog.
    expect(screen.queryByText('postgres://secret')).not.toBeInTheDocument()
    expect(screen.queryByText(/postgres/)).not.toBeInTheDocument()
  })

  it('edit form opens with an empty value input, never prefilled', async () => {
    await seedRows(['DB_URL'])
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    const valueInput = screen.getByLabelText('Value') as HTMLInputElement
    expect(valueInput).toBeInTheDocument()
    expect(valueInput.value).toBe('')
    expect(valueInput).toHaveAttribute('placeholder', expect.stringContaining('replacement value'))
  })

  it('creates a secret through the form with key and value', async () => {
    listSecretsMock.mockResolvedValue([])
    render(<ProjectSecretsTab projectId="p1" envId="e1" />)
    await screen.findByText(/No secrets in this folder yet/i)

    fireEvent.click(screen.getByRole('button', { name: 'Add Secret' }))
    fireEvent.change(screen.getByLabelText('Key'), { target: { value: 'NEW_KEY' } })
    fireEvent.change(screen.getByLabelText('Value'), { target: { value: 'brand-new-value' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add' }))

    await waitFor(() => {
      expect(createSecretMock).toHaveBeenCalledWith('p1', 'e1', { key: 'NEW_KEY', value: 'brand-new-value' })
    })
  })

  it('update sends only the replacement value, never a stored plaintext', async () => {
    await seedRows(['DB_URL'])
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
    fireEvent.change(screen.getByLabelText('Value'), { target: { value: 'replacement' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(updateSecretMock).toHaveBeenCalledWith('p1', 'e1', 'DB_URL', { value: 'replacement' })
    })
    // The old value never entered the payload or the UI.
    expect(JSON.stringify(updateSecretMock.mock.calls)).not.toContain('postgres://secret')
  })

  it('delete requires destructive confirmation and deletes the chosen key', async () => {
    await seedRows(['DB_URL'])
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    // Confirmation dialog must be acknowledged before anything is deleted.
    expect(deleteSecretMock).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog', { name: 'Delete secret' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Delete secret' }))

    await waitFor(() => {
      expect(deleteSecretMock).toHaveBeenCalledWith('p1', 'e1', 'DB_URL')
    })
  })

  it('version history renders metadata without any value', async () => {
    await seedRows(['DB_URL'])
    fireEvent.click(screen.getByRole('button', { name: 'Versions' }))

    const dialog = await screen.findByRole('dialog', { name: /Versions/ })
    expect(within(dialog).getByText('v1')).toBeInTheDocument()
    expect(within(dialog).getByText('v2')).toBeInTheDocument()
    expect(listSecretVersionsMock).toHaveBeenCalledWith('p1', 'e1', 'DB_URL')
    // Version rows carry algorithm/key metadata only — never plaintext.
    expect(within(dialog).queryByText('postgres://secret')).not.toBeInTheDocument()
    expect(within(dialog).queryByText(/postgres/)).not.toBeInTheDocument()
  })

  it('import reports partial failure by key name and never echoes submitted values', async () => {
    await seedRows(['EXISTING'])
    importEnvSecretsMock.mockResolvedValue({ created: 1, updated: 1, failedKeys: ['BAD-KEY'], skipped: 0 })

    const file = new File(['NEW_KEY=newval\nEXISTING=overwrite\nBAD-KEY=badval'], 'test.env', { type: 'text/plain' })
    fireEvent.change(screen.getByLabelText('Import .env file'), { target: { files: [file] } })

    await screen.findByText(/1 key failed to import: BAD-KEY/)
    expect(importEnvSecretsMock).toHaveBeenCalledWith('p1', 'e1', expect.stringContaining('NEW_KEY'), expect.any(Set))

    // Submitted values never appear in the failure report or the page.
    expect(screen.queryByText(/newval|overwrite|badval/)).not.toBeInTheDocument()
  })

  it('import asks before overwriting existing keys and proceeds when confirmed', async () => {
    await seedRows(['EXISTING'])
    const confirmSpy = vi.fn(() => true)
    vi.stubGlobal('confirm', confirmSpy)
    importEnvSecretsMock.mockResolvedValue({ created: 0, updated: 1, failedKeys: [], skipped: 0 })

    const file = new File(['EXISTING=overwrite'], 'test.env', { type: 'text/plain' })
    fireEvent.change(screen.getByLabelText('Import .env file'), { target: { files: [file] } })

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalledWith(expect.stringContaining('EXISTING'))
    })
    await waitFor(() => {
      expect(importEnvSecretsMock).toHaveBeenCalled()
    })
  })
})
