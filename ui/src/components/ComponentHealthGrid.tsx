import type { ReactNode } from 'react'
import { Card, CardHeader, Badge } from './index'

type HealthStatus = 'healthy' | 'degraded' | 'down' | 'unknown'

interface ComponentHealth {
  name: string
  status: HealthStatus
  version?: string
}

interface ComponentHealthGridProps {
  components: ComponentHealth[]
}

const statusVariant: Record<HealthStatus, 'success' | 'warning' | 'error' | 'neutral'> = {
  healthy: 'success',
  degraded: 'warning',
  down: 'error',
  unknown: 'neutral',
}

export function ComponentHealthGrid({ components }: ComponentHealthGridProps): ReactNode {
  if (components.length === 0) {
    return (
      <Card>
        <CardHeader title="Component Health" />
        <p className="dim">No component data available</p>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader title="Component Health" />
      <div className="dash-grid" style={{ marginTop: 'var(--space-4)' }}>
        {components.map((c) => (
          <div key={c.name} style={{ padding: 'var(--space-3)', display: 'flex', alignItems: 'center', gap: 'var(--space-3)', border: '1px solid var(--border-default)', borderRadius: 'var(--radius-s)' }}>
            <Badge variant={statusVariant[c.status]}>{c.status}</Badge>
            <div>
              <p style={{ fontWeight: 600, fontSize: 'var(--text-s)' }}>{c.name}</p>
              {c.version && <p className="dim" style={{ fontSize: 'var(--text-xs)' }}>v{c.version}</p>}
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}
