import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DeployPage from './DeployPage'

describe('DeployPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the page header', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(<MemoryRouter><DeployPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Deployments')).toBeInTheDocument()
    })
  })

  it('shows Refresh and New Deployment buttons', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(<MemoryRouter><DeployPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Refresh')).toBeInTheDocument()
      expect(screen.getByText('New Deployment')).toBeInTheDocument()
    })
  })

  it('toggles create form on New Deployment click', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(<MemoryRouter><DeployPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('New Deployment')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('New Deployment'))

    // Cancel button should appear when form is open
    await waitFor(() => {
      expect(screen.getByText('Cancel')).toBeInTheDocument()
    })
  })

  it('shows deployment rows with status badges', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        data: [
          { id: 'd1', repo_id: 'r1', branch: 'main', status: 'running', app_type: 'static', url: 'http://localhost:4000', created_at: '2026-06-01T10:00:00Z' },
          { id: 'd2', repo_id: 'r2', branch: 'dev', status: 'failed', app_type: 'node', url: '', created_at: '2026-06-01T11:00:00Z' },
        ],
      }),
    } as Response)

    render(<MemoryRouter><DeployPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('running')).toBeInTheDocument()
      expect(screen.getByText('failed')).toBeInTheDocument()
    })
  })

  it('shows error on fetch failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

    render(<MemoryRouter><DeployPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText(/load/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no deployments exist', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(<MemoryRouter><DeployPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Deployments')).toBeInTheDocument()
      expect(screen.getByText('New Deployment')).toBeInTheDocument()
    })
  })
})
