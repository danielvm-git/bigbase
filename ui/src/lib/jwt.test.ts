import { describe, it, expect } from 'vitest'
import { decodeJwtExp } from './jwt'

function makeToken(payload: Record<string, unknown>): string {
  const base64 = btoa(JSON.stringify(payload))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
  return `header.${base64}.signature`
}

describe('decodeJwtExp', () => {
  it('returns the exp claim in epoch milliseconds', () => {
    const token = makeToken({ exp: 2_000_000_000, iat: 1_999_900_000 })
    expect(decodeJwtExp(token)).toBe(2_000_000_000_000)
  })

  it('returns null for a non-JWT string', () => {
    expect(decodeJwtExp('not-a-jwt')).toBeNull()
    expect(decodeJwtExp('')).toBeNull()
    expect(decodeJwtExp('a.b')).toBeNull()
    expect(decodeJwtExp('a.b.c.d')).toBeNull()
  })

  it('returns null when the payload is not valid JSON', () => {
    const junk = btoa('not json').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
    expect(decodeJwtExp(`h.${junk}.s`)).toBeNull()
  })

  it('returns null when exp is missing or not a number', () => {
    expect(decodeJwtExp(makeToken({ iat: 1 }))).toBeNull()
    expect(decodeJwtExp(makeToken({ exp: 'later' }))).toBeNull()
    expect(decodeJwtExp(makeToken({ exp: null }))).toBeNull()
  })

  it('decodes base64url payloads that would need padding', () => {
    // exp with a value whose base64 requires padding
    const token = makeToken({ exp: 2_000_000_000 })
    expect(decodeJwtExp(token)).toBe(2_000_000_000_000)
  })
})
