import { useContext } from 'react'
import { ThemeContext } from '../context/themeState'

export function useTheme() {
  return useContext(ThemeContext)
}
