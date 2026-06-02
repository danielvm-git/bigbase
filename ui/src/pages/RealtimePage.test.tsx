import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import RealtimePage from './RealtimePage'

const mockStatusResponse = {
  total_connections: 2,
  total_rooms: 3,
  connections: [
    { user_id: 1, rooms: ['collection:posts', 'collection:users'] },
    { user_id: 2, rooms: ['collection:posts'] },
  ],
}

describe('RealtimePage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders connection count from API', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockStatusResponse),
    } as Response)

    render(
      <MemoryRouter>
        <RealtimePage />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/2 active connections/i)).toBeInTheDocument()
    })
  })

  it('renders room count', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockStatusResponse),
    } as Response)

    render(
      <MemoryRouter>
        <RealtimePage />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/3 rooms/i)).toBeInTheDocument()
    })
  })

  it('shows connection details in table', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockStatusResponse),
    } as Response)

    render(
      <MemoryRouter>
        <RealtimePage />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('User #1')).toBeInTheDocument()
      expect(screen.getByText('User #2')).toBeInTheDocument()
    })
  })

  it('shows channel subscriptions per connection', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockStatusResponse),
    } as Response)

    render(
      <MemoryRouter>
        <RealtimePage />
      </MemoryRouter>
    )

    await waitFor(() => {
      // 'collection:posts' appears for both users — confirm count
      const postsElements = screen.getAllByText('collection:posts')
      expect(postsElements).toHaveLength(2)
      expect(screen.getByText('collection:users')).toBeInTheDocument()
    })
  })

  it('shows empty state when no connections', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        total_connections: 0,
        total_rooms: 0,
        connections: [],
      }),
    } as Response)

    render(
      <MemoryRouter>
        <RealtimePage />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/no active connections/i)).toBeInTheDocument()
    })
  })

  it('shows error state on fetch failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValueOnce(new Error('Network error'))

    render(
      <MemoryRouter>
        <RealtimePage />
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('Failed to load realtime status')).toBeInTheDocument()
    })
  })
})
