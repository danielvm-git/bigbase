import { describe, it, expect } from 'vitest'
import { TOKENS, token, cssVar } from './tokens'

describe('TOKENS', () => {
  it('has neutral palette', () => {
    expect(TOKENS.colors.neutral[0]).toBe('rgba(255, 255, 255, 1)')
    expect(TOKENS.colors.neutral[900]).toBe('rgba(25, 25, 28, 1)')
  })

  it('has brand palette', () => {
    expect(TOKENS.colors.brand[500]).toBe('rgba(79, 70, 229, 1)')
  })

  it('has semantic colors', () => {
    expect(TOKENS.colors.success).toBe('rgba(16, 185, 129, 1)')
    expect(TOKENS.colors.error).toBe('rgba(239, 68, 68, 1)')
  })

  it('has spacing tokens', () => {
    expect(TOKENS.spacing[4]).toBe('8px')
    expect(TOKENS.spacing[8]).toBe('16px')
  })

  it('has radius tokens', () => {
    expect(TOKENS.radius.s).toBe('8px')
    expect(TOKENS.radius.full).toBe('9999px')
  })

  it('has typography tokens', () => {
    expect(TOKENS.typography.textM).toBe('16px')
    expect(TOKENS.typography.textS).toBe('14px')
  })

  it('has motion tokens', () => {
    expect(TOKENS.motion.durationFast).toBe('150ms')
    expect(TOKENS.motion.easeStandard).toBe('ease')
  })

  it('has shadow tokens', () => {
    expect(TOKENS.shadow.xs).toContain('rgba(0, 0, 0')
  })

  it('has focus ring', () => {
    expect(TOKENS.focusRing).toContain('79, 70, 229')
  })
})

describe('cssVar', () => {
  it('wraps a CSS variable name', () => {
    expect(cssVar('--bg-accent')).toBe('var(--bg-accent)')
  })

  it('adds -- prefix if missing', () => {
    expect(cssVar('bg-accent')).toBe('var(--bg-accent)')
  })
})

describe('token', () => {
  it('accesses nested token values', () => {
    expect(token(t => t.colors.brand[500])).toBe('rgba(79, 70, 229, 1)')
    expect(token(t => t.radius.s)).toBe('8px')
  })
})
