import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

const getGitHubStatus = vi.fn()
const getGitHubRepos = vi.fn()
const getGitRepos = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getGitHubStatus: (...args: unknown[]) => getGitHubStatus(...args),
  getGitHubRepos: (...args: unknown[]) => getGitHubRepos(...args),
  getGitRepos: (...args: unknown[]) => getGitRepos(...args),
  createSite: vi.fn(),
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
