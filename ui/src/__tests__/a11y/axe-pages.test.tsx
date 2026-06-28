/**
 * Page-level accessibility structural tests.
 * Verifies landmark regions, heading hierarchy, and ARIA attributes for full pages.
 */
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import UsersPage from '../../pages/UsersPage'
import SettingsPage from '../../pages/SettingsPage'

function wrap(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

const mockUsersOk = () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response(JSON.stringify({ data: [{ id: 1, email: 'a@b.com', created_at: '2026-01-01' }] }), { status: 200 })
  )
}

describe('UsersPage a11y', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('renders one h1 heading', async () => {
    mockUsersOk()
    const { container } = wrap(<UsersPage />)
    await waitFor(() => screen.getByText('Users'))
    expect(container.querySelectorAll('h1')).toHaveLength(1)
  })

  it('table has accessible column headers', async () => {
    mockUsersOk()
    wrap(<UsersPage />)
    await waitFor(() => screen.getByText('a@b.com'))
    expect(screen.getByRole('columnheader', { name: 'Email' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
  })
})

describe('SettingsPage a11y', () => {
  beforeEach(() => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ email: 'admin@example.com', name: 'Admin' }), { status: 200 })
    )
  })

  it('renders one h1 heading', () => {
    const { container } = wrap(<SettingsPage />)
    expect(container.querySelectorAll('h1')).toHaveLength(1)
  })

  it('has Settings heading', () => {
    wrap(<SettingsPage />)
    expect(screen.getByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
  })
})
