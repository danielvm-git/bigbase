import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Card, CardHeader, PageHeader, Badge, statusBadgeVariant, Button, OnboardingChecklist } from '../components'
import { Icon, type IconName } from '../components/Icon'
import { SystemStatusPanel, type ActivityItem } from '../components/SystemStatusPanel'
import { timeAgo, mapDeployStatus } from '../lib/format'
import { fetchMonitoringMetricsWarmed } from '../lib/metrics'

interface User { id: number; email: string }
interface Site { id: string; name: string }
interface Deployment {
  id: string
  repo_id: string
  branch: string
  commit_sha?: string
  status: string
  app_type: string
  url: string
  created_at: string
}

interface StatLink {
  label: string
  count: number
  icon: IconName
  link: string
}

const JUMP_LINKS: { label: string; icon: IconName; to: string }[] = [
  { label: 'Deploy a site from GitHub', icon: 'rocket', to: '/deploy/new' },
  { label: 'Write a SQL query', icon: 'terminal', to: '/sql' },
  { label: 'Add a serverless function', icon: 'box', to: '/functions' },
  { label: 'Invite a teammate', icon: 'users', to: '/users' },
]

function activityStatusLabel(status: string): string {
  const mapped = mapDeployStatus(status)
  if (mapped === 'ready') return 'COMPLETED'
  if (mapped === 'building') return 'RUNNING'
  if (mapped === 'failed') return 'FAILED'
  return status.toUpperCase()
}

export default function DashboardPage() {
  const navigate = useNavigate()
  const [user, setUser] = useState<User | null>(null)
  const [health, setHealth] = useState<{ status: string; components: number; running?: number } | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [stats, setStats] = useState<StatLink[]>([])
  const [loading, setLoading] = useState(true)
  const [metrics, setMetrics] = useState<{
    system?: { cpu_percent: number; memory_mb: number; goroutines: number; uptime_seconds: number }
  }>({})

  useEffect(() => {
    const ctrl = new AbortController()
    const opts = { signal: ctrl.signal }

    Promise.all([
      fetch('/api/auth/me', opts).then(r => r.ok ? r.json() : null).catch(() => null),
      fetch('/health', opts).then(r => r.json()).catch(() => ({ status: 'unknown', components: 0 })),
      fetch('/api/sites', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/deploy', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/functions', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/git/repos', opts).then(r => r.json()).catch(() => ({ data: [] })),
      fetch('/api/users', opts).then(r => r.json()).catch(() => ({ data: [] })),
    ]).then(async ([u, h, sitesR, depR, fnR, reposR, usersR]) => {
      setUser(u as User | null)
      setHealth(h as { status: string; components: number; running?: number })
      const deps = (depR as { data: Deployment[] }).data || []
      const sites = (sitesR as { data: Site[] }).data || []
      setDeployments(deps)
      const metricsR = await fetchMonitoringMetricsWarmed(ctrl.signal)
      if (!ctrl.signal.aborted) {
        setMetrics(metricsR)
      }
      setStats([
        { label: 'Sites', count: sites.length, icon: 'rocket', link: '/deploy' },
        { label: 'Functions', count: ((fnR as { data: unknown[] }).data || []).length, icon: 'box', link: '/functions' },
        { label: 'Git Repos', count: ((reposR as { data: unknown[] }).data || []).length, icon: 'git-branch', link: '/repos' },
        { label: 'Users', count: ((usersR as { data: unknown[] }).data || []).length, icon: 'users', link: '/users' },
      ])
    }).catch(() => {}).finally(() => setLoading(false))

    return () => ctrl.abort()
  }, [])

  const recentDeps = [...deployments]
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 5)

  const activity: ActivityItem[] = recentDeps.map(d => ({
    id: d.id,
    label: `Deploy ${d.repo_id.slice(0, 8)}`,
    when: timeAgo(d.created_at),
    status: d.status,
    statusLabel: activityStatusLabel(d.status),
  }))

  if (loading) return <div className="loading" role="status" aria-busy="true">Loading dashboard...</div>
  if (!user) return <div className="loading" role="status" aria-busy="true">Loading...</div>

  const displayName = user.email.split('@')[0]
  const healthOk = health?.status === 'ok'

  return (
    <div className="dashboard">
      <PageHeader
        title={`Welcome back, ${displayName}`}
        subtitle="Here's what's running on your BigBase instance."
      >
        <Button variant="primary" size="sm" onClick={() => navigate('/deploy/new')}>
          <Icon name="plus" size={16} /> Create site
        </Button>
      </PageHeader>

      <OnboardingChecklist />

      <SystemStatusPanel
        healthOk={healthOk}
        componentCount={health?.components ?? 0}
        runningCount={health?.running}
        system={metrics.system}
        activity={activity}
      />

      <div className="stats-grid stats-grid-dashboard">
        {stats.map(s => (
          <Link key={s.label} to={s.link} className="stat-card stat-card-linked">
            <div className="stat-card-top">
              <div className="stat-icon">
                <Icon name={s.icon} size={18} />
              </div>
              <span className="stat-card-chevron dim" aria-hidden>›</span>
            </div>
            <div className="stat-count">{s.count}</div>
            <div className="stat-label">{s.label}</div>
          </Link>
        ))}
      </div>

      <div className="dash-cols dash-cols-dashboard">
        <Card>
          <CardHeader title="Recent deployments">
            <button type="button" className="btn btn-link btn-sm" onClick={() => navigate('/deploy')}>
              View all
            </button>
          </CardHeader>
          {recentDeps.length === 0 ? (
            <p className="dim">No deployments yet.</p>
          ) : (
            <div className="dashboard-deploy-list">
              {recentDeps.map(d => (
                <div key={d.id} className="dashboard-deploy-row">
                  <Badge variant={statusBadgeVariant(mapDeployStatus(d.status))}>
                    {mapDeployStatus(d.status).toUpperCase()}
                  </Badge>
                  <div className="dashboard-deploy-body">
                    <div className="dashboard-deploy-title">{d.repo_id.slice(0, 8)}</div>
                    <div className="dim dashboard-deploy-msg">
                      {d.branch} · {d.app_type || 'app'}
                    </div>
                  </div>
                  {d.commit_sha && (
                    <span className="mono dim dashboard-deploy-commit">{d.commit_sha.slice(0, 7)}</span>
                  )}
                  <span className="dim dashboard-deploy-time">{timeAgo(d.created_at)}</span>
                </div>
              ))}
            </div>
          )}
        </Card>

        <Card>
          <CardHeader title="Jump back in" />
          <div className="jump-back-list">
            {JUMP_LINKS.map(q => (
              <Link key={q.label} to={q.to} className="jump-back-item">
                <span className="stat-icon jump-back-icon">
                  <Icon name={q.icon} size={15} />
                </span>
                <span className="jump-back-label">{q.label}</span>
                <span className="dim" aria-hidden>›</span>
              </Link>
            ))}
          </div>
        </Card>
      </div>
    </div>
  )
}
