import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import UsersPage from './UsersPage'

const mockUsers = [
  { id: 1, email: 'admin@example.com', role: 'owner', verified: true, created_at: '2026-06-01T10:00:00Z' },
  { id: 2, email: 'member@example.com', role: 'member', verified: false, created_at: '2026-06-01T11:00:00Z' },
  { id: 3, email: 'dev@example.com', role: 'admin', verified: true, created_at: '2026-06-01T12:00:00Z' },
]

describe('UsersPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  function mockUsersOk() {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: mockUsers }),
    } as Response)
  }

  it('renders page header', async () => {
    mockUsersOk()
    render(<MemoryRouter><UsersPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('Users')).toBeInTheDocument()
    })
  })

  it('renders all user emails', async () => {
    mockUsersOk()
    render(<MemoryRouter><UsersPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument()
      expect(screen.getByText('member@example.com')).toBeInTheDocument()
      expect(screen.getByText('dev@example.com')).toBeInTheDocument()
    })
  })

  it('shows user IDs', async () => {
    mockUsersOk()
    render(<MemoryRouter><UsersPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('1')).toBeInTheDocument()
    })
  })

  it('shows Delete buttons', async () => {
    mockUsersOk()
    render(<MemoryRouter><UsersPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getAllByText('Delete')).toHaveLength(3)
    })
  })

  it('deletes a user when confirmed', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: mockUsers }) } as Response)
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) } as Response)
      .mockResolvedValue({ ok: true, json: () => Promise.resolve({ data: mockUsers.slice(1) }) } as Response)

    render(<MemoryRouter><UsersPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument()
    })

    fireEvent.click(screen.getAllByText('Delete')[0])

    await waitFor(() => {
      expect(screen.queryByText('admin@example.com')).not.toBeInTheDocument()
    })
  })

  it('does not delete when cancelled', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    mockUsersOk()

    render(<MemoryRouter><UsersPage /></MemoryRouter>)
    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument()
    })

    fireEvent.click(screen.getAllByText('Delete')[0])

    await waitFor(() => {
      expect(screen.getByText('admin@example.com')).toBeInTheDocument()
    })
  })

  it('shows error state on fetch fail', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: 'Could not load users' }),
    } as Response)

    render(<MemoryRouter><UsersPage /></MemoryRouter>)

    await waitFor(() => {
      expect(screen.getByText('Could not load users')).toBeInTheDocument()
    })
  })

  it('shows loading state', () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => new Promise(() => {}))

    render(<MemoryRouter><UsersPage /></MemoryRouter>)

    expect(screen.getByText('Loading users...')).toBeInTheDocument()
  })
})
