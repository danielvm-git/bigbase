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
    expect(document.documentElement.style.getPropertyValue('--fg-accent')).toBe('rgb(236, 72, 153)')
  })

  it('sets rainbow attribute for June theme', () => {
    applyAccentToDocument('june')
    expect(document.documentElement.getAttribute('data-accent-rainbow')).toBe('true')
  })
})
