import { Card, CardHeader } from './index'

interface MetricsProps {
  system?: { cpu_percent: number; memory_mb: number; goroutines: number; uptime_seconds: number }
  requests?: { total: number; avg_latency_ms: number; by_status: Record<string, number> }
  componentCount: number
  healthOk: boolean
}

export function DashboardMetrics({ system, requests, componentCount, healthOk }: MetricsProps) {
  if (!system) return null

  return (
    <div className="dash-grid" style={{ marginBottom: 'var(--space-8)' }}>
      <Card>
        <CardHeader title="Request Rate" />
        <p className="stat-value">{requests?.total ?? 0}</p>
        <p className="dim">total requests</p>
      </Card>
      <Card>
        <CardHeader title="Error Rate" />
        <p className="stat-value" style={{ color: (requests?.by_status?.['500'] ?? 0) > 0 ? 'var(--error-fg)' : 'var(--success-fg)' }}>
          {requests?.by_status?.['500'] ?? 0}
        </p>
        <p className="dim">server errors</p>
      </Card>
      <Card>
        <CardHeader title="CPU" />
        <p className="stat-value">{system.cpu_percent.toFixed(1)}%</p>
        <div className="bar-track" style={{ marginTop: 'var(--space-3)' }}>
          <div className="bar-fill" style={{ width: `${Math.min(system.cpu_percent, 100)}%`, background: system.cpu_percent > 80 ? 'var(--error)' : 'var(--brand-500)' }} />
        </div>
      </Card>
      <Card>
        <CardHeader title="Components" />
        <p className="stat-value" style={{ color: healthOk ? 'var(--success-fg)' : 'var(--warning-fg)' }}>
          {componentCount}/16
        </p>
        <p className="dim">healthy</p>
      </Card>
    </div>
  )
}
