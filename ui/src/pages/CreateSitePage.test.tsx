import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

const getGitHubStatus = vi.fn()
const getGitHubRepos = vi.fn()
const getGitRepos = vi.fn()

const createSite = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getGitHubStatus: (...args: unknown[]) => getGitHubStatus(...args),
  getGitHubRepos: (...args: unknown[]) => getGitHubRepos(...args),
  getGitRepos: (...args: unknown[]) => getGitRepos(...args),
  createSite: (...args: unknown[]) => createSite(...args),
  githubInstallURL: () => '/api/github/install',
}))

vi.mock('../lib/previewMode', () => ({
  isPreviewForced: () => false,
  previewQuerySuffix: () => '',
}))

// eslint-disable-next-line import/first
import CreateSitePage from './CreateSitePage'

function renderPage() {
  return render(
    <MemoryRouter>
      <CreateSitePage />
    </MemoryRouter>,
  )
}

describe('CreateSitePage layout', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getGitRepos.mockResolvedValue({ data: [], previewMode: false })
    getGitHubStatus.mockResolvedValue({
      data: { connected: true, configured: true, login: 'testuser' },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({
      data: [
        { id: 1, full_name: 'acme/one', default_branch: 'main', private: false },
        { id: 2, full_name: 'acme/two', default_branch: 'main', private: true, description: 'Second' },
      ],
      previewMode: false,
    })
  })

  it('applies grid layout to choice cards', async () => {
    const { container } = renderPage()
    await waitFor(() => {
      expect(screen.getByText("Where's your code?")).toBeInTheDocument()
    })
    const grid = container.querySelector('.choice-grid')
    expect(grid).toBeTruthy()
    expect(getComputedStyle(grid!).display).toBe('grid')
  })

  it('renders repo picker as a vertical list', async () => {
    const { container } = renderPage()
    await waitFor(() => {
      expect(screen.getByText('acme/one')).toBeInTheDocument()
    })
    const picker = container.querySelector('.repo-picker')
    expect(picker).toBeTruthy()
    expect(getComputedStyle(picker!).flexDirection).toBe('column')
    expect(container.querySelectorAll('.repo-picker-item').length).toBe(2)
  })

  it('uses a 3-step wizard rail', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Source')).toBeInTheDocument()
    })
    expect(screen.getByText('Configure')).toBeInTheDocument()
    expect(screen.getByText('Deploy')).toBeInTheDocument()
    expect(screen.queryByText('Review')).not.toBeInTheDocument()
  })

  it('does not render pre-fix copy or 4-step review wizard', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText("Where's your code?")).toBeInTheDocument()
    })
    expect(screen.queryByText('How do you want to add your app?')).not.toBeInTheDocument()
    expect(screen.queryByText('Manual upload')).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Create a new site' })).toBeInTheDocument()
  })

  it('wizard step rail uses horizontal flex (not stacked ol numbers)', async () => {
    const { container } = renderPage()
    await waitFor(() => {
      expect(screen.getByText('Source')).toBeInTheDocument()
    })
    const steps = container.querySelector('.wizard-steps')
    expect(steps).toBeTruthy()
    expect(getComputedStyle(steps!).display).toBe('flex')
    expect(container.querySelectorAll('.wizard-step-label').length).toBe(3)
  })
})

describe('CreateSitePage deploy step', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getGitRepos.mockResolvedValue({ data: [], previewMode: false })
    getGitHubStatus.mockResolvedValue({
      data: { connected: true, configured: true, login: 'testuser' },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({
      data: [{ id: 1, full_name: 'acme/app', default_branch: 'main', private: false }],
      previewMode: false,
    })
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows build terminal on deploy step', async () => {
    createSite.mockResolvedValue({
      previewMode: false,
      site: { id: 'site-1', name: 'my-app', git_repo_id: 'repo-1', production_branch: 'main' },
      deployment: {
        id: 'dep-1',
        status: 'building',
        url: 'https://my-app.bigbase.click',
      },
    })
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const path = typeof input === 'string' ? input : input instanceof URL ? input.pathname : input.url
      if (path.includes('/logs')) {
        return {
          ok: true,
          json: async () => ({
            deployment_id: 'dep-1',
            status: 'building',
            lines: ['→ Cloning repository'],
            log_available: true,
          }),
        } as Response
      }
      return {
        ok: true,
        json: async () => ({ status: 'building', url: 'https://my-app.bigbase.click' }),
      } as Response
    })

    const { container } = renderPage()
    await waitFor(() => {
      expect(screen.getByText('acme/app')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('acme/app').closest('button')!)
    fireEvent.click(screen.getByRole('button', { name: /Continue →/i }))

    await waitFor(() => {
      expect(screen.getByText('Configure your site')).toBeInTheDocument()
    })
    const nameInput = document.querySelector('.input-group input.input') as HTMLInputElement
    fireEvent.change(nameInput, { target: { value: 'my-app' } })
    fireEvent.click(screen.getByRole('button', { name: 'Deploy' }))

    await waitFor(() => {
      expect(screen.getByTestId('stream-log')).toBeInTheDocument()
    })
    expect(container.querySelector('.deploy-step-layout')).toBeTruthy()
    expect(screen.getByText('Build output')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText('→ Cloning repository')).toBeInTheDocument()
    })
    expect(screen.getByText('Live')).toBeInTheDocument()
  })
})

describe('CreateSitePage GitHub source', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getGitRepos.mockResolvedValue({ data: [], previewMode: false })
  })

  it('shows Reconnect when connected but repos fetch fails', async () => {
    getGitHubStatus.mockResolvedValue({
      data: { connected: true, configured: true, login: 'testuser' },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({
      data: [],
      previewMode: false,
      error: 'failed to list repositories from GitHub',
    })

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('failed to list repositories from GitHub')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: 'Reconnect GitHub' })).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('Search repositories…')).not.toBeInTheDocument()
  })

  it('shows Authorize when connected but repo list is empty', async () => {
    getGitHubStatus.mockResolvedValue({
      data: { connected: true, configured: true, login: 'testuser' },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({ data: [], previewMode: false })

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Authorize GitHub' })).toBeInTheDocument()
    })
    expect(screen.queryByPlaceholderText('Search repositories…')).not.toBeInTheDocument()
  })

  it('shows Authorize GitHub when not connected', async () => {
    getGitHubStatus.mockResolvedValue({
      data: { connected: false, configured: true },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({ data: [], previewMode: false })

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Authorize GitHub' })).toBeInTheDocument()
    })
    expect(
      screen.getByText(/Choose which repositories BigBase can deploy/i),
    ).toBeInTheDocument()
  })
})
