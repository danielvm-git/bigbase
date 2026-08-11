import type { CSSProperties, ReactNode } from 'react'
import { Card, CardHeader } from './index'

interface MetricCardProps {
  label: string
  value: string | number
  subtitle?: string
  trend?: 'up' | 'down' | 'flat'
  color?: 'success' | 'warning' | 'error' | 'neutral'
  secondaryValue?: string
}

const trendArrows: Record<string, string> = { up: '↑', down: '↓', flat: '→' }

/** Textual counterpart of each arrow so trend is never conveyed by glyph shape alone. */
const trendLabels: Record<string, string> = { up: 'trending up', down: 'trending down', flat: 'trend flat' }

const colorVar = (c?: string): string => {
  switch (c) {
    case 'success': return 'var(--success-fg)'
    case 'warning': return 'var(--warning-fg)'
    case 'error': return 'var(--error-fg)'
    default: return 'var(--fg-primary)'
  }
}

/** Visually hidden but exposed to assistive technology (WCAG 1.4.1). */
const srOnly: CSSProperties = {
  position: 'absolute',
  width: '1px',
  height: '1px',
  padding: '0',
  margin: '-1px',
  overflow: 'hidden',
  clip: 'rect(0 0 0 0)',
  clipPath: 'inset(50%)',
  whiteSpace: 'nowrap',
  border: '0',
}

export function MetricCard({ label, value, subtitle, trend, color, secondaryValue }: MetricCardProps): ReactNode {
  return (
    <Card>
      <CardHeader title={label} />
      <p className="stat-value" style={{ color: colorVar(color) }}>
        {trend && (
          <span style={{ marginRight: 'var(--space-2)', fontSize: '0.8em' }}>
            <span style={srOnly}>{trendLabels[trend]}</span>
            <span aria-hidden="true">{trendArrows[trend]}</span>
          </span>
        )}
        {color && color !== 'neutral' && <span style={srOnly}>{color}</span>}
        {value}
      </p>
      {secondaryValue && <p className="stat-value" style={{ fontSize: 'var(--text-s)', color: 'var(--fg-secondary)' }}>{secondaryValue}</p>}
      {subtitle && <p className="dim">{subtitle}</p>}
    </Card>
  )
}
