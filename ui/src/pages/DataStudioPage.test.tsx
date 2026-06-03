import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ToastProvider } from '../context/ToastContext'
import DataStudioPage from './DataStudioPage'

const mockRecords = [
  { id: 1, name: 'Alice', score: 10 },
  { id: 2, name: 'Bob', score: 20 },
]

function renderPage() {
  return render(
    <MemoryRouter>
      <ToastProvider>
        <DataStudioPage />
      </ToastProvider>
    </MemoryRouter>,
  )
}

describe('DataStudioPage schema mode', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('shows schema columns and Add column action', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((url: string | URL | Request) => {
      const path = String(url)
      if (path.endsWith('/api/collections/')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ data: ['users'] }),
        } as Response)
      }
      if (path.includes('/api/collections/users')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ data: mockRecords }),
        } as Response)
      }
      return Promise.reject(new Error(`unexpected fetch: ${path}`))
    })

    renderPage()
    await waitFor(() => {
      expect(screen.getByText('users')).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText('users'))
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Schema' })).toBeInTheDocument()
    })
    fireEvent.click(screen.getByRole('button', { name: 'Schema' }))
    await waitFor(() => {
      expect(screen.getByText('id')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Add column' })).toBeInTheDocument()
    })
  })
})
