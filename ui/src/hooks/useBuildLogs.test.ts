import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { useBuildLogs } from './useBuildLogs'

describe('useBuildLogs', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches logs for a deployment', async () => {
    const mockLogs = {
      deployment_id: 'dep-123',
      status: 'building',
      lines: ['Step 1: cloning', 'Step 2: installing'],
      log_available: true
    }

    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => mockLogs
    } as Response)

    const { result } = renderHook(() => useBuildLogs('dep-123'))

    expect(result.current.loading).toBe(true)

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.lines).toEqual(['Step 1: cloning', 'Step 2: installing'])
    expect(result.current.error).toBeNull()
    expect(fetchSpy).toHaveBeenCalledWith('/api/deployments/dep-123/logs')
  })

  it('handles fetch errors', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({ error: 'internal server error' })
    } as Response)

    const { result } = renderHook(() => useBuildLogs('dep-123'))

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.error).toBe('internal server error')
    expect(result.current.lines).toEqual([])
  })
})
