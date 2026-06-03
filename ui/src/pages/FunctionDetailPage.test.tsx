import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import FunctionDetailPage from './FunctionDetailPage'

const mockFn = {
  id: 'fn-1',
  name: 'hello',
  runtime: 'javascript',
  source: 'export default () => 1',
  trigger: 'http',
  schedule: '',
  env: '{}',
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
})
