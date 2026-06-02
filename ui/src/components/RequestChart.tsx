import type { ReactNode } from 'react'
import { Card, CardHeader } from './index'

interface RequestChartProps {
  byStatus: Record<string, number>
  total: number
}

const statusColors: Record<string, string> = {
  '200': '#22c55e',
  '201': '#22c55e',
  '300': '#3b82f6',
  '301': '#3b82f6',
  '302': '#3b82f6',
  '400': '#f59e0b',
  '401': '#f59e0b',
  '403': '#f59e0b',
  '404': '#f59e0b',
  '500': '#ef4444',
}

function barColor(code: string): string {
  return statusColors[code] || '#6b7280'
}

export function RequestChart({ byStatus, total }: RequestChartProps): ReactNode {
  if (total === 0) {
    return (
      <Card>
        <CardHeader title="Requests by Status" />
        <p className="dim">No requests</p>
      </Card>
    )
  }

  const entries = Object.entries(byStatus).sort(([a], [b]) => a.localeCompare(b))

  return (
    <Card>
      <CardHeader title={`Requests by Status (${total} total)`} />
      <div style={{ display: 'flex', gap: 'var(--space-1)', height: 24, borderRadius: 'var(--radius-s)', overflow: 'hidden' }}>
        {entries.map(([code, count]) => {
          const pct = (count / total) * 100
          if (pct < 1) return null
          return (
            <div
              key={code}
              title={`${code}: ${count}`}
              style={{ width: `${pct}%`, background: barColor(code), minWidth: 2, transition: 'width var(--duration-medium) var(--ease-standard)' }}
            />
          )
        })}
      </div>
      <div style={{ display: 'flex', gap: 'var(--space-4)', marginTop: 'var(--space-3)', flexWrap: 'wrap' }}>
        {entries.map(([code, count]) => (
          <span key={code} style={{ fontSize: 'var(--text-xs)', display: 'flex', alignItems: 'center', gap: 'var(--space-1)' }}>
            <span style={{ width: 8, height: 8, borderRadius: 2, background: barColor(code), display: 'inline-block' }} />
            {code}: {count}
          </span>
        ))}
      </div>
    </Card>
  )
}
