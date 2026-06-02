import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardHeader, PageHeader, Badge, statusBadgeVariant, Button } from '../components'
import { useToast } from '../hooks/useToast'

interface Stat { label: string; count: number; link: string }
interface Deployment { id: string; repo_id: string; branch: string; status: string; app_type: string; url: string; created_at: string }
interface Message { id: string; channel: string; to_addr: string; subject: string; status: string; created_at: string }
interface FileObj { id: string; name: string; size: number; mime_type: string; created_at: string }

function fmtSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function DashboardPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const [user, setUser] = useState<{ id: number; email: string } | null>(null)
  const [health, setHealth] = useState<{ status: string; components: number } | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [messages, setMessages] = useState<Message[]>([])
  const [files, setFiles] = useState<FileObj[]>([])
  const [stats, setStats] = useState<Stat[]>([])
  const [loading, setLoading] = useState(true)
  const [metrics, setMetrics] = useState<{
    system?: { cpu_percent: number; memory_mb: number; goroutines: number; uptime_seconds: number }
    requests?: { total: number; avg_latency_ms: number; by_status: Record<string, number> }
  }>({})

  useEffect(() => {
    const ctrl = new AbortController()
    const opts = { signal: ctrl.signal }

    Promise.all([
      fetch('/api/auth/me', opts).then(r => r.ok ? r.json() : null).catch(() => null),
      fetch('/health', opts).then(r => r.json()).catch(() => ({ status: 'unknown', components: 0 })),
      fetch('/api/git/repos', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/deploy', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/messaging/messages', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/storage/files', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/functions', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/monitoring/metrics', opts).then(r => r.json()).catch(() => ({})),
    ]).then(([u, h, reposR, depR, msgR, fileR, fnR, metricsR]) => {
      setUser(u as { id: number; email: string } | null)
      setHealth(h as { status: string; components: number })
      const deps = (depR as { data: Deployment[] }).data || []
      const msgs = (msgR as { data: Message[] }).data || []
      const fils = (fileR as { data: FileObj[] }).data || []
      setDeployments(deps)
      setMessages(msgs)
      setFiles(fils)
      setMetrics(metricsR as typeof metrics)
      setStats([
        { label: 'Git Repos', count: ((reposR as { data: unknown[] }).data || []).length, link: '/repos' },
        { label: 'Deployments', count: deps.length, link: '/deploy' },
        { label: 'Messages', count: msgs.length, link: '/messaging' },
        { label: 'Files', count: fils.length, link: '/storage' },
        { label: 'Functions', count: ((fnR as { data: unknown[] }).data || []).length, link: '/functions' },
      ])
    }).catch(() => {}).finally(() => setLoading(false))

    return () => ctrl.abort()
  }, [])

  const depByStatus: Record<string, number> = {}
  deployments.forEach(d => { depByStatus[d.status] = (depByStatus[d.status] || 0) + 1 })
  const depTotal = deployments.length

  const msgByChannel: Record<string, number> = {}
  messages.forEach(m => { msgByChannel[m.channel] = (msgByChannel[m.channel] || 0) + 1 })
  const msgTotal = messages.length

  const totalBytes = files.reduce((acc, f) => acc + f.size, 0)

  const statusColors: Record<string, string> = { running: '#22c55e', failed: '#ef4444', building: '#f59e0b', pending: '#6b7280' }
  const channelColors: Record<string, string> = { email: '#3b82f6', sms: '#8b5cf6', push: '#ec4899' }

  const recentDeps = deployments.slice(0, 5)
  const recentMsgs = messages.slice(0, 5)

  if (loading) return <div className="loading">Loading dashboard...</div>
  if (!user) return <div className="loading">Loading...</div>

  return (
    <div className="dashboard">
      <PageHeader title="Dashboard" />

      <div style={{ display: 'flex', gap: 'var(--space-4)', marginBottom: 'var(--space-12)', flexWrap: 'wrap' }}>
        <Button variant="primary" size="sm" onClick={() => { navigate('/deploy/new'); toast.show('Create a new deployment', 'info') }}>
          + Deploy Site
        </Button>
        <Button variant="secondary" size="sm" onClick={() => { navigate('/functions'); toast.show('Manage your serverless functions', 'info') }}>
          ⚡ Run Function
        </Button>
        <Button variant="secondary" size="sm" onClick={() => { toast.show('Feature coming soon', 'info') }}>
          📦 Create Collection
        </Button>
      </div>

      {health && (
        <div className="card" style={{
          marginBottom: 'var(--space-8)',
          padding: 'var(--space-6) var(--space-8)',
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--space-6)',
          background: health.status === 'ok' ? 'var(--success-bg)' : 'var(--warning-bg)',
          border: `1px solid ${health.status === 'ok' ? 'var(--success)' : 'var(--warning)'}`,
        }}>
          <span style={{ fontSize: '1.2rem' }}>{health.status === 'ok' ? '✅' : '⚠️'}</span>
          <span style={{ fontWeight: 600, color: health.status === 'ok' ? 'var(--success-fg)' : 'var(--warning-fg)' }}>
            {health.status === 'ok' ? 'All systems operational' : 'System issues detected'}
          </span>
          <span style={{ color: 'var(--fg-tertiary)', fontSize: 'var(--text-s)' }}>
            {health.components} component{health.components !== 1 ? 's' : ''} running
          </span>
        </div>
      )}

      {metrics.system && (
        <div className="dash-grid" style={{ marginBottom: 'var(--space-8)' }}>
          <Card>
            <CardHeader title="Request Rate" />
            <p className="stat-value">{metrics.requests?.total ?? 0}</p>
            <p className="dim">total requests</p>
          </Card>
          <Card>
            <CardHeader title="Error Rate" />
            <p className="stat-value" style={{ color: (metrics.requests?.by_status?.['500'] ?? 0) > 0 ? 'var(--error-fg)' : 'var(--success-fg)' }}>
              {metrics.requests?.by_status?.['500'] ?? 0}
            </p>
            <p className="dim">server errors</p>
          </Card>
          <Card>
            <CardHeader title="CPU" />
            <p className="stat-value">{metrics.system.cpu_percent.toFixed(1)}%</p>
            <div className="bar-track" style={{ marginTop: 'var(--space-3)' }}>
              <div className="bar-fill" style={{ width: `${Math.min(metrics.system.cpu_percent, 100)}%`, background: metrics.system.cpu_percent > 80 ? 'var(--error)' : 'var(--brand-500)' }} />
            </div>
          </Card>
          <Card>
            <CardHeader title="Components" />
            <p className="stat-value" style={{ color: health?.status === 'ok' ? 'var(--success-fg)' : 'var(--warning-fg)' }}>
              {health?.components ?? 0}/16
            </p>
            <p className="dim">healthy</p>
          </Card>
        </div>
      )}

      <div className="dash-grid">
        <Card>
          <CardHeader title="Signed in as" />
          <p style={{ fontSize: 'var(--text-m)', fontWeight: 500 }}>{user.email}</p>
          <p className="dim">ID #{user.id}</p>
        </Card>

        {health && (
          <Card>
            <CardHeader title="System" />
            <p className="stat-value">{health.components}</p>
            <p className="dim">components &middot; {health.status}</p>
          </Card>
        )}

        <Card>
          <CardHeader title="Storage" />
          <p className="stat-value">{fmtSize(totalBytes)}</p>
          <p className="dim">{files.length} files</p>
        </Card>
      </div>

      {stats.length > 0 && (
        <div className="stats-grid">
          {stats.map(s => (
            <a key={s.label} href={`/admin/#${s.link}`} className="stat-card">
              <span className="stat-count">{s.count}</span>
              <span className="stat-label">{s.label}</span>
            </a>
          ))}
        </div>
      )}

      <div className="dash-cols">
        <div className="dash-col">
          {depTotal > 0 && (
            <Card>
              <CardHeader title="Deployments by Status" />
              <div className="bar-chart">
                {Object.entries(depByStatus).map(([status, count]) => (
                  <div key={status} className="bar-row">
                    <span className="bar-label">{status}</span>
                    <div className="bar-track">
                      <div className="bar-fill" style={{ width: `${(count / depTotal) * 100}%`, background: statusColors[status] || '#6b7280' }} />
                    </div>
                    <span className="bar-count">{count}</span>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {msgTotal > 0 && (
            <Card>
              <CardHeader title="Messages by Channel" />
              <div className="bar-chart">
                {Object.entries(msgByChannel).map(([ch, count]) => (
                  <div key={ch} className="bar-row">
                    <span className="bar-label">{ch}</span>
                    <div className="bar-track">
                      <div className="bar-fill" style={{ width: `${(count / msgTotal) * 100}%`, background: channelColors[ch] || '#6b7280' }} />
                    </div>
                    <span className="bar-count">{count}</span>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </div>

        <div className="dash-col">
          {recentDeps.length > 0 && (
            <Card>
              <CardHeader title="Recent Deployments" />
              <div className="activity-feed">
                {recentDeps.map(d => (
                  <div key={d.id} className="activity-item">
                    <Badge variant={statusBadgeVariant(d.status)}>{d.status}</Badge>
                    <span className="activity-text">
                      <code>{d.repo_id.slice(0, 8)}</code> &middot; {d.app_type || '?'}
                    </span>
                    <span className="activity-time">{new Date(d.created_at).toLocaleDateString()}</span>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {recentMsgs.length > 0 && (
            <Card>
              <CardHeader title="Recent Messages" />
              <div className="activity-feed">
                {recentMsgs.map(m => (
                  <div key={m.id} className="activity-item">
                    <span className={`channel-badge channel-${m.channel}`}>{m.channel}</span>
                    <span className="activity-text">{m.to_addr}</span>
                    <span className="activity-time">{new Date(m.created_at).toLocaleDateString()}</span>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
