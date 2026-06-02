import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

function mockDashboardAPIs() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input?.toString?.() ?? ''
    if (url.includes('/api/auth/me')) return okJson({ id: 1, email: 'admin@example.com' })
    if (url.includes('/health')) return okJson({ status: 'ok', components: 16 })
    if (url.includes('/api/git/repos')) return okJson({ data: [{ id: 'r1', name: 'repo' }] })
    if (url.includes('/api/deploy')) return okJson({
      data: [{ id: 'd1', repo_id: 'r1', branch: 'main', status: 'running', app_type: 'static', url: 'http://localhost:4000', created_at: '2026-06-01T10:00:00Z' }],
    })
    if (url.includes('/api/messaging/messages')) return okJson({
      data: [{ id: 'm1', channel: 'email', to_addr: 'a@b.com', subject: 'Welcome', status: 'sent', created_at: '2026-06-01T10:00:00Z' }],
    })
    if (url.includes('/api/storage/files')) return okJson({ data: [{ id: 'f1', name: 'test.png', size: 1024, mime_type: 'image/png', created_at: '2026-06-01T10:00:00Z' }] })
    if (url.includes('/api/functions')) return okJson({ data: [{ id: 'fn1', name: 'hello', runtime: 'node', status: 'active' }] })
    if (url.includes('/api/monitoring/metrics')) return okJson({
      system: { cpu_percent: 23.5, memory_mb: 512, goroutines: 42, uptime_seconds: 3600 },
      requests: { total: 142, avg_latency_ms: 45, by_status: { '200': 130, '404': 10, '500': 2 } },
    })
    return Promise.reject(new Error(`Unmocked fetch: ${url}`))
  })
}

function mockDashboardAPIsHealthWarning() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input?.toString?.() ?? ''
    if (url.includes('/api/auth/me')) return okJson({ id: 1, email: 'admin@example.com' })
    if (url.includes('/health')) return okJson({ status: 'degraded', components: 12 })
    return okJson({ data: [] })
  })
}

function mockDashboardAPIsNoUser() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input?.toString?.() ?? ''
    if (url.includes('/api/auth/me')) return okJson({}, false)
    if (url.includes('/health')) return okJson({ status: 'ok', components: 16 })
    return okJson({ data: [] })
  })
}

function okJson(data: unknown, ok = true): Promise<Response> {
  return Promise.resolve({ ok, json: () => Promise.resolve(data) } as Response)
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

  it('shows loading state when all fetches fail', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    await waitFor(() => {
      // Should still render without crashing
      expect(screen.getByText(/loading/i)).toBeInTheDocument()
    })
  })
})
