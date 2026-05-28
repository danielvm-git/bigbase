import { useEffect, useState } from 'react'

interface Stat {
  label: string
  count: number
  link: string
}

export default function DashboardPage() {
  const [user, setUser] = useState<{ id: number; email: string } | null>(null)
  const [stats, setStats] = useState<Stat[]>([])
  const [health, setHealth] = useState<{ status: string; components: number } | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    const opts = { signal: ctrl.signal }

    fetch('/api/auth/me', opts).then(r => r.ok ? r.json() : null).then(d => setUser(d)).catch(() => {})

    Promise.all([
      fetch('/health', opts).then(r => r.json()).catch(() => ({ status: 'unknown', components: 0 })),
      fetch('/api/git/repos', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/deploy', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/messaging/messages', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/storage/files', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/functions', opts).then(r => r.json()).catch(() => ({ data: [] })),
    ]).then(([h, repos, deploys, msgs, files, fns]) => {
      setHealth(h as { status: string; components: number })
      setStats([
        { label: 'Git Repos', count: ((repos as { data: unknown[] }).data || []).length, link: '/repos' },
        { label: 'Deployments', count: ((deploys as { data: unknown[] }).data || []).length, link: '/deploy' },
        { label: 'Messages', count: ((msgs as { data: unknown[] }).data || []).length, link: '/messaging' },
        { label: 'Files', count: ((files as { data: unknown[] }).data || []).length, link: '/storage' },
        { label: 'Functions', count: ((fns as { data: unknown[] }).data || []).length, link: '/functions' },
      ])
    }).catch(() => {})

    return () => ctrl.abort()
  }, [])

  if (!user) return <p className="loading">Loading...</p>

  return (
    <div className="dashboard">
      <h1>Dashboard</h1>
      <div className="card">
        <h3>Signed in as</h3>
        <p className="email">{user.email}</p>
        <p className="dim id">ID #{user.id}</p>
      </div>

      {health && (
        <div className="card">
          <h3>System</h3>
          <p className="email">{health.components} components · {health.status}</p>
        </div>
      )}

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
    </div>
  )
}
