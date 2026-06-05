import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { fetchDeployLogs, useDeployLogs } from './useDeployLogs'

describe('fetchDeployLogs', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('parses lines from API response', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        deployment_id: 'dep-1',
        status: 'building',
        lines: ['→ Cloning repository', '→ Clone complete'],
        log_available: true,
      }),
    } as Response)

    const data = await fetchDeployLogs('dep-1')
    expect(data?.lines).toEqual(['→ Cloning repository', '→ Clone complete'])
    expect(data?.status).toBe('building')
  })
})

describe('useDeployLogs', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sets isStreaming while enabled and not terminal', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        deployment_id: 'dep-1',
        status: 'building',
        lines: ['→ Building'],
      }),
    } as Response)

    const { result } = renderHook(() =>
      useDeployLogs({ deploymentId: 'dep-1', enabled: true, pollIntervalMs: 100 }),
    )

    await waitFor(() => {
      expect(result.current.lines).toContain('→ Building')
    })
    expect(result.current.isStreaming).toBe(true)
  })

  it('stops streaming when deployStatus is running', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        deployment_id: 'dep-1',
        status: 'building',
        lines: ['→ Building'],
      }),
    } as Response)

    const { result } = renderHook(() =>
      useDeployLogs({
        deploymentId: 'dep-1',
        enabled: true,
        deployStatus: 'running',
        pollIntervalMs: 100,
      }),
    )

    await waitFor(() => {
      expect(result.current.lines).toContain('→ Building')
    })
    expect(result.current.isStreaming).toBe(false)
  })

  it('sets fetchError when logs request fails', async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false } as Response)

    const { result } = renderHook(() =>
      useDeployLogs({ deploymentId: 'dep-1', enabled: true, pollIntervalMs: 100 }),
    )

    await waitFor(() => {
      expect(result.current.fetchError).toBe('Could not load build logs')
    })
  })

  it('uses preview lines when deployment id is empty', () => {
    const { result } = renderHook(() =>
      useDeployLogs({
        deploymentId: '',
        enabled: false,
        previewLines: ['preview line'],
      }),
    )
    expect(result.current.lines).toEqual(['preview line'])
    expect(result.current.isStreaming).toBe(false)
  })
})
