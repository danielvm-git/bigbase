import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useBuildLogs } from './useBuildLogs'

// Minimal WebSocket mock for testing
class MockWebSocket {
  static instances: MockWebSocket[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: ((e: CloseEvent) => void) | null = null
  onerror: (() => void) | null = null
  readyState: number = 0 // CONNECTING
  close = vi.fn()
  send = vi.fn()

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }
}

const OPEN = 1
const CLOSED = 3

function createMessageEvent(data: string): MessageEvent {
  return new MessageEvent('message', { data })
}
function createCloseEvent(code = 1000, reason = ''): CloseEvent {
  return new CloseEvent('close', { code, reason })
}

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
    expect(fetchSpy).toHaveBeenCalledWith('/api/deploy/dep-123/logs')
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

  it('streams logs via WebSocket and exposes isStreaming', async () => {
    MockWebSocket.instances = []
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket

    // Mock initial fetch (polling fallback data)
    const mockLogs = {
      deployment_id: 'dep-456',
      status: 'building',
      lines: [],
      log_available: true
    }

    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => mockLogs
    } as Response)

    const { result } = renderHook(() => useBuildLogs('dep-456'))

    // Should have created a WebSocket connection
    await vi.waitFor(() => expect(MockWebSocket.instances.length).toBe(1))
    const ws = MockWebSocket.instances[0]
    expect(ws.url).toContain('/api/deploy/dep-456/logs/stream')

    // Simulate WebSocket open
    act(() => { ws.readyState = OPEN; ws.onopen?.() })

    // Should be streaming
    await vi.waitFor(() => expect(result.current.isStreaming).toBe(true))

    // Simulate receiving log lines
    act(() => { ws.onmessage?.(createMessageEvent('npm install')) })
    act(() => { ws.onmessage?.(createMessageEvent('npm run build')) })

    await vi.waitFor(() => expect(result.current.lines).toContain('npm install'))
    expect(result.current.lines).toContain('npm run build')

    // Simulate WebSocket clean close
    act(() => { ws.readyState = CLOSED; ws.onclose?.(createCloseEvent(1000, 'done')) })

    await vi.waitFor(() => expect(result.current.isStreaming).toBe(false))

    delete (global as any).WebSocket
  })

  it('falls back to polling when WebSocket errors', async () => {
    vi.useFakeTimers()
    MockWebSocket.instances = []
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket

    const mockLogsBuilding = {
      deployment_id: 'dep-789',
      status: 'building',
      lines: ['Step 1: cloning'],
      log_available: true
    }

    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => mockLogsBuilding
    } as Response)

    const { result } = renderHook(() => useBuildLogs('dep-789'))

    // Flush initial fetch (which runs in the useEffect)
    await vi.runAllTimersAsync()

    // Wait for WebSocket to be created
    await vi.waitFor(() => expect(MockWebSocket.instances.length).toBe(1))
    const ws = MockWebSocket.instances[0]

    // Simulate WebSocket error — should trigger fallback to polling
    act(() => { ws.onerror?.() })

    // After error, isStreaming should be false
    await vi.waitFor(() => expect(result.current.isStreaming).toBe(false))

    // Polling should kick in (status is 'building' from initial fetch)
    // Advance 2s for first polling interval
    await vi.advanceTimersByTimeAsync(2000)
    expect(fetchSpy).toHaveBeenCalledTimes(2) // initial + first poll

    vi.useRealTimers()
    delete (global as any).WebSocket
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
