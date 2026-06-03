import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes, createMemoryRouter, RouterProvider } from 'react-router-dom'
import FunctionDetailPage, { FunctionLogsRedirect } from './FunctionDetailPage'

const mockFn = {
  id: 'fn-1',
  name: 'hello',
  runtime: 'javascript',
  source: 'export default () => 1',
  trigger: 'http',
  schedule: '',
  env: { FOO: 'bar' },
  timeout: 30,
  created_at: '2026-06-01T10:00:00Z',
}

describe('FunctionDetailPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('loads function by id and shows tabs', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockFn),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-1']}>
        <Routes>
          <Route path="/functions/:id" element={<FunctionDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'hello' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Code' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Logs' })).toBeInTheDocument()
    })
  })

  it('shows formatted env on variables tab', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockFn),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-1?tab=variables']}>
        <Routes>
          <Route path="/functions/:id" element={<FunctionDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByDisplayValue(/"FOO"/)).toBeInTheDocument()
    })
  })

  it('saves variables with object env in PUT body', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockFn),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ ...mockFn, env: { FOO: 'baz' } }),
      } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-1?tab=variables']}>
        <Routes>
          <Route path="/functions/:id" element={<FunctionDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByDisplayValue(/"FOO"/)).toBeInTheDocument()
    })

    fireEvent.change(screen.getByDisplayValue(/"FOO"/), {
      target: { value: '{\n  "FOO": "baz"\n}' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save variables' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2)
      const putCall = fetchMock.mock.calls[1]
      expect(putCall[0]).toBe('/api/functions/fn-1')
      const body = JSON.parse(String(putCall[1]?.body))
      expect(body.env).toEqual({ FOO: 'baz' })
    })
  })

  it('shows error when function is missing', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ error: 'not found' }),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/missing']}>
        <Routes>
          <Route path="/functions/:id" element={<FunctionDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByText('Function not found')).toBeInTheDocument()
    })
  })

  it('defaults invalid tab query to code panel', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockFn),
    } as Response)

    render(
      <MemoryRouter initialEntries={['/functions/fn-1?tab=invalid']}>
        <Routes>
          <Route path="/functions/:id" element={<FunctionDetailPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Save code' })).toBeInTheDocument()
    })
  })
})

describe('FunctionLogsRedirect', () => {
  it('redirects legacy logs route to detail logs tab', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockFn),
    } as Response)

    const router = createMemoryRouter(
      [
        { path: '/functions/:id/logs', element: <FunctionLogsRedirect /> },
        { path: '/functions/:id', element: <FunctionDetailPage /> },
      ],
      { initialEntries: ['/functions/fn-1/logs'] },
    )

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Logs' })).toBeInTheDocument()
    })
  })
})
