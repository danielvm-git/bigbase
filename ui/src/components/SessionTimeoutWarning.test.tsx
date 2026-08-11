import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, act, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import SessionTimeoutWarning, { fmtCountdown } from './SessionTimeoutWarning'
import { AuthProvider } from '../context/AuthContext'
import { useAuthSession } from '../hooks/useAuthSession'
import { ToastProvider } from '../context/ToastContext'
import { PENDING_ROUTE_KEY } from '../context/authState'
import LoginPage from '../pages/LoginPage'

function makeToken(expSeconds: number): string {
  const base64 = btoa(JSON.stringify({ exp: expSeconds, iat: expSeconds - 600 }))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
  return `header.${base64}.signature`
}

/** Mirrors the real App: the warning sits outside <Routes>, login is a route. */
function Harness({
  expInMs = 10 * 60 * 1000,
  thresholdMs,
}: {
  expInMs?: number
  thresholdMs?: number
}) {
  const { setSession } = useAuthSession()
  return (
    <div>
      <button
        type="button"
        onClick={() =>
          setSession({
            token: makeToken(Math.floor((Date.now() + expInMs) / 1000)),
            refreshToken: 'rt-test',
          })
        }
      >
        start-session
      </button>
      <SessionTimeoutWarning thresholdMs={thresholdMs} />
    </div>
  )
}

function renderHarness(options?: {
  expInMs?: number
  initialEntries?: string[]
  thresholdMs?: number
}) {
  return render(
    <AuthProvider>
      <ToastProvider>
        <MemoryRouter initialEntries={options?.initialEntries ?? ['/deploy']}>
          <Harness expInMs={options?.expInMs} thresholdMs={options?.thresholdMs} />
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="*" element={<div data-testid="app-page">app page</div>} />
          </Routes>
        </MemoryRouter>
      </ToastProvider>
    </AuthProvider>,
  )
}

function startSession() {
  act(() => {
    fireEvent.click(screen.getByRole('button', { name: 'start-session' }))
  })
}

beforeEach(() => {
  sessionStorage.clear()
  vi.useFakeTimers()
  vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    status: 404,
    ok: false,
    json: async () => ({}),
  } as unknown as Response)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
  sessionStorage.clear()
})

describe('fmtCountdown', () => {
  it('formats mm:ss rounding up', () => {
    expect(fmtCountdown(300_000)).toBe('5:00')
    expect(fmtCountdown(299_999)).toBe('5:00')
    expect(fmtCountdown(65_000)).toBe('1:05')
    expect(fmtCountdown(0)).toBe('0:00')
    expect(fmtCountdown(-10)).toBe('0:00')
  })
})

describe('SessionTimeoutWarning', () => {
  it('shows the dialog with a live countdown when remaining drops below the threshold', () => {
    renderHarness()
    startSession()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    // 10 min session; advance past the 5 min threshold → 4:59 remaining.
    act(() => vi.advanceTimersByTime(5 * 60 * 1000 + 1000))
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(screen.getByText('Session expiring soon')).toBeInTheDocument()
    expect(screen.getByText('4:59')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Stay signed in' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Dismiss' })).toBeInTheDocument()
    // Announcement toast fires once when the dialog first appears
    expect(
      screen.getByText(/Your session expires in 4:59\. Choose "Stay signed in" to continue working\.$/),
    ).toBeInTheDocument()

    // Countdown keeps ticking
    act(() => vi.advanceTimersByTime(60_000))
    expect(screen.getByText('3:59')).toBeInTheDocument()
  })

  it('"Stay signed in" refreshes the token and closes the dialog', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => {
        // Expiry relative to the (faked) moment the refresh runs, so the new
        // session lands comfortably above the warning threshold again.
        const newExp = Math.floor(Date.now() / 1000) + 600
        return {
          token: makeToken(newExp),
          refresh_token: 'rt-new',
          expires_at: new Date(newExp * 1000).toISOString(),
        }
      },
    } as unknown as Response)

    renderHarness()
    startSession()
    act(() => vi.advanceTimersByTime(5 * 60 * 1000 + 1000))
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Stay signed in' }))
    })
    await act(async () => {})

    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/auth/refresh',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ refresh_token: 'rt-test' }),
      }),
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByText('Session extended.')).toBeInTheDocument()
  })

  it('dismiss keeps the dialog closed for the current session', () => {
    renderHarness()
    startSession()
    act(() => vi.advanceTimersByTime(5 * 60 * 1000 + 1000))
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    act(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    act(() => vi.advanceTimersByTime(60_000))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('expiry saves the route and redirects to login with a session-expired notice', () => {
    renderHarness({ expInMs: 30_000 })
    startSession()
    act(() => vi.advanceTimersByTime(35_000))
    // Route (pathname + search) saved before the redirect
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBe('/deploy')
    // Deferred navigation flushed
    act(() => vi.runOnlyPendingTimers())
    expect(screen.getByText(/Your session expired/i)).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('falls back to full re-auth when the refresh request answers 401', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 401,
      ok: false,
      json: async () => ({ error: 'refresh token expired' }),
    } as unknown as Response)

    renderHarness()
    startSession()
    act(() => vi.advanceTimersByTime(5 * 60 * 1000 + 1000))
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Stay signed in' }))
    })
    await act(async () => {})
    act(() => vi.runOnlyPendingTimers())

    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBe('/deploy')
    expect(screen.getByText(/Your session expired/i)).toBeInTheDocument()
  })

  it('any API 401 (outside /api/auth/*) triggers the graceful re-auth flow', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 401,
      ok: false,
      json: async () => ({ error: 'unauthorized' }),
    } as unknown as Response)

    renderHarness()
    startSession()

    // An in-flight API call answers 401; the AuthProvider interceptor wraps
    // fetch and fires the unauthorized event.
    await act(async () => {
      await window.fetch('/api/sites')
    })
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBe('/deploy')

    act(() => vi.runOnlyPendingTimers())
    expect(screen.getByText(/Your session expired/i)).toBeInTheDocument()
  })

  it('does not warn while on the login page', () => {
    renderHarness({ initialEntries: ['/login'] })
    startSession()
    act(() => vi.advanceTimersByTime(5 * 60 * 1000 + 1000))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
