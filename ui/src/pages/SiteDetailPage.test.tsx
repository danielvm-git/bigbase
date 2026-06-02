import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

const mockSite = {
  id: 'site-1', name: 'My Site', full_name: 'org/mysite',
  git_repo_id: 'r1', production_branch: 'main', status: 'running',
  url: 'https://mysite.example.com',
}

const mockDeployments = [
  { id: 'd1', repo_id: 'r1', branch: 'main', commit_sha: 'abc1234', status: 'running', url: 'http://localhost:4000', app_type: 'static', created_at: '2026-06-01T10:00:00Z' },
  { id: 'd2', repo_id: 'r1', branch: 'main', commit_sha: 'def5678', status: 'failed', url: '', app_type: 'node', created_at: '2026-06-01T09:00:00Z' },
]

vi.mock('../lib/sitesData', () => ({
  getSite: () => Promise.resolve({ data: mockSite, previewMode: false }),
  getSiteDeployments: () => Promise.resolve({ data: mockDeployments, previewMode: false }),
}))

// eslint-disable-next-line import/first
import SiteDetailPage from './SiteDetailPage'

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/deploy/site-1']}>
      <Routes>
        <Route path="/deploy/:siteId" element={<SiteDetailPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('SiteDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders site name as page header', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'My Site' })).toBeInTheDocument()
    })
  })

  it('shows branch info', async () => {
    renderPage()
    await waitFor(() => {
      const branchEls = screen.getAllByText(/main/)
      expect(branchEls.length).toBeGreaterThan(0)
    })
  })

  it('shows repo full name', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/org\/mysite/)).toBeInTheDocument()
    })
  })

  it('renders StatusTimeline steps', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('pending')).toBeInTheDocument()
      expect(screen.getByText('building')).toBeInTheDocument()
      expect(screen.getByText('deploying')).toBeInTheDocument()
    })
  })

  it('shows deployment commit hashes', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('abc1234')).toBeInTheDocument()
      expect(screen.getByText('def5678')).toBeInTheDocument()
    })
  })

  it('shows status badges', async () => {
    renderPage()
    await waitFor(() => {
      const runningBadges = screen.getAllByText('running')
      expect(runningBadges.length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('failed')).toBeInTheDocument()
    })
  })

  it('shows back link', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('← All sites')).toBeInTheDocument()
    })
  })

  it('renders Redeploy button', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Redeploy')).toBeInTheDocument()
      expect(screen.getByText('Refresh')).toBeInTheDocument()
    })
  })
})
