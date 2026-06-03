import { useEffect, useState, type ReactNode } from 'react'
import { ThemeContext } from './themeState'
import type { Theme } from './themeState'
import {
  type AccentId,
  getAccentTheme,
  isAccentId,
} from './accentThemes'

const THEME_KEY = 'bigbase-theme'
const ACCENT_KEY = 'bigbase-accent'

function getStoredTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_KEY)
    if (stored === 'dark' || stored === 'light') return stored
  } catch { /* localStorage unavailable */ }
  return 'light'
}

function getStoredAccent(): AccentId {
  try {
    const stored = localStorage.getItem(ACCENT_KEY)
    if (stored && isAccentId(stored)) return stored
  } catch { /* localStorage unavailable */ }
  return 'default'
}

export function applyAccentToDocument(accent: AccentId) {
  const root = document.documentElement
  const theme = getAccentTheme(accent)

  if (theme.rainbow) {
    root.setAttribute('data-accent-rainbow', 'true')
  } else {
    root.removeAttribute('data-accent-rainbow')
  }

  root.style.setProperty('--brand-500', `rgb(${theme.brand500})`)
  root.style.setProperty('--brand-600', `rgb(${theme.brand600})`)
  root.style.setProperty('--brand-700', `rgb(${theme.brand700})`)
  root.style.setProperty('--border-accent', `rgb(${theme.brand500})`)
  root.style.setProperty('--bg-accent', `rgb(${theme.brand500})`)
  root.style.setProperty('--bg-accent-hover', `rgb(${theme.brand600})`)
  root.style.setProperty('--bg-accent-active', `rgb(${theme.brand700})`)
  root.style.setProperty('--fg-accent', `rgb(${theme.brand500})`)
  root.style.setProperty('--brand-tint', `rgba(${theme.brand500}, 0.10)`)
  root.style.setProperty('--focus-ring', `0 0 0 3px rgba(${theme.brand500}, 0.18)`)
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(getStoredTheme)
  const [accent, setAccentState] = useState<AccentId>(getStoredAccent)

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    try { localStorage.setItem(THEME_KEY, theme) } catch { /* noop */ }
  }, [theme])

  useEffect(() => {
    applyAccentToDocument(accent)
    try { localStorage.setItem(ACCENT_KEY, accent) } catch { /* noop */ }
  }, [accent])

  const toggle = () => setTheme(prev => (prev === 'light' ? 'dark' : 'light'))
  const setAccent = (next: AccentId) => setAccentState(next)

  return (
    <ThemeContext.Provider value={{ theme, toggle, accent, setAccent }}>
      {children}
    </ThemeContext.Provider>
  )
}
