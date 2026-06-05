/** Legacy metrics row; dashboard uses SystemStatusPanel. Tests guard request-rate rendering. */
import { Card, CardHeader } from './index'
import { MetricCard } from './MetricCard'

interface MetricsProps {
  system?: { cpu_percent: number; memory_mb: number; goroutines: number; uptime_seconds: number }
  requests?: { total: number; avg_latency_ms: number; by_status: Record<string, number> }
  componentCount: number
  healthOk: boolean
}

export function DashboardMetrics({ system, requests, componentCount, healthOk }: MetricsProps) {
  if (!system) {
    return (
      <div className="dash-grid" style={{ marginBottom: 'var(--space-8)' }}>
        <Card>
          <CardHeader title="Metrics" />
          <p className="dim">Metrics unavailable — server may still be starting.</p>
        </Card>
      </div>
    )
  }

  const errorCount = requests?.by_status?.['500'] ?? 0

  return (
    <div className="dash-grid" style={{ marginBottom: 'var(--space-8)' }}>
      <MetricCard label="Request Rate" value={requests?.total ?? 0} subtitle="total requests" />
      <MetricCard
        label="Error Rate"
        value={errorCount}
        color={errorCount > 0 ? 'error' : 'success'}
        subtitle="server errors"
      />
      <MetricCard
        label="CPU"
        value={`${system.cpu_percent.toFixed(1)}%`}
        color={system.cpu_percent > 80 ? 'error' : 'neutral'}
        subtitle={`${system.memory_mb.toFixed(0)} MB memory`}
      />
      <MetricCard
        label="Components"
        value={`${componentCount} healthy`}
        color={healthOk ? 'success' : 'warning'}
        subtitle={`${system.goroutines} goroutines`}
      />
    </div>
  )
}
