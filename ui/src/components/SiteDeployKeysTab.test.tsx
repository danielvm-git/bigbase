import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

const getDeployKeys = vi.fn()
const createDeployKey = vi.fn()
const revokeDeployKey = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getDeployKeys: (id: string) => getDeployKeys(id),
  createDeployKey: (id: string, name: string) => createDeployKey(id, name),
  revokeDeployKey: (id: string, keyId: string) => revokeDeployKey(id, keyId),
}))

import { SiteDeployKeysTab } from './SiteDeployKeysTab'

describe('SiteDeployKeysTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── 1. Renders deploy keys list ──

  it('renders deploy keys in a table with name, prefix, and dates', async () => {
    const keys = [
      {
        key_id: 'key-1',
        name: 'ci-key',
        prefix: 'bb_dep_abc',
        last_used_at: '2026-06-15T10:00:00Z',
        created_at: '2026-01-10T08:00:00Z',
      },
      {
        key_id: 'key-2',
        name: 'dev-key',
        prefix: 'bb_dep_def',
        last_used_at: null,
        created_at: '2026-03-20T12:00:00Z',
      },
    ]
    getDeployKeys.mockResolvedValue(keys)

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => {
      expect(screen.getByText('ci-key')).toBeInTheDocument()
    })
    expect(screen.getByText('dev-key')).toBeInTheDocument()
    expect(screen.getByText('bb_dep_abc…')).toBeInTheDocument()
    expect(screen.getByText('bb_dep_def…')).toBeInTheDocument()
    // Key with no last_used_at shows "Never"
    expect(screen.getByText('Never')).toBeInTheDocument()
    // Verify a Revoke button exists per row
    expect(screen.getAllByRole('button', { name: 'Revoke' }).length).toBe(2)
  })

  // ── 2. Empty state ──

  it('shows empty state when no deploy keys exist', async () => {
    getDeployKeys.mockResolvedValue([])

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => {
      expect(screen.getByText('No deploy keys')).toBeInTheDocument()
    })
    // Header button + empty-state action button
    const buttons = screen.getAllByRole('button', { name: 'Generate Key' })
    expect(buttons.length).toBe(2)
  })

  // ── 3. Loading state ──

  it('shows loading message while fetching deploy keys', () => {
    getDeployKeys.mockReturnValue(new Promise(() => {}))

    render(<SiteDeployKeysTab siteId="s1" />)

    expect(screen.getByText('Loading deploy keys…')).toBeInTheDocument()
  })

  // ── 4. Error state ──

  it('shows error message when loading fails', async () => {
    getDeployKeys.mockRejectedValue(new Error('Network error'))

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => {
      expect(screen.getByText('Failed to load deploy keys')).toBeInTheDocument()
    })
  })

  // ── 5. Generate modal opens on button click ──

  it('opens generate modal when Generate Key is clicked', async () => {
    const user = userEvent.setup()
    getDeployKeys.mockResolvedValue([])

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => {
      expect(screen.getByText('No deploy keys')).toBeInTheDocument()
    })

    await user.click(screen.getAllByRole('button', { name: 'Generate Key' })[0])

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
    expect(screen.getByPlaceholderText('e.g. ci-bot')).toBeInTheDocument()
  })

  // ── 6. Generate modal creates key and shows raw token ──

  it('generates a key and calls the create API', async () => {
    const user = userEvent.setup()
    createDeployKey.mockResolvedValue({
      ok: true,
      data: {
        key_id: 'new-key',
        name: 'my-ci-key',
        key: 'bb_dep_secret123',
        prefix: 'bb_dep_sec',
        created_at: '2026-07-11T00:00:00Z',
      },
    })
    // Second resolve for the refresh triggered by handleGenerated
    getDeployKeys.mockResolvedValue([])

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => screen.getByText('No deploy keys'))
    await user.click(screen.getAllByRole('button', { name: 'Generate Key' })[0])
    await waitFor(() => screen.getByRole('dialog'))

    // Fill in the key name
    await user.type(screen.getByPlaceholderText('e.g. ci-bot'), 'my-ci-key')

    // Click "Generate Key" inside the dialog
    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Generate Key' }))

    await waitFor(() => {
      expect(createDeployKey).toHaveBeenCalledWith('s1', 'my-ci-key')
    })

    // handleGenerated closes the modal and refreshes
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  // ── 7. Token cleared on modal close ──

  it('shows the generate form again when reopened after generating a key', async () => {
    const user = userEvent.setup()
    createDeployKey.mockResolvedValue({
      ok: true,
      data: {
        key_id: 'new-key',
        name: 'test',
        key: 'bb_dep_secret',
        prefix: 'bb_dep_sec',
        created_at: '2026-07-11T00:00:00Z',
      },
    })
    getDeployKeys.mockResolvedValue([])

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => screen.getByText('No deploy keys'))
    await user.click(screen.getAllByRole('button', { name: 'Generate Key' })[0])
    await waitFor(() => screen.getByRole('dialog'))

    await user.type(screen.getByPlaceholderText('e.g. ci-bot'), 'test')
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Generate Key' }))

    // handleGenerated closes the modal immediately after generation
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    // Reopen modal
    await user.click(screen.getAllByRole('button', { name: 'Generate Key' })[0])
    await waitFor(() => screen.getByRole('dialog'))

    // Form input should be shown (not the previous result)
    expect(screen.getByPlaceholderText('e.g. ci-bot')).toBeInTheDocument()
  })

  // ── 8. Revoke confirmation dialog ──

  it('shows revoke confirmation dialog when Revoke is clicked on a key row', async () => {
    const user = userEvent.setup()
    const keys = [
      {
        key_id: 'key-1',
        name: 'ci-key',
        prefix: 'bb_dep_abc',
        last_used_at: '2026-06-15T10:00:00Z',
        created_at: '2026-01-10T08:00:00Z',
      },
    ]
    getDeployKeys.mockResolvedValue(keys)

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => screen.getByText('ci-key'))

    await user.click(screen.getByRole('button', { name: 'Revoke' }))

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
    const dlg = screen.getByRole('dialog')
    expect(within(dlg).getByText(/Are you sure you want to revoke/)).toBeInTheDocument()
    expect(within(dlg).getByText(/lose access immediately/)).toBeInTheDocument()
    // The key name appears in the confirmation message
    expect(within(dlg).getByText('ci-key')).toBeInTheDocument()
  })

  // ── 9. Revoke removes key from list ──

  it('revokes a key and refreshes the list', async () => {
    const user = userEvent.setup()
    const keys = [
      {
        key_id: 'key-1',
        name: 'ci-key',
        prefix: 'bb_dep_abc',
        last_used_at: '2026-06-15T10:00:00Z',
        created_at: '2026-01-10T08:00:00Z',
      },
    ]
    getDeployKeys.mockResolvedValueOnce(keys).mockResolvedValueOnce([])
    revokeDeployKey.mockResolvedValue({ ok: true })

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => screen.getByText('ci-key'))

    await user.click(screen.getByRole('button', { name: 'Revoke' }))
    await waitFor(() => screen.getByRole('dialog'))

    // Confirm revoke via the dialog's Revoke button
    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Revoke' }))

    await waitFor(() => {
      expect(revokeDeployKey).toHaveBeenCalledWith('s1', 'key-1')
    })

    // List refreshes — empty state appears
    await waitFor(() => {
      expect(screen.getByText('No deploy keys')).toBeInTheDocument()
    })
  })

  // ── 10. Rate-limit error ──

  it('shows rate-limited message when createDeployKey returns a rate error', async () => {
    const user = userEvent.setup()
    createDeployKey.mockResolvedValue({ ok: false, error: 'rate limit exceeded' })
    getDeployKeys.mockResolvedValue([])

    render(<SiteDeployKeysTab siteId="s1" />)

    await waitFor(() => screen.getByText('No deploy keys'))
    await user.click(screen.getAllByRole('button', { name: 'Generate Key' })[0])
    await waitFor(() => screen.getByRole('dialog'))

    await user.type(screen.getByPlaceholderText('e.g. ci-bot'), 'my-key')
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Generate Key' }))

    await waitFor(() => {
      expect(
        screen.getByText('Rate limited. Please try again later.'),
      ).toBeInTheDocument()
    })
  })
})
