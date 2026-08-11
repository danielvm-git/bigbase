import { Card, Badge, statusBadgeVariant } from './index'
import { Icon } from './Icon'
import { fmtUptime, fmtMemoryMB, memoryBarPercent } from '../lib/format'

export interface ActivityItem {
  id: string
  label: string
  when: string
  status: string
  statusLabel: string
}

interface SystemStatusPanelProps {
  healthOk: boolean
  componentCount: number
  runningCount?: number
  system?: {
    cpu_percent: number
    memory_mb: number
    goroutines: number
    uptime_seconds: number
  }
  activity: ActivityItem[]
}

export function SystemStatusPanel({
  healthOk,
  componentCount,
  runningCount,
  system,
  activity,
}: SystemStatusPanelProps) {
  const cpu = system?.cpu_percent ?? 0
  const memoryMB = system?.memory_mb ?? 0
  const running = runningCount ?? (healthOk ? componentCount : 0)
  const uptime = system ? fmtUptime(system.uptime_seconds) : '—'
  const memPct = memoryBarPercent(memoryMB)

  return (
    <Card className="system-status-panel">
      <div className="system-status-header">
        <div className="system-status-title-row">
          <span className="stat-icon system-status-icon">
            <Icon name="activity" size={16} />
          </span>
          <div>
            <h2 className="system-status-title" style={{ fontSize: 'inherit', lineHeight: 'inherit' }}>
              {healthOk ? 'All systems operational' : 'System issues detected'}
            </h2>
            <div className="dim system-status-sub">
              Live health across {componentCount} components · uptime {uptime}
            </div>
          </div>
        </div>
        <Badge variant={healthOk ? 'success' : 'warning'}>{healthOk ? 'Healthy' : 'Degraded'}</Badge>
      </div>

      {system ? (
        <div className="metric-grid">
          <div className="metric-tile">
            <div className="metric-tile-top">
              <span className="metric-label">Components</span>
              <Icon name="activity" size={15} className="dim" />
            </div>
            <div className="metric-value">
              {running}<span className="unit"> / {componentCount}</span>
            </div>
            <div className="dim">{healthOk ? 'all running' : 'some components offline'}</div>
          </div>
          <div className="metric-tile">
            <div className="metric-tile-top">
              <span className="metric-label">CPU</span>
              <Icon name="activity" size={15} className="dim" />
            </div>
            <div className="metric-value">
              {cpu.toFixed(1)}<span className="unit">%</span>
            </div>
            <div
              className="bar-track metric-bar"
              role="progressbar"
              aria-label={`CPU: ${cpu.toFixed(1)} percent`}
              aria-valuenow={Math.min(cpu, 100)}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div
                className="bar-fill"
                style={{
                  width: `${Math.min(cpu, 100)}%`,
                  background: cpu > 80 ? 'var(--error)' : 'var(--brand-500)',
                }}
              />
            </div>
          </div>
          <div className="metric-tile">
            <div className="metric-tile-top">
              <span className="metric-label">Memory</span>
              <Icon name="hard-drive" size={15} className="dim" />
            </div>
            <div className="metric-value">{fmtMemoryMB(memoryMB)}</div>
            <div className="dim">process heap</div>
            <div
              className="bar-track metric-bar"
              role="progressbar"
              aria-label={`Memory: ${fmtMemoryMB(memoryMB)} (${memPct} percent of 1 GiB)`}
              aria-valuenow={memPct}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div
                className="bar-fill"
                style={{ width: `${memPct}%`, background: 'var(--success)' }}
              />
            </div>
          </div>
        </div>
      ) : (
        <p className="dim">Metrics unavailable — server may still be starting.</p>
      )}

      {activity.length > 0 && (
        <div className="system-status-activity">
          <div className="metric-label system-activity-label">Activity</div>
          <div className="system-activity-list">
            {activity.map(a => (
              <div key={a.id} className="activity-item system-activity-item">
                <span className="stat-icon system-activity-icon">
                  <Icon name="rocket" size={15} />
                </span>
                <div className="system-activity-body">
                  <div className="system-activity-label-text">{a.label}</div>
                  <div className="dim">{a.when}</div>
                </div>
                <Badge variant={statusBadgeVariant(a.status)}>{a.statusLabel}</Badge>
              </div>
            ))}
          </div>
        </div>
      )}
    </Card>
  )
}
