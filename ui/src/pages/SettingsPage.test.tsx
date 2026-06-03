import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import SettingsPage from './SettingsPage'

// Stub the API hooks so the page doesn't make network calls.
vi.mock('../hooks/useAuth', () => ({
  useCurrentUser: () => ({
    data: { id: 'u1', email: 'owner@example.com', name: 'Owner' },
    isLoading: false,
  }),
}))

vi.mock('../hooks/useWorkspace', () => ({
  useWorkspace: () => ({
    data: { id: 'w1', name: 'My Workspace' },
    isLoading: false,
  }),
  useMembers: () => ({
    data: [
      { id: 'm1', email: 'alice@example.com', role: 'admin' },
      { id: 'm2', email: 'bob@example.com', role: 'member' },
    ],
    isLoading: false,
  }),
}))

vi.mock('../hooks/useBilling', () => ({
  useBilling: () => ({
    data: { plan: 'pro', renews: '2026-12-31' },
    isLoading: false,
  }),
  useUsage: () => ({
    data: { functions: 12, storage_mb: 480, sites: 4 },
    isLoading: false,
  }),
}))

describe('SettingsPage', () => {
  it('renders the page header', () => {
    render(<SettingsPage />)
    expect(screen.getByRole('heading', { name: 'Settings' })).toBeInTheDocument()
  })

  it('renders all three tabs as buttons', () => {
    render(<SettingsPage />)
    expect(screen.getByRole('button', { name: 'Account' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Workspace' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Billing' })).toBeInTheDocument()
  })

  it('shows the Account tab content by default', () => {
    render(<SettingsPage />)
    expect(screen.getByText('owner@example.com')).toBeInTheDocument()
  })

  it('switches to the Workspace tab and shows members', () => {
    render(<SettingsPage />)
    fireEvent.click(screen.getByRole('button', { name: 'Workspace' }))
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
  })

  it('switches to the Billing tab and shows plan + usage', () => {
    render(<SettingsPage />)
    fireEvent.click(screen.getByRole('button', { name: 'Billing' }))
    expect(screen.getByText(/pro/i)).toBeInTheDocument()
    // Functions usage (12), not the renews date (2026-12-31)
    const functionsCell = screen.getByText('Functions').parentElement
    expect(functionsCell).toHaveTextContent('12')
  })
})
