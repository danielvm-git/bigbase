import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

const getSiteCache = vi.fn()
const clearSiteCache = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getSiteCache: (id: string) => getSiteCache(id),
  clearSiteCache: (id: string) => clearSiteCache(id),
}))

import { SiteCacheTab } from './SiteCacheTab'

const populatedStatus = {
  entries: [
    { key: 'a1b2c3d4e5f6aaaa', site_id: 's1', repo_id: 'r1', branch: 'main', size: 134217728, hit_count: 7, created_at: '2026-06-01T10:00:00Z' },
  ],
  total_size_bytes: 134217728,
  total_hits: 7,
}
const populated = { status: populatedStatus, ok: true }
const emptyOk = { status: { entries: [], total_size_bytes: 0, total_hits: 0 }, ok: true }

describe('SiteCacheTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows an empty state when no cache entries exist', async () => {
    getSiteCache.mockResolvedValue(emptyOk)
    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => {
      expect(screen.getByText(/No cached dependencies yet/)).toBeInTheDocument()
    })
    // No Clear button when empty.
    expect(screen.queryByRole('button', { name: /clear cache/i })).not.toBeInTheDocument()
  })

  it('renders cache entries with formatted size and aggregate hits', async () => {
    getSiteCache.mockResolvedValue(populated)
    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => {
      // "128 MB" appears in both the aggregate and the entry row.
      expect(screen.getAllByText('128 MB').length).toBeGreaterThanOrEqual(2)
    })
    // Truncated key (first 12 chars).
    expect(screen.getByText('a1b2c3d4e5f6')).toBeInTheDocument()
    // Aggregate hits shown.
    expect(screen.getAllByText('7').length).toBeGreaterThanOrEqual(1)
  })

  it('labels the table with a caption and scopes column headers', async () => {
    getSiteCache.mockResolvedValue(populated)
    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => {
      expect(screen.getByRole('table', { name: 'Build cache' })).toBeInTheDocument()
    })
    for (const name of ['Key', 'Branch', 'Size', 'Hits', 'Created']) {
      expect(screen.getByRole('columnheader', { name })).toHaveAttribute('scope', 'col')
    }
  })

  it('clears the cache after confirmation and reloads', async () => {
    getSiteCache.mockResolvedValueOnce(populated).mockResolvedValueOnce(emptyOk)
    clearSiteCache.mockResolvedValue({ ok: true })
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => screen.getByRole('button', { name: /clear cache/i }))
    fireEvent.click(screen.getByRole('button', { name: /clear cache/i }))

    await waitFor(() => {
      expect(clearSiteCache).toHaveBeenCalledWith('s1')
      expect(screen.getByText(/No cached dependencies yet/)).toBeInTheDocument()
    })
  })

  it('does not clear when confirmation is declined', async () => {
    getSiteCache.mockResolvedValue(populated)
    vi.spyOn(window, 'confirm').mockReturnValue(false)

    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => screen.getByRole('button', { name: /clear cache/i }))
    fireEvent.click(screen.getByRole('button', { name: /clear cache/i }))

    expect(clearSiteCache).not.toHaveBeenCalled()
  })

  it('surfaces an error when the initial load fails (not a false empty state)', async () => {
    getSiteCache.mockResolvedValue({ status: { entries: [], total_size_bytes: 0, total_hits: 0 }, ok: false })
    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => {
      expect(screen.getByText(/Could not load cache status/)).toBeInTheDocument()
    })
  })

  it('surfaces an error when clearing fails', async () => {
    getSiteCache.mockResolvedValue(populated)
    clearSiteCache.mockResolvedValue({ ok: false, error: 'boom' })
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<SiteCacheTab siteId="s1" />)
    await waitFor(() => screen.getByRole('button', { name: /clear cache/i }))
    fireEvent.click(screen.getByRole('button', { name: /clear cache/i }))

    await waitFor(() => {
      expect(screen.getByText('boom')).toBeInTheDocument()
    })
  })
})
