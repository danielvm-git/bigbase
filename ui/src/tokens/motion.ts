import { useState, useEffect } from 'react'

export const MOTION_TOKENS = {
  durationFast: 150,
  durationShort: 160,
  durationMedium: 200,
  durationExtended: 250,
  durationSlow: 300,
  easeStandard: 'ease' as const,
  easeEmphasized: 'cubic-bezier(0.32, 0.72, 0, 1)' as const,
  easeInOut: 'ease-in-out' as const,
  easeOut: 'ease-out' as const,
} as const

/** Returns true when the user prefers reduced motion. Updates reactively. */
export function usePrefersReducedMotion(): boolean {
  const query = '(prefers-reduced-motion: reduce)'
  const [prefersReduced, setPrefersReduced] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(query).matches
  })

  useEffect(() => {
    if (typeof window === 'undefined') return
    const mql = window.matchMedia(query)
    const handler = (e: MediaQueryListEvent) => setPrefersReduced(e.matches)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [])

  return prefersReduced
}
