import type { HTMLAttributes } from 'react'

type BadgeVariant = 'neutral' | 'success' | 'warning' | 'error' | 'accent' | 'info'

/** Mirrors prototype react-stubs/tokens.ts StatusKind. */
export type StatusKind = 'ready' | 'building' | 'failed' | 'pending'

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant
}

const variantClass: Record<BadgeVariant, string> = {
  neutral: 'badge-neutral',
  success: 'badge-success',
  warning: 'badge-warning',
  error: 'badge-error',
  accent: 'badge-accent',
  info: 'badge-info',
}

/** Distinct non-color shape cue per variant; decorative (aria-hidden), the text stays the accessible cue. */
const variantGlyph: Record<BadgeVariant, string> = {
  neutral: '•',
  success: '✓',
  warning: '!',
  error: '✕',
  accent: '★',
  info: 'i',
}

export function Badge({ variant = 'neutral', className = '', children, ...rest }: BadgeProps) {
  return (
    <span className={`badge ${variantClass[variant]} ${className}`} {...rest}>
      <span aria-hidden="true" className="badge-indicator">{variantGlyph[variant]}</span>
      {children}
    </span>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function statusBadgeVariant(status: string): BadgeVariant {
  const s = status.toLowerCase()
  if (s === 'running' || s === 'active' || s === 'ok' || s === 'healthy') return 'success'
  if (s === 'failed' || s === 'error' || s === 'deleted') return 'error'
  if (s === 'building' || s === 'pending' || s === 'deploying') return 'warning'
  if (s === 'draining' || s === 'deploying') return 'warning'
  if (s === 'rolled_back' || s === 'replaced' || s === 'stopped') return 'neutral'
  return 'neutral'
}
