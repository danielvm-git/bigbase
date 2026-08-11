import type { ReactNode } from 'react'

export interface BarSegment {
  value: number
  label: string
  color: string
}

type BarGaugeMode = 'single' | 'stacked' | 'grouped'

interface BarGaugeProps {
  used: number
  total: number
  label?: string
  color?: string
  formatValue?: (n: number) => string
  mode?: BarGaugeMode
  segments?: BarSegment[]
}

function SingleBar({ used, total, label, color, formatValue }: Omit<BarGaugeProps, 'mode' | 'segments'>) {
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
        role="progressbar"
        aria-label={label ? `${label}: ${fmt(used)} of ${fmt(total)}` : `${fmt(used)} of ${fmt(total)}`}
        aria-valuenow={Math.min(used, total)}
        aria-valuemin={0}
        aria-valuemax={total}
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

function StackedBar({ total, segments = [], formatValue }: { total: number; segments: BarSegment[]; formatValue?: (n: number) => string }) {
  const fmt = formatValue ?? ((n: number) => n.toString())
  const segTotal = segments.reduce((acc, s) => acc + s.value, 0)
  const base = Math.max(total, segTotal) || 1

  return (
    <div style={{ width: '100%' }}>
      <div style={{ display: 'flex', gap: 4, marginBottom: 4, fontSize: 'var(--text-sm)' }}>
        {segments.map(s => (
          <span key={s.label} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', background: s.color, display: 'inline-block' }} />
            <span>{s.label}</span>
            <span style={{ color: 'var(--color-muted)' }}>{fmt(s.value)}</span>
          </span>
        ))}
      </div>
      <div style={{ height: 10, background: 'var(--color-border, #e5e7eb)', borderRadius: 'var(--radius-s)', overflow: 'hidden', display: 'flex' }}>
        {segments.map(s => (
          <div
            key={s.label}
            role="img"
            aria-label={`${s.label}: ${fmt(s.value)}`}
            style={{
              height: '100%',
              width: `${(s.value / base) * 100}%`,
              background: s.color,
              transition: 'width 0.4s ease',
            }}
          />
        ))}
      </div>
    </div>
  )
}

function GroupedBar({ total, segments = [], formatValue }: { total: number; segments: BarSegment[]; formatValue?: (n: number) => string }) {
  const fmt = formatValue ?? ((n: number) => n.toString())
  const base = Math.max(total, ...segments.map(s => s.value)) || 1

  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 6 }}>
      {segments.map(s => (
        <div key={s.label}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2, fontSize: 'var(--text-sm)' }}>
            <span>{s.label}</span>
            <span style={{ color: 'var(--color-muted)' }}>{fmt(s.value)}</span>
          </div>
          <div style={{ height: 8, background: 'var(--color-border, #e5e7eb)', borderRadius: 'var(--radius-s)', overflow: 'hidden' }}>
            <div
              role="img"
              aria-label={`${s.label}: ${fmt(s.value)}`}
              style={{
                height: '100%',
                width: `${(s.value / base) * 100}%`,
                background: s.color,
                borderRadius: 'var(--radius-s)',
                transition: 'width 0.4s ease',
              }}
            />
          </div>
        </div>
      ))}
    </div>
  )
}

export function BarGauge({ used, total, label, color = '#3b82f6', formatValue, mode = 'single', segments }: BarGaugeProps): ReactNode {
  if (mode === 'stacked' && segments) {
    return <StackedBar total={total} segments={segments} formatValue={formatValue} />
  }
  if (mode === 'grouped' && segments) {
    return <GroupedBar total={total} segments={segments} formatValue={formatValue} />
  }
  return <SingleBar used={used} total={total} label={label} color={color} formatValue={formatValue} />
}
