import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { MOTION_TOKENS, usePrefersReducedMotion } from './motion'

function mockMatchMedia(matches: boolean) {
  const listeners: Array<(e: MediaQueryListEvent) => void> = []
  const mql = {
    matches,
    addEventListener: vi.fn((_: string, fn: (e: MediaQueryListEvent) => void) => listeners.push(fn)),
    removeEventListener: vi.fn(),
  }
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockReturnValue(mql),
  })
  return { mql, listeners }
}

describe('MOTION_TOKENS', () => {
  it('has correct duration values', () => {
    expect(MOTION_TOKENS.durationFast).toBe(150)
    expect(MOTION_TOKENS.durationMedium).toBe(200)
    expect(MOTION_TOKENS.durationSlow).toBe(300)
  })

  it('has correct easing values', () => {
    expect(MOTION_TOKENS.easeStandard).toBe('ease')
    expect(MOTION_TOKENS.easeEmphasized).toBe('cubic-bezier(0.32, 0.72, 0, 1)')
  })
})

describe('usePrefersReducedMotion', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns false when prefers-reduced-motion does not match', () => {
    mockMatchMedia(false)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(false)
  })

  it('returns true when prefers-reduced-motion matches', () => {
    mockMatchMedia(true)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(true)
  })

  it('updates when media query changes', () => {
    const { listeners } = mockMatchMedia(false)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(false)

    act(() => {
      listeners.forEach(fn => fn({ matches: true } as MediaQueryListEvent))
    })

    expect(result.current).toBe(true)
  })

  it('removes event listener on unmount', () => {
    const { mql } = mockMatchMedia(false)
    const { unmount } = renderHook(() => usePrefersReducedMotion())
    unmount()
    expect(mql.removeEventListener).toHaveBeenCalled()
  })
})
