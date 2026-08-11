import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import {
  AuthContext,
  NO_SESSION_REMAINING,
  PENDING_ROUTE_KEY,
  type AuthSessionInput,
  type RefreshResult,
} from './authState'
import { decodeJwtExp } from '../lib/jwt'

const TICK_MS = 1000
const UNAUTHORIZED_EVENT = 'bigbase:unauthorized'
const WRAPPED = Symbol('bigbase.wrappedFetch')

function isAuthEndpoint(url: string): boolean {
  return url.includes('/api/auth/')
}

/** Resolve expiry (epoch ms) from the JWT exp claim first, then fallbacks. */
function resolveExpiry(input: AuthSessionInput): number | null {
  if (input.token) {
    const fromJwt = decodeJwtExp(input.token)
    if (fromJwt !== null) return fromJwt
  }
  if (
    typeof input.expiresAtMs === 'number' &&
    Number.isFinite(input.expiresAtMs) &&
    input.expiresAtMs > 0
  ) {
    return input.expiresAtMs
  }
  return null
}

/**
 * Session provider (e87s05).
 *
 * Exposes the access-token expiry (decoded from the JWT `exp` claim) as a
 * ticking `remainingMs`, a `refresh()` action for the "Stay signed in" flow,
 * and pending-route helpers for graceful re-authentication. The refresh token
 * is held in memory only — it is never written to storage.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [expiresAt, setExpiresAt] = useState<number | null>(null)
  const [now, setNow] = useState(() => Date.now())
  const refreshTokenRef = useRef<string | null>(null)

  // One-second tick drives the countdown exposed to consumers.
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), TICK_MS)
    return () => clearInterval(id)
  }, [])

  const setSession = useCallback((session: AuthSessionInput) => {
    if (session.refreshToken != null) {
      refreshTokenRef.current = session.refreshToken
    }
    setExpiresAt(resolveExpiry(session))
    setNow(Date.now())
  }, [])

  const clearSession = useCallback(() => {
    refreshTokenRef.current = null
    setExpiresAt(null)
    setNow(Date.now())
  }, [])

  const refresh = useCallback(async (): Promise<RefreshResult> => {
    const rt = refreshTokenRef.current
    if (!rt) return { ok: false, reason: 'no-refresh-token' }
    try {
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      })
      if (!res.ok) {
        clearSession()
        return { ok: false, reason: 'expired' }
      }
      const data = (await res.json()) as {
        token?: string
        refresh_token?: string
        expires_at?: string
      }
      const parsedFallback = data.expires_at ? Date.parse(data.expires_at) : NaN
      const expiresAtMs =
        decodeJwtExp(data.token ?? '') ?? (Number.isFinite(parsedFallback) ? parsedFallback : null)
      refreshTokenRef.current = data.refresh_token ?? rt
      setExpiresAt(expiresAtMs)
      setNow(Date.now())
      return { ok: true, expiresAtMs }
    } catch {
      return { ok: false, reason: 'network' }
    }
  }, [clearSession])

  const savePendingRoute = useCallback((route: string) => {
    try {
      if (route && route.startsWith('/')) {
        sessionStorage.setItem(PENDING_ROUTE_KEY, route)
      }
    } catch {
      /* storage unavailable */
    }
  }, [])

  const restorePendingRoute = useCallback(() => {
    try {
      const route = sessionStorage.getItem(PENDING_ROUTE_KEY)
      sessionStorage.removeItem(PENDING_ROUTE_KEY)
      return route && route.startsWith('/') ? route : null
    } catch {
      return null
    }
  }, [])

  const peekPendingRoute = useCallback(() => {
    try {
      const route = sessionStorage.getItem(PENDING_ROUTE_KEY)
      return route && route.startsWith('/') ? route : null
    } catch {
      return null
    }
  }, [])

  // Global 401 interceptor: any API 401 outside /api/auth/* fires an event
  // that SessionTimeoutWarning turns into a graceful re-auth redirect.
  useEffect(() => {
    const win = window as Window & {
      fetch: typeof fetch & { [WRAPPED]?: boolean }
    }
    const original = win.fetch
    if (!original || original[WRAPPED]) return

    const wrapped = ((input: RequestInfo | URL, init?: RequestInit) => {
      const promise = original(input, init)
      const url =
        typeof input === 'string'
          ? input
          : input instanceof URL
            ? input.href
            : (input as Request).url
      promise
        .then(res => {
          if (res.status === 401 && !isAuthEndpoint(url)) {
            win.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT, { detail: { url } }))
          }
        })
        .catch(() => {})
      return promise
    }) as typeof fetch
    Object.defineProperty(wrapped, WRAPPED, { value: true })
    win.fetch = wrapped
    return () => {
      if (win.fetch === wrapped) win.fetch = original
    }
  }, [])

  const value = useMemo(
    () => ({
      expiresAt,
      remainingMs:
        expiresAt === null ? NO_SESSION_REMAINING : Math.max(0, expiresAt - now),
      isAuthenticated: expiresAt !== null,
      setSession,
      clearSession,
      refresh,
      savePendingRoute,
      restorePendingRoute,
      peekPendingRoute,
    }),
    [
      expiresAt,
      now,
      setSession,
      clearSession,
      refresh,
      savePendingRoute,
      restorePendingRoute,
      peekPendingRoute,
    ],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
