import { useCallback, useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Dialog } from './Dialog'
import { useAuthSession } from '../hooks/useAuthSession'
import { useToast } from '../hooks/useToast'
import { fmtCountdown } from '../lib/countdown'

const DEFAULT_THRESHOLD_MS = 5 * 60 * 1000
const UNAUTHORIZED_EVENT = 'bigbase:unauthorized'

/**
 * Session timeout warning (e87s05, WCAG 2.2.6 / 2.2.5).
 *
 * Shows a dialog with a live countdown once the access token is within
 * `thresholdMs` of expiring. "Stay signed in" refreshes the token via
 * POST /api/auth/refresh; dismissal hides the warning until the session is
 * renewed. If the session expires (countdown reaches zero, a refresh fails,
 * or any API call answers 401), the current route is saved to sessionStorage
 * and the user is redirected to /login for graceful re-authentication.
 */
export default function SessionTimeoutWarning({
  thresholdMs = DEFAULT_THRESHOLD_MS,
}: {
  thresholdMs?: number
}) {
  const {
    remainingMs,
    expiresAt,
    isAuthenticated,
    refresh,
    clearSession,
    savePendingRoute,
  } = useAuthSession()
  const { show } = useToast()
  const location = useLocation()
  const nav = useNavigate()
  const [refreshing, setRefreshing] = useState(false)
  const [dismissedFor, setDismissedFor] = useState<number | null>(null)
  const wasVisible = useRef(false)

  const onLoginPage = location.pathname.startsWith('/login')
  const showDialog =
    isAuthenticated &&
    !onLoginPage &&
    expiresAt !== dismissedFor &&
    remainingMs < thresholdMs

  const handleUnauthorized = useCallback(() => {
    if (location.pathname.startsWith('/login')) return
    savePendingRoute(location.pathname + location.search)
    clearSession()
    // Deferred so a sibling handler's plain redirect (e.g. Layout's
    // /api/auth/me catch) cannot clobber the state-carrying navigation.
    setTimeout(() => {
      nav('/login', { state: { sessionExpired: true } })
    }, 0)
  }, [location.pathname, location.search, savePendingRoute, clearSession, nav])

  // Global 401 → graceful re-auth (interceptor lives in AuthProvider).
  useEffect(() => {
    const on401 = () => handleUnauthorized()
    window.addEventListener(UNAUTHORIZED_EVENT, on401)
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, on401)
  }, [handleUnauthorized])

  // Countdown reached zero without a refresh → full re-authentication.
  useEffect(() => {
    if (showDialog && remainingMs <= 0) {
      handleUnauthorized()
    }
  }, [showDialog, remainingMs, handleUnauthorized])

  // Announce the warning once per dialog appearance.
  useEffect(() => {
    if (showDialog && !wasVisible.current) {
      show(
        `Your session expires in ${fmtCountdown(remainingMs)}. Choose "Stay signed in" to continue working.`,
        'info',
      )
    }
    wasVisible.current = showDialog
  }, [showDialog, remainingMs, show])

  const handleStaySignedIn = async () => {
    setRefreshing(true)
    const result = await refresh()
    setRefreshing(false)
    if (result.ok) {
      show('Session extended.', 'success')
    } else if (result.reason === 'network') {
      show('Could not reach the server. Check your connection.', 'error')
    } else {
      // Refresh token missing or invalid → fall back to full re-auth.
      handleUnauthorized()
    }
  }

  if (!showDialog) return null

  return (
    <Dialog
      open={showDialog}
      title="Session expiring soon"
      onClose={() => setDismissedFor(expiresAt)}
      onConfirm={handleStaySignedIn}
      confirmLabel="Stay signed in"
      cancelLabel="Dismiss"
      loading={refreshing}
    >
      <p>
        Your session expires in <strong>{fmtCountdown(remainingMs)}</strong>. Choose
        &ldquo;Stay signed in&rdquo; to continue working, or sign in again after the
        session ends. Your work in progress will be preserved.
      </p>
    </Dialog>
  )
}
