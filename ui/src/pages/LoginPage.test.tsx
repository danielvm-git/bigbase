import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import LoginPage from './LoginPage'
import { AuthProvider } from '../context/AuthContext'
import { useAuthSession } from '../hooks/useAuthSession'
import { PENDING_ROUTE_KEY } from '../context/authState'

function makeToken(expSeconds: number): string {
  const base64 = btoa(JSON.stringify({ exp: expSeconds, iat: expSeconds - 600 }))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
  return `header.${base64}.signature`
}

const EXP_S = Math.floor(Date.now() / 1000) + 600

describe('LoginPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    sessionStorage.clear()
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 404,
    } as Response)
  })

  afterEach(() => {
    sessionStorage.clear()
  })

  it('shows validation errors for empty sign-in fields', async () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Sign In' }))
    await waitFor(() => {
      expect(screen.getByText('Email is required')).toBeInTheDocument()
      expect(screen.getByText('Password is required')).toBeInTheDocument()
    })
  })

  it('shows forgot password flow', async () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Forgot password?' }))
    expect(screen.getByText('Reset password')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Send reset link' }))
    await waitFor(() => {
      expect(screen.getByText('Email is required')).toBeInTheDocument()
    })
  })

  it('shows confirmation after valid forgot-password submit', async () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Forgot password?' }))
    fireEvent.change(screen.getByPlaceholderText('Email'), {
      target: { value: 'user@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send reset link' }))
    await waitFor(() => {
      expect(screen.getByText(/you will receive an email shortly/i)).toBeInTheDocument()
    })
  })
})

describe('LoginPage re-authentication (e87s05)', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    sessionStorage.clear()
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 404,
    } as Response)
  })

  afterEach(() => {
    sessionStorage.clear()
  })
  /** Renders LoginPage at /login with a route probe for destinations. */
  function RouteProbe() {
    const { remainingMs } = useAuthSession()
    const location = useLocation()
    return (
      <>
        <span data-testid="destination">{location.pathname + location.search}</span>
        <span data-testid="session-remaining">{remainingMs}</span>
      </>
    )
  }

  function renderWithAuth(options?: {
    state?: { sessionExpired?: boolean }
    pendingRoute?: string
  }) {
    if (options?.pendingRoute) {
      sessionStorage.setItem(PENDING_ROUTE_KEY, options.pendingRoute)
    }
    return render(
      <AuthProvider>
        <MemoryRouter
          initialEntries={[
            options?.state
              ? { pathname: '/login', state: options.state }
              : '/login',
          ]}
        >
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="*"
              element={
                <>
                  <RouteProbe />
                </>
              }
            />
          </Routes>
        </MemoryRouter>
      </AuthProvider>,
    )
  }

  it('shows a notice when redirected with sessionExpired state', () => {
    renderWithAuth({ state: { sessionExpired: true } })
    expect(screen.getByText(/Your session expired/i)).toBeInTheDocument()
  })

  it('shows a notice when a pending route is waiting for restoration', () => {
    renderWithAuth({ pendingRoute: '/deploy/s1?tab=config' })
    expect(screen.getByText(/Your session expired/i)).toBeInTheDocument()
    // Notice does not consume the pending route — it must survive until login
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBe('/deploy/s1?tab=config')
  })

  it('restores the saved route and clears it after successful login', async () => {
    sessionStorage.setItem(PENDING_ROUTE_KEY, '/deploy/s1?tab=config')
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => ({
        token: makeToken(EXP_S),
        refresh_token: 'rt-login',
        expires_at: new Date(EXP_S * 1000).toISOString(),
      }),
    } as unknown as Response)

    renderWithAuth()

    fireEvent.change(screen.getByPlaceholderText('Email'), {
      target: { value: 'user@example.com' },
    })
    fireEvent.change(screen.getByPlaceholderText('Password'), {
      target: { value: 'password123' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    await waitFor(() => {
      expect(screen.getByTestId('destination')).toHaveTextContent('/deploy/s1?tab=config')
    })
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBeNull()
    // Auth context received the session from the login response
    expect(screen.getByTestId('session-remaining').textContent).not.toBe(
      String(Number.MAX_SAFE_INTEGER),
    )
  })

  it('navigates to the dashboard when no route is pending', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => ({
        token: makeToken(EXP_S),
        refresh_token: 'rt-login',
        expires_at: new Date(EXP_S * 1000).toISOString(),
      }),
    } as unknown as Response)

    renderWithAuth()

    fireEvent.change(screen.getByPlaceholderText('Email'), {
      target: { value: 'user@example.com' },
    })
    fireEvent.change(screen.getByPlaceholderText('Password'), {
      target: { value: 'password123' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    await waitFor(() => {
      expect(screen.getByTestId('destination')).toHaveTextContent('/')
    })
  })
})
