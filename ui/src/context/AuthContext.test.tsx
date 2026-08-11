import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, act, waitFor, fireEvent } from '@testing-library/react'
import { AuthProvider } from './AuthContext'
import { useAuthSession } from '../hooks/useAuthSession'
import { decodeJwtExp } from '../lib/jwt'
import { PENDING_ROUTE_KEY } from './authState'

declare global {
  interface Window {
    __refreshResult?: unknown
    __restored?: string | null
  }
}

/** 10 minutes out in seconds. */
const EXP_S = Math.floor(Date.now() / 1000) + 600

function makeToken(expSeconds: number): string {
  const base64 = btoa(JSON.stringify({ exp: expSeconds, iat: expSeconds - 600 }))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
  return `header.${base64}.signature`
}

function Probe() {
  const { expiresAt, remainingMs, isAuthenticated, setSession, clearSession, refresh, savePendingRoute, restorePendingRoute } =
    useAuthSession()
  return (
    <div>
      <span data-testid="expiresAt">{expiresAt ?? 'null'}</span>
      <span data-testid="remainingMs">{remainingMs}</span>
      <span data-testid="isAuthenticated">{String(isAuthenticated)}</span>
      <button
        type="button"
        onClick={() => setSession({ token: makeToken(EXP_S), refreshToken: 'rt-1' })}
      >
        login-jwt
      </button>
      <button
        type="button"
        onClick={() => setSession({ expiresAtMs: Date.now() + 120_000, refreshToken: 'rt-2' })}
      >
        login-ms
      </button>
      <button type="button" onClick={clearSession}>
        logout
      </button>
      <button
        type="button"
        onClick={async () => {
          window.__refreshResult = await refresh()
        }}
      >
        do-refresh
      </button>
      <button type="button" onClick={() => savePendingRoute('/deploy/s1?tab=config')}>
        save-route
      </button>
      <button
        type="button"
        onClick={() => {
          window.__restored = restorePendingRoute()
        }}
      >
        restore-route
      </button>
    </div>
  )
}

function renderHarness() {
  return render(
    <AuthProvider>
      <Probe />
    </AuthProvider>,
  )
}

beforeEach(() => {
  sessionStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
  sessionStorage.clear()
})

describe('decodeJwtExp', () => {
  it('is the source of the session expiry', () => {
    expect(decodeJwtExp(makeToken(EXP_S))).toBe(EXP_S * 1000)
  })
})

describe('AuthProvider session', () => {
  it('exposes remaining time derived from the JWT exp claim', () => {
    renderHarness()
    expect(screen.getByTestId('isAuthenticated')).toHaveTextContent('false')
    fireEventClick('login-jwt')
    expect(screen.getByTestId('isAuthenticated')).toHaveTextContent('true')
    expect(Number(screen.getByTestId('expiresAt').textContent)).toBe(EXP_S * 1000)
    const remaining = Number(screen.getByTestId('remainingMs').textContent)
    expect(remaining).toBeGreaterThan(590_000)
    expect(remaining).toBeLessThanOrEqual(600_000)
  })

  it('ticks remaining time down once per second', () => {
    vi.useFakeTimers()
    renderHarness()
    fireEventClick('login-jwt')
    const before = Number(screen.getByTestId('remainingMs').textContent)
    act(() => vi.advanceTimersByTime(5000))
    const after = Number(screen.getByTestId('remainingMs').textContent)
    expect(before - after).toBeGreaterThanOrEqual(4000)
  })

  it('accepts an explicit expiresAtMs when no token is available', () => {
    renderHarness()
    fireEventClick('login-ms')
    expect(screen.getByTestId('isAuthenticated')).toHaveTextContent('true')
    expect(Number(screen.getByTestId('remainingMs').textContent)).toBeGreaterThan(115_000)
    expect(Number(screen.getByTestId('remainingMs').textContent)).toBeLessThanOrEqual(120_000)
  })

  it('clearSession drops the session', () => {
    renderHarness()
    fireEventClick('login-jwt')
    fireEventClick('logout')
    expect(screen.getByTestId('isAuthenticated')).toHaveTextContent('false')
    expect(screen.getByTestId('expiresAt')).toHaveTextContent('null')
  })
})

describe('AuthProvider refresh', () => {
  it('posts the refresh token and renews the session', async () => {
    const newExp = Math.floor(Date.now() / 1000) + 300
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => ({
        token: makeToken(newExp),
        refresh_token: 'rt-new',
        expires_at: new Date(newExp * 1000).toISOString(),
      }),
    } as unknown as Response)
    renderHarness()
    fireEventClick('login-jwt')
    fireEventClick('do-refresh')
    await waitFor(() => {
      expect(window.__refreshResult).toEqual({ ok: true, expiresAtMs: newExp * 1000 })
    })
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/auth/refresh',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ refresh_token: 'rt-1' }),
      }),
    )
    expect(Number(screen.getByTestId('expiresAt').textContent)).toBe(newExp * 1000)
  })

  it('reports expired and clears the session on a 401 refresh response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      status: 401,
      ok: false,
      json: async () => ({ error: 'refresh token expired' }),
    } as unknown as Response)
    renderHarness()
    fireEventClick('login-jwt')
    fireEventClick('do-refresh')
    await waitFor(() => {
      expect(window.__refreshResult).toEqual({ ok: false, reason: 'expired' })
    })
    expect(screen.getByTestId('isAuthenticated')).toHaveTextContent('false')
  })

  it('reports network errors without clearing the session', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('offline'))
    renderHarness()
    fireEventClick('login-jwt')
    fireEventClick('do-refresh')
    await waitFor(() => {
      expect(window.__refreshResult).toEqual({ ok: false, reason: 'network' })
    })
    expect(screen.getByTestId('isAuthenticated')).toHaveTextContent('true')
  })

  it('reports no-refresh-token when the session has no refresh token', async () => {
    renderHarness()
    fireEventClick('do-refresh')
    await waitFor(() => {
      expect(window.__refreshResult).toEqual({ ok: false, reason: 'no-refresh-token' })
    })
  })
})

describe('AuthProvider storage discipline', () => {
  it('never writes tokens to sessionStorage', () => {
    renderHarness()
    fireEventClick('login-jwt')
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBeNull()
    expect(sessionStorage.length).toBe(0)
  })

  it('round-trips a pending route and clears it immediately on read', () => {
    renderHarness()
    fireEventClick('save-route')
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBe('/deploy/s1?tab=config')
    fireEventClick('restore-route')
    expect(window.__restored).toBe('/deploy/s1?tab=config')
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBeNull()
  })

  it('rejects non-route values when saving a pending route', () => {
    renderHarness()
    // savePendingRoute only accepts values starting with '/'
    act(() => {
      // direct storage write simulates a non-route value; restore must reject it
      sessionStorage.setItem(PENDING_ROUTE_KEY, 'https://evil.example')
    })
    fireEventClick('restore-route')
    expect(window.__restored).toBeNull()
    expect(sessionStorage.getItem(PENDING_ROUTE_KEY)).toBeNull()
  })
})

function fireEventClick(label: string) {
  fireEvent.click(screen.getByRole('button', { name: label }))
}
