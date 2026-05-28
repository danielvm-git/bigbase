import { useEffect, useState } from 'react'

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
  const [user, setUser] = useState<{ id: number; email: string } | null>(null)
  const [health, setHealth] = useState<{ status: string; components: number } | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [messages, setMessages] = useState<Message[]>([])
  const [files, setFiles] = useState<FileObj[]>([])
  const [stats, setStats] = useState<Stat[]>([])
  const [loading, setLoading] = useState(true)

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
    ]).then(([u, h, reposR, depR, msgR, fileR, fnR]) => {
      setUser(u as { id: number; email: string } | null)
      setHealth(h as { status: string; components: number })
      const deps = (depR as { data: Deployment[] }).data || []
      const msgs = (msgR as { data: Message[] }).data || []
      const fils = (fileR as { data: FileObj[] }).data || []
      setDeployments(deps)
      setMessages(msgs)
      setFiles(fils)
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

  const statusClass = (s: string) => {
    if (s === 'running') return 'status-ok'
    if (s === 'failed') return 'status-err'
    if (s === 'building') return 'status-warn'
    return ''
  }

  if (loading) return <p className="loading">Loading dashboard...</p>
  if (!user) return <p className="loading">Loading...</p>

  return (
    <div className="dashboard">
      <h1>Dashboard</h1>

      <div className="dash-grid">
        <div className="card">
          <h3>Signed in as</h3>
          <p className="email">{user.email}</p>
          <p className="dim">ID #{user.id}</p>
        </div>

        {health && (
          <div className="card">
            <h3>System</h3>
            <p className="stat-value">{health.components}</p>
            <p className="dim">components · {health.status}</p>
          </div>
        )}

        <div className="card">
          <h3>Storage</h3>
          <p className="stat-value">{fmtSize(totalBytes)}</p>
          <p className="dim">{files.length} files</p>
        </div>
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
            <div className="card">
              <h3>Deployments by Status</h3>
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
            </div>
          )}

          {msgTotal > 0 && (
            <div className="card">
              <h3>Messages by Channel</h3>
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
            </div>
          )}
        </div>

        <div className="dash-col">
          {recentDeps.length > 0 && (
            <div className="card">
              <h3>Recent Deployments</h3>
              <div className="activity-feed">
                {recentDeps.map(d => (
                  <div key={d.id} className="activity-item">
                    <span className={`status-dot ${statusClass(d.status)}`} />
                    <span className="activity-text">
                      <code>{d.repo_id.slice(0, 8)}</code> · {d.app_type || '?'}
                    </span>
                    <span className={`status-badge ${statusClass(d.status)}`}>{d.status}</span>
                    <span className="dim activity-time">{new Date(d.created_at).toLocaleDateString()}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {recentMsgs.length > 0 && (
            <div className="card">
              <h3>Recent Messages</h3>
              <div className="activity-feed">
                {recentMsgs.map(m => (
                  <div key={m.id} className="activity-item">
                    <span className={`channel-badge channel-${m.channel}`}>{m.channel}</span>
                    <span className="activity-text">{m.to_addr}</span>
                    <span className="dim activity-time">{new Date(m.created_at).toLocaleDateString()}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
