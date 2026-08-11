/**
 * Minimal JWT payload decoding for the session-timeout warning (e87s05).
 *
 * Only the payload's `exp` claim is read. JWT payloads are base64url-encoded
 * JSON and are public data (the token is signed, not encrypted), so decoding
 * client-side is safe. The auth context uses this to compute how much time
 * remains before the access token expires.
 */

function decodeBase64Url(segment: string): string {
  const base64 = segment.replace(/-/g, '+').replace(/_/g, '/')
  const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
  const binary = globalThis.atob(padded)
  const bytes = Uint8Array.from(binary, c => c.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

/**
 * Returns the JWT `exp` claim as epoch **milliseconds**, or `null` when the
 * token is malformed or carries no numeric `exp`.
 */
export function decodeJwtExp(token: string): number | null {
  if (typeof token !== 'string' || token === '') return null
  const parts = token.split('.')
  if (parts.length !== 3) return null
  try {
    const payload = JSON.parse(decodeBase64Url(parts[1])) as { exp?: unknown }
    if (typeof payload.exp === 'number' && Number.isFinite(payload.exp)) {
      return payload.exp * 1000
    }
  } catch {
    return null
  }
  return null
}
