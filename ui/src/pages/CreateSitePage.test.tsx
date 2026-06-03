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
    expect(screen.getAllByText('Reconnect GitHub').length).toBeGreaterThanOrEqual(1)
    expect(screen.queryByPlaceholderText('Search repositories…')).not.toBeInTheDocument()
  })

  it('shows Connect when connected but repo list is empty', async () => {
    getGitHubStatus.mockResolvedValue({
      data: { connected: true, configured: true, login: 'testuser' },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({ data: [], previewMode: false })

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Connect GitHub' })).toBeInTheDocument()
    })
    expect(screen.queryByPlaceholderText('Search repositories…')).not.toBeInTheDocument()
  })

  it('shows Connect GitHub when not connected', async () => {
    getGitHubStatus.mockResolvedValue({
      data: { connected: false, configured: true },
      previewMode: false,
    })
    getGitHubRepos.mockResolvedValue({ data: [], previewMode: false })

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Connect GitHub' })).toBeInTheDocument()
    })
    expect(screen.getByText('Install the BigBase GitHub App to list and deploy your repositories.')).toBeInTheDocument()
  })
})
