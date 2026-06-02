import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import FunctionLogsPage from './FunctionLogsPage'

const mockLogsResponse = {
  data: [
    {
      id: 'exec-1',
      status: 'success',
      logs: ['Starting execution...', 'step 1', 'step 2'],
      error: '',
      created_at: '2026-06-01T10:30:00Z',
    },
    {
      id: 'exec-2',
      status: 'error',
      logs: ['Starting execution...', 'oops'],
      error: 'ReferenceError: x is not defined',
      created_at: '2026-06-01T10:25:00Z',
    },
  ],
}

describe('FunctionLogsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders execution history entries', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockLogsResponse),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-123/logs']}>
        <Routes>
          <Route path="/functions/:id/logs" element={<FunctionLogsPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/execution history/i)).toBeInTheDocument()
    })
  })

  it('shows status badge per execution', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockLogsResponse),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-123/logs']}>
        <Routes>
          <Route path="/functions/:id/logs" element={<FunctionLogsPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('success')).toBeInTheDocument()
      expect(screen.getByText('error')).toBeInTheDocument()
    })
  })

  it('shows log entries per execution', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockLogsResponse),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-123/logs']}>
        <Routes>
          <Route path="/functions/:id/logs" element={<FunctionLogsPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      const startElements = screen.getAllByText(/Starting execution/)
      expect(startElements.length).toBeGreaterThan(0)
      expect(screen.getByText(/step 1/)).toBeInTheDocument()
    })
  })

  it('shows error message for failed executions', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockLogsResponse),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-123/logs']}>
        <Routes>
          <Route path="/functions/:id/logs" element={<FunctionLogsPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/ReferenceError/)).toBeInTheDocument()
    })
  })

  it('shows empty state when no executions', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: [] }),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-123/logs']}>
        <Routes>
          <Route path="/functions/:id/logs" element={<FunctionLogsPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/no executions yet/i)).toBeInTheDocument()
    })
  })

  it('shows error state on fetch failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

    render(
      <MemoryRouter initialEntries={['/functions/fn-123/logs']}>
        <Routes>
          <Route path="/functions/:id/logs" element={<FunctionLogsPage />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('Failed to load execution logs')).toBeInTheDocument()
    })
  })
})
