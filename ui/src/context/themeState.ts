import { createContext } from 'react'
import type { AccentId } from './accentThemes'

export type Theme = 'light' | 'dark'

export interface ThemeCtx {
  theme: Theme
  toggle: () => void
  accent: AccentId
  setAccent: (accent: AccentId) => void
}

export const ThemeContext = createContext<ThemeCtx>({
  theme: 'light',
  toggle: () => {},
  accent: 'default',
  setAccent: () => {},
})
