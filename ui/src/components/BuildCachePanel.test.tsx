import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

const getCacheStats = vi.fn()
const clearAllCache = vi.fn()
const pruneCache = vi.fn()
const setCacheMaxSize = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getCacheStats: () => getCacheStats(),
  clearAllCache: () => clearAllCache(),
  pruneCache: (d: number) => pruneCache(d),
  setCacheMaxSize: (b: number) => setCacheMaxSize(b),
}))

import { BuildCachePanel } from './BuildCachePanel'

const stats = { total_entries: 14, total_size_bytes: 1288490188, max_size_bytes: 2147483648 }

describe('BuildCachePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getCacheStats.mockResolvedValue(stats)
  })

  it('renders usage, max size, and entry count', async () => {
    render(<BuildCachePanel />)
    await waitFor(() => {
      expect(screen.getByText(/1\.2 GB \/ 2 GB/)).toBeInTheDocument()
    })
    expect(screen.getByText('14')).toBeInTheDocument()
  })

  it('prunes entries older than 7 days', async () => {
    pruneCache.mockResolvedValue({ ok: true, pruned: 3 })
    render(<BuildCachePanel />)
    await waitFor(() => screen.getByRole('button', { name: /prune/i }))
    fireEvent.click(screen.getByRole('button', { name: /prune/i }))
    await waitFor(() => expect(pruneCache).toHaveBeenCalledWith(7))
  })

  it('clears all caches after confirmation', async () => {
    clearAllCache.mockResolvedValue({ ok: true })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<BuildCachePanel />)
    await waitFor(() => screen.getByRole('button', { name: /clear all/i }))
    fireEvent.click(screen.getByRole('button', { name: /clear all/i }))
    await waitFor(() => expect(clearAllCache).toHaveBeenCalled())
  })

  it('does not clear when confirmation is declined', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<BuildCachePanel />)
    await waitFor(() => screen.getByRole('button', { name: /clear all/i }))
    fireEvent.click(screen.getByRole('button', { name: /clear all/i }))
    expect(clearAllCache).not.toHaveBeenCalled()
  })

  it('saves max size converting GB to bytes', async () => {
    setCacheMaxSize.mockResolvedValue({ ok: true })
    render(<BuildCachePanel />)
    const input = await screen.findByLabelText('Max cache size in GB')
    fireEvent.change(input, { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(setCacheMaxSize).toHaveBeenCalledWith(5 * 1024 * 1024 * 1024))
  })

  it('rejects a non-positive max size without calling the API', async () => {
    render(<BuildCachePanel />)
    const input = await screen.findByLabelText('Max cache size in GB')
    fireEvent.change(input, { target: { value: '0' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(setCacheMaxSize).not.toHaveBeenCalled()
    expect(screen.getByText(/positive number of GB/)).toBeInTheDocument()
  })

  it('surfaces an error when an operation fails', async () => {
    clearAllCache.mockResolvedValue({ ok: false, error: 'disk error' })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<BuildCachePanel />)
    await waitFor(() => screen.getByRole('button', { name: /clear all/i }))
    fireEvent.click(screen.getByRole('button', { name: /clear all/i }))
    await waitFor(() => expect(screen.getByText('disk error')).toBeInTheDocument())
  })
})
