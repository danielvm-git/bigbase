import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import CiciPage from './CiciPage'

describe('CiciPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders page header and repo selector initially', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(<MemoryRouter><CiciPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('CI/CD')).toBeInTheDocument()
      expect(screen.getByText('Select a repo to view workflows.')).toBeInTheDocument()
    })
  })

  it('loads workflows and runs when a repo is selected', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'r1', name: 'my-repo' }] }) } as Response)
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'w1', name: 'CI', config: 'on: push' }] }) } as Response)
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'run1', workflow_id: 'w1', status: 'running' }] }) } as Response)

    render(<MemoryRouter><CiciPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Select a repo to view workflows.')).toBeInTheDocument()
    })

    // Select the repo
    const select = document.querySelector('select') as HTMLSelectElement
    fireEvent.change(select, { target: { value: 'r1' } })

    await waitFor(() => {
      // Workflow name should appear
      expect(screen.getByText('CI')).toBeInTheDocument()
    })
  })

  it('shows New Workflow button when repo is selected', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 'r1', name: 'my-repo' }] }) } as Response)
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [] }) } as Response)

    render(<MemoryRouter><CiciPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Select a repo to view workflows.')).toBeInTheDocument()
    })

    const select = document.querySelector('select') as HTMLSelectElement
    fireEvent.change(select, { target: { value: 'r1' } })

    await waitFor(() => {
      expect(screen.getByText('New Workflow')).toBeInTheDocument()
    })
  })

  it('renders header even when fetch fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

    render(<MemoryRouter><CiciPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('CI/CD')).toBeInTheDocument()
      expect(screen.getByText('Select a repo to view workflows.')).toBeInTheDocument()
    })
  })
})
