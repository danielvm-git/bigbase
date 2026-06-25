import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

const getSiteCache = vi.fn()
const clearSiteCache = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getSiteCache: (id: string) => getSiteCache(id),
  clearSiteCache: (id: string) => clearSiteCache(id),
}))

// eslint-disable-next-line import/first
import { SiteCacheTab } from './SiteCacheTab'

const populated = {
  entries: [
    { key: 'a1b2c3d4e5f6aaaa', site_id: 's1', repo_id: 'r1', branch: 'main', size: 134217728, hit_count: 7, created_at: '2026-06-01T10:00:00Z' },
  ],
  total_size_bytes: 134217728,
  total_hits: 7,
}

describe('SiteCacheTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows an empty state when no cache entries exist', async () => {
    getSiteCache.mockResolvedValue({ entries: [], total_size_bytes: 0, total_hits: 0 })
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

  it('clears the cache after confirmation and reloads', async () => {
    getSiteCache.mockResolvedValueOnce(populated).mockResolvedValueOnce({ entries: [], total_size_bytes: 0, total_hits: 0 })
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
