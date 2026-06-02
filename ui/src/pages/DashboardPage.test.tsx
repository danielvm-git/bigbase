import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

function mockDashboardAPIs() {
  vi.spyOn(globalThis, 'fetch')
    // 1. /api/auth/me
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ id: 1, email: 'admin@example.com' }) } as Response)
    // 2. /health
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ status: 'ok', components: 16 }) } as Response)
    // 3. /api/git/repos
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'r1', name: 'repo' }] }) } as Response)
    // 4. /api/deploy
    .mockResolvedValueOnce({
      ok: true, json: () => Promise.resolve({
        data: [
          { id: 'd1', repo_id: 'r1', branch: 'main', status: 'running', app_type: 'static', url: 'http://localhost:4000', created_at: '2026-06-01T10:00:00Z' },
        ],
      }),
    } as Response)
    // 5. /api/messaging/messages
    .mockResolvedValueOnce({
      ok: true, json: () => Promise.resolve({
        data: [
          { id: 'm1', channel: 'email', to_addr: 'a@b.com', subject: 'Welcome', status: 'sent', created_at: '2026-06-01T10:00:00Z' },
        ],
      }),
    } as Response)
    // 6. /api/storage/files
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'f1', name: 'test.png', size: 1024, mime_type: 'image/png', created_at: '2026-06-01T10:00:00Z' }] }) } as Response)
    // 7. /api/functions
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'fn1', name: 'hello', runtime: 'node', status: 'active' }] }) } as Response)
    // 8. /api/monitoring/metrics
    .mockResolvedValueOnce({
      ok: true, json: () => Promise.resolve({
        system: { cpu_percent: 23.5, memory_mb: 512, goroutines: 42, uptime_seconds: 3600 },
        requests: { total: 142, avg_latency_ms: 45, by_status: { '200': 130, '404': 10, '500': 2 } },
      }),
    } as Response)
}

function mockDashboardAPIsHealthWarning() {
  vi.spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ id: 1, email: 'admin@example.com' }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ status: 'degraded', components: 12 }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ system: { cpu_percent: 95, memory_mb: 512, goroutines: 42, uptime_seconds: 3600 }, requests: { total: 0, avg_latency_ms: 0, by_status: {} } }) } as Response)
}

function mockDashboardAPIsNoUser() {
  vi.spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce({ ok: false, json: () => Promise.resolve({}) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ status: 'ok', components: 16 }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ system: { cpu_percent: 0, memory_mb: 512, goroutines: 0, uptime_seconds: 0 }, requests: { total: 0, avg_latency_ms: 0, by_status: {} } }) } as Response)
}

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders page header and user info', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
      expect(screen.getByText('admin@example.com')).toBeInTheDocument()
    })
  })

  it('shows health banner with OK status', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('All systems operational')).toBeInTheDocument()
      expect(screen.getByText(/16 components? running/)).toBeInTheDocument()
    })
  })

  it('shows warning banner when health degraded', async () => {
    mockDashboardAPIsHealthWarning()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('System issues detected')).toBeInTheDocument()
    })
  })

  it('shows quick action buttons', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('+ Deploy Site')).toBeInTheDocument()
      expect(screen.getByText('⚡ Run Function')).toBeInTheDocument()
      expect(screen.getByText('📦 Create Collection')).toBeInTheDocument()
    })
  })

  it('shows stat cards for resources', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Git Repos')).toBeInTheDocument()
      expect(screen.getByText('Deployments')).toBeInTheDocument()
      expect(screen.getByText('Messages')).toBeInTheDocument()
      expect(screen.getByText('Files')).toBeInTheDocument()
      expect(screen.getByText('Functions')).toBeInTheDocument()
    })
  })

  it('shows request rate and error rate from metrics', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Request Rate')).toBeInTheDocument()
      expect(screen.getByText('Error Rate')).toBeInTheDocument()
      // 500 count is 2
      expect(screen.getByText('2')).toBeInTheDocument()
    })
  })

  it('shows CPU and component count from metrics', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('CPU')).toBeInTheDocument()
      expect(screen.getByText('Components')).toBeInTheDocument()
      expect(screen.getByText('23.5%')).toBeInTheDocument()
    })
  })

  it('shows recent deployments table', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Recent Deployments')).toBeInTheDocument()
      // "static" appears inside an activity-text span alongside <code>repo_id</code>
      expect(screen.getByText(/static/)).toBeInTheDocument()
    })
  })

  it('shows recent messages table', async () => {
    mockDashboardAPIs()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Recent Messages')).toBeInTheDocument()
      expect(screen.getByText('a@b.com')).toBeInTheDocument()
    })
  })

  it('shows loading state when user not loaded', async () => {
    mockDashboardAPIsNoUser()

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText(/loading/i)).toBeInTheDocument()
    })
  })

  it('handles all API calls failing gracefully', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      // Should still render without crashing
      expect(screen.getByText(/loading/i)).toBeInTheDocument()
    })
  })
})
