import { createContext } from 'react'

/** sessionStorage key for the route to restore after re-authentication.
 *  Only pathname+search is ever stored here — never tokens. */
export const PENDING_ROUTE_KEY = 'bigbase.pendingRoute'

/** remainingMs value reported when there is no active session. */
export const NO_SESSION_REMAINING = Number.MAX_SAFE_INTEGER

export interface AuthSessionInput {
  /** Raw access token — its `exp` claim is decoded for the expiry. */
  token?: string | null
  /** Refresh token, kept in memory only (never persisted). */
  refreshToken?: string | null
  /** Explicit expiry (epoch ms) fallback when no token is available. */
  expiresAtMs?: number | null
}

export type RefreshResult =
  | { ok: true; expiresAtMs: number | null }
  | { ok: false; reason: 'no-refresh-token' | 'expired' | 'network' }

export interface AuthContextValue {
  /** Epoch ms when the current access token expires, or null. */
  expiresAt: number | null
  /** Milliseconds until expiry, ticking once per second. */
  remainingMs: number
  isAuthenticated: boolean
  setSession: (session: AuthSessionInput) => void
  clearSession: () => void
  /** POST /api/auth/refresh with the in-memory refresh token. */
  refresh: () => Promise<RefreshResult>
  /** Store a route (pathname + search) for restoration after login. */
  savePendingRoute: (route: string) => void
  /** Read + clear the pending route (cleared immediately on read). */
  restorePendingRoute: () => string | null
  /** Read the pending route without clearing it. */
  peekPendingRoute: () => string | null
}

export const AuthContext = createContext<AuthContextValue>({
  expiresAt: null,
  remainingMs: NO_SESSION_REMAINING,
  isAuthenticated: false,
  setSession: () => {},
  clearSession: () => {},
  refresh: async () => ({ ok: false, reason: 'no-refresh-token' }),
  savePendingRoute: () => {},
  restorePendingRoute: () => null,
  peekPendingRoute: () => null,
})
