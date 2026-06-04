import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

function mockDashboardAPIs() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input?.toString?.() ?? ''
    if (url.includes('/api/auth/me')) return okJson({ id: 1, email: 'admin@example.com' })
    if (url.includes('/health')) return okJson({ status: 'ok', components: 16, running: 16 })
    if (url.includes('/api/sites')) return okJson({ data: [{ id: 's1', name: 'web-app' }] })
    if (url.includes('/api/git/repos')) return okJson({ data: [{ id: 'r1', name: 'repo' }] })
    if (url.includes('/api/deploy')) return okJson({
      data: [{
        id: 'd1',
        repo_id: 'repo-main',
        branch: 'main',
        commit_sha: 'a8d4517',
        status: 'running',
        app_type: 'static',
        url: 'http://localhost:4000',
        created_at: new Date(Date.now() - 120000).toISOString(),
      }],
    })
    if (url.includes('/api/functions')) return okJson({ data: [{ id: 'fn1', name: 'hello', runtime: 'node', status: 'active' }] })
    if (url.includes('/api/users')) return okJson({ data: [{ id: 'u1', email: 'u@test.com' }] })
    if (url.includes('/api/monitoring/metrics')) return okJson({
      system: { cpu_percent: 23.5, memory_mb: 512, goroutines: 42, uptime_seconds: 3600 },
      requests: { total: 142, avg_latency_ms: 45, by_status: { '200': 130, '500': 2 } },
    })
    return Promise.reject(new Error(`Unmocked fetch: ${url}`))
  })
}

function mockDashboardAPIsHealthWarning() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input?.toString?.() ?? ''
    if (url.includes('/api/auth/me')) return okJson({ id: 1, email: 'admin@example.com' })
    if (url.includes('/health')) return okJson({ status: 'degraded', components: 16, running: 12 })
    if (url.includes('/api/monitoring/metrics')) return okJson({
      system: { cpu_percent: 10, memory_mb: 128, goroutines: 10, uptime_seconds: 60 },
    })
    return okJson({ data: [] })
  })
}

function mockDashboardAPIsNoUser() {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input?.toString?.() ?? ''
    if (url.includes('/api/auth/me')) return okJson({}, false)
    if (url.includes('/health')) return okJson({ status: 'ok', components: 16, running: 16 })
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

  it('renders welcome header and create site action', async () => {
    mockDashboardAPIs()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText(/Welcome back, admin/)).toBeInTheDocument()
      expect(screen.getByText(/what's running on your BigBase instance/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /create site/i })).toBeInTheDocument()
    })
  })

  it('shows system status with CPU and Memory', async () => {
    mockDashboardAPIs()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('All systems operational')).toBeInTheDocument()
      expect(screen.getByText('CPU')).toBeInTheDocument()
      expect(screen.getByText('Memory')).toBeInTheDocument()
      expect(screen.getByText('23.5')).toBeInTheDocument()
      expect(screen.getByText('512 MB')).toBeInTheDocument()
    })
  })

  it('shows warning when health degraded', async () => {
    mockDashboardAPIsHealthWarning()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('System issues detected')).toBeInTheDocument()
      expect(screen.getByText('some components offline')).toBeInTheDocument()
    })
  })

  it('warms metrics with a second fetch for CPU sampling', async () => {
    mockDashboardAPIs()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('Welcome back, admin')).toBeInTheDocument()
    })
    const metricCalls = vi.mocked(fetch).mock.calls.filter(c => String(c[0]).includes('/api/monitoring/metrics'))
    expect(metricCalls.length).toBeGreaterThanOrEqual(2)
  })

  it('shows four prototype stat cards', async () => {
    mockDashboardAPIs()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('Sites')).toBeInTheDocument()
      expect(screen.getByText('Functions')).toBeInTheDocument()
      expect(screen.getByText('Git Repos')).toBeInTheDocument()
      expect(screen.getByText('Users')).toBeInTheDocument()
      expect(screen.queryByText('Messages')).not.toBeInTheDocument()
      expect(screen.queryByText('Files')).not.toBeInTheDocument()
    })
  })

  it('shows recent deployments and jump back in', async () => {
    mockDashboardAPIs()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('Recent deployments')).toBeInTheDocument()
      expect(screen.getByText('Jump back in')).toBeInTheDocument()
      expect(screen.getByText(/Deploy a site from GitHub/)).toBeInTheDocument()
      expect(screen.getByText(/a8d4517/)).toBeInTheDocument()
    })
  })

  it('shows activity from recent deployments', async () => {
    mockDashboardAPIs()
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('Activity')).toBeInTheDocument()
      expect(screen.getByText(/Deploy repo-ma/)).toBeInTheDocument()
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
      expect(screen.getByText(/loading/i)).toBeInTheDocument()
    })
  })
})
