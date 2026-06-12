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

    expect(result.current.lines).toEqual([])
  })

  it('polls for logs when status is building', async () => {
    vi.useFakeTimers()
    const mockLogsBuilding = {
      deployment_id: 'dep-123',
      status: 'building',
      lines: ['Step 1: cloning'],
      log_available: true
    }

    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => mockLogsBuilding
    } as Response)

    const { result } = renderHook(() => useBuildLogs('dep-123'))

    // Initial fetch happens immediately
    await vi.waitUntil(() => result.current.status === 'building')
    expect(fetchSpy).toHaveBeenCalledTimes(1)

    // Advance time by 2s -> Second call
    await vi.advanceTimersByTimeAsync(2000)
    expect(fetchSpy).toHaveBeenCalledTimes(2)

    // Advance time by another 2s -> Third call
    await vi.advanceTimersByTimeAsync(2000)
    expect(fetchSpy).toHaveBeenCalledTimes(3)

    vi.useRealTimers()
  })

  it('stops polling when status is running', async () => {
    vi.useFakeTimers()
    const mockLogsBuilding = {
      deployment_id: 'dep-123',
      status: 'building',
      lines: ['building...'],
      log_available: true
    }
    const mockLogsRunning = {
      deployment_id: 'dep-123',
      status: 'running',
      lines: ['done!'],
      log_available: true
    }

    const fetchSpy = vi.spyOn(global, 'fetch')
      .mockResolvedValueOnce({ ok: true, json: async () => mockLogsBuilding } as Response)
      .mockResolvedValueOnce({ ok: true, json: async () => mockLogsRunning } as Response)
      .mockResolvedValue({ ok: true, json: async () => mockLogsRunning } as Response)

    const { result } = renderHook(() => useBuildLogs('dep-123'))

    // 1. Initial call
    await vi.waitUntil(() => result.current.status === 'building')
    expect(fetchSpy).toHaveBeenCalledTimes(1)

    // 2. Advance 2s -> Second call (returns running)
    await vi.advanceTimersByTimeAsync(2000)
    await vi.waitUntil(() => result.current.status === 'running')
    expect(fetchSpy).toHaveBeenCalledTimes(2)

    // 3. Advance 2s -> Should NOT call again
    await vi.advanceTimersByTimeAsync(2000)
    expect(fetchSpy).toHaveBeenCalledTimes(2)

    vi.useRealTimers()
  })
})
