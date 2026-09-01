import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'

// Minimal EventSource stub (jsdom has none). Captures instances so tests can
// drive open/message events.
class FakeEventSource {
  static instances: FakeEventSource[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent<string>) => void) | null = null
  onerror: (() => void) | null = null
  readyState = 0
  constructor(url: string) { this.url = url; FakeEventSource.instances.push(this) }
  close() { this.readyState = 2 }
  emitOpen() { this.readyState = 1; this.onopen?.() }
  emitMessage(data: unknown) {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(data) }))
  }
}

// Keep metrics null so the SSE EventSource effect never runs (it early-returns
// on `if (!metrics)`), letting us focus on the logs pagination flow.
vi.mock('../lib/metrics', () => ({
  fetchMonitoringMetricsWarmed: vi.fn().mockResolvedValue({ system: null }),
}))

import MonitoringPage from './MonitoringPage'

type Row = { id: string; level: string; message: string; created_at: string }

function logPage(cursor: string | null) {
  if (!cursor) {
    return {
      data: Array.from({ length: 100 }, (_, i): Row => ({
        id: `${1000 - i}`, level: 'info', message: `m${i}`, created_at: '2026-01-01T00:00:00Z',
      })),
      next_cursor: 'c100',
      has_more: true,
    }
  }
  return {
    data: Array.from({ length: 50 }, (_, i): Row => ({
      id: `${900 - i}`, level: 'info', message: `p2-${i}`, created_at: '2026-01-01T00:00:00Z',
    })),
    next_cursor: '',
    has_more: false,
  }
}

describe('MonitoringPage logs pagination', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource)
    vi.stubGlobal('fetch', vi.fn(async (input: string) => {
      const url = new URL(input, 'http://localhost')
      if (url.pathname === '/api/monitoring/logs') {
        return { ok: true, json: async () => logPage(url.searchParams.get('cursor')) } as Response
      }
      if (url.pathname === '/api/monitoring/alerts') {
        return { ok: true, json: async () => ({ data: [] }) } as Response
      }
      return { ok: true, json: async () => ({}) } as Response
    }))
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('shows Load more, appends the next page, then hides it on the last page', async () => {
    const user = userEvent.setup()
    render(<MemoryRouter><MonitoringPage /></MemoryRouter>)

    // Enter the Logs tab once the initial load resolves.
    const logsTab = await screen.findByRole('tab', { name: 'Logs' })
    await user.click(logsTab)

    // Page 1 rendered.
    await screen.findByText('m0')

    // Load more visible → click → page 2 appended below page 1.
    const loadMore = await screen.findByRole('button', { name: /load more/i })
    await user.click(loadMore)
    await screen.findByText('p2-0')
    expect(screen.getByText('m0')).toBeInTheDocument()

    // Last page reached → Load more disappears.
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /load more/i })).toBeNull()
    })
  }, 15000)

  it('tails live logs: shows LIVE on open and prepends streamed entries', async () => {
    const user = userEvent.setup()
    render(<MemoryRouter><MonitoringPage /></MemoryRouter>)

    await user.click(await screen.findByRole('tab', { name: 'Logs' }))
    await screen.findByText('m0')

    const es = FakeEventSource.instances.find(e => e.url.includes('/api/monitoring/logs/stream'))
    expect(es).toBeTruthy()

    act(() => es!.emitOpen())
    expect(await screen.findByText('LIVE')).toBeInTheDocument()

    act(() => es!.emitMessage({ id: '9999', level: 'error', message: 'live-entry', created_at: '2026-01-01T00:00:00Z' }))
    expect(await screen.findByText('live-entry')).toBeInTheDocument()

    // Duplicate id must not double-insert.
    act(() => es!.emitMessage({ id: '9999', level: 'error', message: 'live-entry', created_at: '2026-01-01T00:00:00Z' }))
    expect(screen.getAllByText('live-entry')).toHaveLength(1)
  })
})
