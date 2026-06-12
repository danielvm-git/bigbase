import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useRequestLogs } from './useRequestLogs'

describe('useRequestLogs', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches request logs for a site', async () => {
    const mockData = {
      data: [
        { id: 'l1', method: 'GET', path: '/test', status: 200, duration_ms: 5, created_at: '2026-06-12T15:00:00Z' }
      ],
      total: 1,
      site_id: 's1'
    }

    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => mockData
    } as Response)

    const { result } = renderHook(() => useRequestLogs('s1'))

    expect(result.current.loading).toBe(true)

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.logs).toEqual(mockData.data)
    expect(result.current.total).toBe(1)
    expect(fetchSpy).toHaveBeenCalledWith(expect.stringContaining('/api/sites/s1/logs'))
  })

  it('handles filters', async () => {
    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: [], total: 0 })
    } as Response)

    const { result } = renderHook(() => useRequestLogs('s1'))
    
    // Wait for initial fetch
    await waitFor(() => expect(result.current.loading).toBe(false))
    
    // Update filters
    await act(async () => {
      result.current.setPathPrefix('/api')
    })
    await act(async () => {
      result.current.setStatusClass('4xx')
    })

    // Wait for filtered fetch
    await waitFor(() => expect(result.current.loading).toBe(false))

    const lastCall = fetchSpy.mock.calls[fetchSpy.mock.calls.length - 1][0] as string
    expect(lastCall).toContain('path=%2Fapi')
    expect(lastCall).toContain('status_class=4xx')
  })
})
