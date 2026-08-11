import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider, applyAccentToDocument } from './ThemeContext'
import { useTheme } from '../hooks/useTheme'
import { ACCENT_THEMES } from './accentThemes'

// WCAG 2.x relative luminance + contrast ratio (matches e88s01-contrast-matrix.mjs).
function channel(c: number): number {
  const s = c / 255
  return s <= 0.04045 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
}
function luminance([r, g, b]: [number, number, number]): number {
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}
function contrast(a: [number, number, number], b: [number, number, number]): number {
  const la = luminance(a)
  const lb = luminance(b)
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05)
}
function rgbFromVar(value: string): [number, number, number] {
  const m = value.match(/rgb\((\d+), (\d+), (\d+)\)/)
  if (!m) throw new Error(`unparseable rgb() value: ${value}`)
  return [Number(m[1]), Number(m[2]), Number(m[3])]
}

const WHITE: [number, number, number] = [255, 255, 255]
const NEUTRAL_850: [number, number, number] = [29, 29, 33]

function ThemeProbe() {
  const { accent, setAccent } = useTheme()
  return (
    <div>
      <span data-testid="accent">{accent}</span>
      <button type="button" onClick={() => setAccent('march')}>March</button>
    </div>
  )
}

describe('ThemeContext accent', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-accent-rainbow')
    document.documentElement.removeAttribute('data-theme')
  })

  it('persists accent to localStorage', () => {
    render(
      <ThemeProvider>
        <ThemeProbe />
      </ThemeProvider>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'March' }))
    expect(screen.getByTestId('accent').textContent).toBe('march')
    expect(localStorage.getItem('bigbase-accent')).toBe('march')
  })

  it('applies brand CSS variables on documentElement', () => {
    applyAccentToDocument('october')
    expect(document.documentElement.style.getPropertyValue('--brand-500')).toBe('rgb(236, 72, 153)')
    // Light mode (no data-theme): accent text uses the per-theme brandLink step (>=7:1 on white).
    expect(document.documentElement.style.getPropertyValue('--fg-accent')).toBe('rgb(157, 23, 77)')
    expect(document.documentElement.style.getPropertyValue('--bg-accent')).toBe('rgb(219, 39, 119)')
  })

  it('sets rainbow attribute for June theme', () => {
    applyAccentToDocument('june')
    expect(document.documentElement.getAttribute('data-accent-rainbow')).toBe('true')
  })

  it('applies per-theme brandLink light-mode links with >=7:1 on white (1.4.6)', () => {
    for (const theme of ACCENT_THEMES) {
      // Light mode: no data-theme attribute.
      document.documentElement.removeAttribute('data-theme')
      applyAccentToDocument(theme.id)
      const link = rgbFromVar(document.documentElement.style.getPropertyValue('--fg-accent'))
      const ratio = contrast(link, WHITE)
      expect(ratio, `${theme.id} light link ${link.join(',')} on white`).toBeGreaterThanOrEqual(7)
    }
  })

  it('applies per-theme brand300 dark-mode links with >=7:1 on neutral-850 (1.4.6)', () => {
    for (const theme of ACCENT_THEMES) {
      document.documentElement.setAttribute('data-theme', 'dark')
      applyAccentToDocument(theme.id)
      const link = rgbFromVar(document.documentElement.style.getPropertyValue('--fg-accent'))
      const ratio = contrast(link, NEUTRAL_850)
      expect(ratio, `${theme.id} dark link ${link.join(',')} on neutral-850`).toBeGreaterThanOrEqual(7)
    }
  })
})
