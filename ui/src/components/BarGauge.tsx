import type { ReactNode } from 'react'

interface BarGaugeProps {
  used: number
  total: number
  label?: string
  color?: string
  formatValue?: (n: number) => string
}

/**
 * A horizontal bar gauge showing used/total as a filled progress bar.
 */
export function BarGauge({ used, total, label, color = '#3b82f6', formatValue }: BarGaugeProps): ReactNode {
  const pct = total > 0 ? Math.min((used / total) * 100, 100) : 0
  const fmt = formatValue ?? ((n: number) => n.toString())

  return (
    <div style={{ width: '100%' }}>
      {label && (
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 'var(--space-1)', fontSize: 'var(--text-sm)' }}>
          <span>{label}</span>
          <span style={{ color: 'var(--color-muted)' }}>{fmt(used)} / {fmt(total)}</span>
        </div>
      )}
      <div
        style={{
          height: 10,
          background: 'var(--color-border, #e5e7eb)',
          borderRadius: 'var(--radius-s)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${pct}%`,
            background: color,
            borderRadius: 'var(--radius-s)',
            transition: 'width 0.4s ease',
          }}
        />
      </div>
    </div>
  )
}
