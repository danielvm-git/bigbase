import type { ReactNode } from 'react'
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

const colorVar = (c?: string): string => {
  switch (c) {
    case 'success': return 'var(--success-fg)'
    case 'warning': return 'var(--warning-fg)'
    case 'error': return 'var(--error-fg)'
    default: return 'var(--fg-primary)'
  }
}

export function MetricCard({ label, value, subtitle, trend, color, secondaryValue }: MetricCardProps): ReactNode {
  return (
    <Card>
      <CardHeader title={label} />
      <p className="stat-value" style={{ color: colorVar(color) }}>
        {trend && <span style={{ marginRight: 'var(--space-2)', fontSize: '0.8em' }}>{trendArrows[trend]}</span>}
        {value}
      </p>
      {secondaryValue && <p className="stat-value" style={{ fontSize: 'var(--text-s)', color: 'var(--fg-secondary)' }}>{secondaryValue}</p>}
      {subtitle && <p className="dim">{subtitle}</p>}
    </Card>
  )
}
