import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'

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
  })
})
