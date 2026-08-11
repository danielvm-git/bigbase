import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider, applyAccentToDocument } from './ThemeContext'
import { useTheme } from '../hooks/useTheme'

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
    // Light mode (no data-theme): accent text uses the 600 step for >=7:1 on white.
    expect(document.documentElement.style.getPropertyValue('--fg-accent')).toBe('rgb(219, 39, 119)')
    expect(document.documentElement.style.getPropertyValue('--bg-accent')).toBe('rgb(219, 39, 119)')
  })

  it('sets rainbow attribute for June theme', () => {
    applyAccentToDocument('june')
    expect(document.documentElement.getAttribute('data-accent-rainbow')).toBe('true')
  })
})
