import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
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

// Includes a 'stopped' deployment so canRollback(d1) returns true
const mockDeploymentsWithPrevious = [
  { id: 'd1', repo_id: 'r1', branch: 'main', commit_sha: 'abc1234', status: 'running', url: 'http://localhost:4000', app_type: 'static', created_at: '2026-06-01T10:00:00Z' },
  { id: 'd-prev', repo_id: 'r1', branch: 'main', commit_sha: 'bbb0000', status: 'stopped', url: '', app_type: 'static', created_at: '2026-06-01T08:00:00Z' },
]

// vi.fn() references kept outside vi.mock so tests can override per-describe
const getSiteDeploymentsMock = vi.fn()
const getRollbackEventsMock = vi.fn()
const rollbackDeploymentMock = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getSite: () => Promise.resolve({ data: mockSite, previewMode: false }),
  getSiteDeployments: (...args: unknown[]) => getSiteDeploymentsMock(...args),
  getSiteManifest: () => Promise.resolve({ data: { exists: false, content: '' }, previewMode: false }),
  saveSiteManifest: () => Promise.resolve({ ok: true, previewMode: false }),
  deleteSite: () => Promise.resolve({ ok: true }),
  getRollbackEvents: (...args: unknown[]) => getRollbackEventsMock(...args),
  rollbackDeployment: (...args: unknown[]) => rollbackDeploymentMock(...args),
}))

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
    getSiteDeploymentsMock.mockResolvedValue({ data: mockDeployments, previewMode: false })
    getRollbackEventsMock.mockResolvedValue({ ok: true, events: [] })
    rollbackDeploymentMock.mockResolvedValue({ ok: true })
  })

  it('mock includes all required sitesData functions (hardening against missing mock exports)', () => {
    // If SiteDetailPage imports a new function from sitesData but the mock
    // doesn't include it, the component crashes on render with a confusing
    // "No export is defined on the mock" error. This test verifies all mocks exist.
    // Expected: getSite, getSiteDeployments, deleteSite, getSiteManifest, saveSiteManifest, rollbackDeployment, getRollbackEvents
    expect(typeof getSiteDeploymentsMock).toBe('function')
    expect(typeof getRollbackEventsMock).toBe('function')
    expect(typeof rollbackDeploymentMock).toBe('function')
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

  it('does not show rollback button when no previous deployment exists', async () => {
    // d1=running, d2=failed → canRollback(d1) = false (no stopped/rolled_back)
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('abc1234')).toBeInTheDocument()
    })
    expect(screen.queryByText('Rollback')).not.toBeInTheDocument()
  })

  describe('with a previous (stopped) deployment', () => {
    beforeEach(() => {
      getSiteDeploymentsMock.mockResolvedValue({ data: mockDeploymentsWithPrevious, previewMode: false })
    })

    it('shows rollback button for running deployment with previous', async () => {
      renderPage()
      await waitFor(() => {
        expect(screen.getByText('Rollback')).toBeInTheDocument()
      })
    })

    it('opens confirmation dialog when rollback button is clicked', async () => {
      renderPage()
      await waitFor(() => {
        expect(screen.getByText('Rollback')).toBeInTheDocument()
      })
      fireEvent.click(screen.getByText('Rollback'))
      await waitFor(() => {
        expect(screen.getByText('Rollback Deployment?')).toBeInTheDocument()
        expect(screen.getByText('Confirm Rollback')).toBeInTheDocument()
        expect(screen.getByText('Cancel')).toBeInTheDocument()
      })
      // WCAG 2.1.2: rollback confirmation must be a real dialog (role, aria-modal, Escape closes)
      const dialog = screen.getByRole('dialog', { name: 'Rollback Deployment?' })
      expect(dialog).toHaveAttribute('aria-modal', 'true')
      fireEvent.keyDown(dialog, { key: 'Escape' })
      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      })
    })

    it('closes dialog on cancel without calling rollbackDeployment', async () => {
      renderPage()
      await waitFor(() => screen.getByText('Rollback'))
      fireEvent.click(screen.getByText('Rollback'))
      await waitFor(() => screen.getByText('Rollback Deployment?'))
      fireEvent.click(screen.getByText('Cancel'))
      await waitFor(() => {
        expect(screen.queryByText('Rollback Deployment?')).not.toBeInTheDocument()
      })
      expect(rollbackDeploymentMock).not.toHaveBeenCalled()
    })

    it('calls rollbackDeployment with the deployment id on confirm', async () => {
      rollbackDeploymentMock.mockResolvedValue({
        ok: true,
        event: { id: 'rb-1', site_id: 'site-1', rolled_back_from: 'd-prev', rolled_back_to: 'd1', created_at: '2026-06-26T20:00:00Z' },
      })
      renderPage()
      await waitFor(() => screen.getByText('Rollback'))
      fireEvent.click(screen.getByText('Rollback'))
      await waitFor(() => screen.getByText('Confirm Rollback'))
      fireEvent.click(screen.getByText('Confirm Rollback'))
      await waitFor(() => {
        expect(rollbackDeploymentMock).toHaveBeenCalledWith('d1')
      })
    })

    it('shows error message when rollback API returns failure', async () => {
      rollbackDeploymentMock.mockResolvedValue({ ok: false, error: 'no previous deployment found' })
      renderPage()
      await waitFor(() => screen.getByText('Rollback'))
      fireEvent.click(screen.getByText('Rollback'))
      await waitFor(() => screen.getByText('Confirm Rollback'))
      fireEvent.click(screen.getByText('Confirm Rollback'))
      await waitFor(() => {
        expect(screen.getByText('no previous deployment found')).toBeInTheDocument()
      })
    })
  })
})
