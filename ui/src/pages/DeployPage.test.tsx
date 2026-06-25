import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DeployPage from './DeployPage'

const getSites = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getSites: (...args: unknown[]) => getSites(...args),
  // BuildCachePanel renders at the bottom of the populated list.
  getCacheStats: () => Promise.resolve({ total_entries: 0, total_size_bytes: 0, max_size_bytes: 2147483648 }),
  clearAllCache: () => Promise.resolve({ ok: true }),
  pruneCache: () => Promise.resolve({ ok: true, pruned: 0 }),
  setCacheMaxSize: () => Promise.resolve({ ok: true }),
}))

vi.mock('../lib/previewMode', () => ({
  isPreviewForced: () => false,
  previewQuerySuffix: () => '',
}))

describe('DeployPage (Sites list)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the Sites header and create CTA', async () => {
    getSites.mockResolvedValue({ data: [], previewMode: false })

    render(
      <MemoryRouter>
        <DeployPage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Sites' })).toBeInTheDocument()
    })
    expect(screen.getAllByRole('button', { name: /Create site/i }).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/Deploy and host web apps straight from Git/i)).toBeInTheDocument()
  })

  it('shows empty state when no sites', async () => {
    getSites.mockResolvedValue({ data: [], previewMode: false })

    render(
      <MemoryRouter>
        <DeployPage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('Create your first site')).toBeInTheDocument()
    })
  })

  it('renders site cards in a grid', async () => {
    getSites.mockResolvedValue({
      data: [
        {
          id: 's1',
          name: 'marketing-site',
          full_name: 'acme/marketing-site',
          git_repo_id: 'r1',
          production_branch: 'main',
          root_path: './',
          latest_deployment: {
            id: 'd1',
            repo_id: 'r1',
            branch: 'main',
            commit_sha: 'abc1234',
            status: 'running',
            url: 'http://localhost:10042',
            port: 10042,
            app_type: 'static',
            created_at: new Date().toISOString(),
          },
        },
      ],
      previewMode: false,
    })

    const { container } = render(
      <MemoryRouter>
        <DeployPage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('marketing-site')).toBeInTheDocument()
    })

    const grid = container.querySelector('.site-grid')
    expect(grid).toBeTruthy()
    expect(getComputedStyle(grid!).display).toBe('grid')
    expect(container.querySelectorAll('.site-card').length).toBe(1)
  })

  it('filters sites by search query', async () => {
    getSites.mockResolvedValue({
      data: [
        {
          id: 's1',
          name: 'alpha',
          full_name: 'a/alpha',
          git_repo_id: 'r1',
          production_branch: 'main',
          root_path: './',
        },
        {
          id: 's2',
          name: 'beta',
          full_name: 'b/beta',
          git_repo_id: 'r2',
          production_branch: 'dev',
          root_path: './',
        },
      ],
      previewMode: false,
    })

    render(
      <MemoryRouter>
        <DeployPage />
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('alpha')).toBeInTheDocument()
      expect(screen.getByText('beta')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByPlaceholderText('Search sites'), { target: { value: 'alpha' } })

    expect(screen.getByText('alpha')).toBeInTheDocument()
    expect(screen.queryByText('beta')).not.toBeInTheDocument()
  })
})
