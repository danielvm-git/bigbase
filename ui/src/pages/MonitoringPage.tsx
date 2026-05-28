import { useEffect, useState, useCallback } from 'react'

interface SystemMetrics {
  cpu_percent: number
  memory_mb: number
  goroutines: number
  uptime_seconds: number
}

interface RequestMetrics {
  total: number
  by_endpoint: Record<string, { count: number; status_count: Record<string, number>; avg_latency_ms: number }>
  by_status: Record<string, number>
  avg_latency_ms: number
}

interface Metrics {
  system: SystemMetrics
  requests: RequestMetrics
}

interface LogEntry {
  id: string
  level: string
  message: string
  created_at: string
}

interface Alert {
  id: string
  name: string
  metric: string
  threshold: number
  operator: string
  enabled: boolean
}

export default function MonitoringPage() {
  const [metrics, setMetrics] = useState<Metrics | null>(null)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [tab, setTab] = useState<'overview' | 'logs' | 'alerts'>('overview')
  const [logQuery, setLogQuery] = useState('')
  const [showAlertForm, setShowAlertForm] = useState(false)
  const [alertForm, setAlertForm] = useState({ name: '', metric: '', threshold: 0, operator: 'gt', enabled: true })

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await fetch('/api/monitoring/metrics')
      if (res.ok) setMetrics(await res.json())
    } catch {}
  }, [])

  const fetchLogs = useCallback(async (q?: string) => {
    try {
      const url = q ? `/api/monitoring/logs?q=${encodeURIComponent(q)}` : '/api/monitoring/logs'
      const res = await fetch(url)
      if (res.ok) { const d = await res.json(); setLogs((d as { data: LogEntry[] }).data || []) }
    } catch {}
  }, [])

  const fetchAlerts = useCallback(async () => {
    try {
      const res = await fetch('/api/monitoring/alerts')
      if (res.ok) { const d = await res.json(); setAlerts((d as { data: Alert[] }).data || []) }
    } catch {}
  }, [])

  useEffect(() => {
    setLoading(true)
    Promise.all([fetchMetrics(), fetchLogs(), fetchAlerts()]).finally(() => setLoading(false))
  }, [fetchMetrics, fetchLogs, fetchAlerts])

  useEffect(() => {
    if (!metrics) return
    const timer = setInterval(fetchMetrics, 5000)
    return () => clearInterval(timer)
  }, [metrics, fetchMetrics])

  const handleCreateAlert = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/monitoring/alerts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(alertForm),
      })
      if (!res.ok) { const d = await res.json(); setError(d.error || 'create failed'); return }
      setShowAlertForm(false)
      setAlertForm({ name: '', metric: '', threshold: 0, operator: 'gt', enabled: true })
      fetchAlerts()
    } catch { setError('network error') }
  }

  const fmtUptime = (s: number) => {
    const d = Math.floor(s / 86400)
    const h = Math.floor((s % 86400) / 3600)
    return `${d}d ${h}h`
  }

  const levelClass = (l: string) => {
    if (l === 'error') return 'status-err'
    if (l === 'warn') return 'status-warn'
    return 'status-ok'
  }

  if (loading) return <p className="loading">Loading monitoring...</p>

  return (
    <div className="page">
      <div className="page-header">
        <h1>Monitoring</h1>
        <button className="refresh-btn" onClick={() => { fetchMetrics(); fetchLogs(); fetchAlerts() }}>Refresh</button>
      </div>
      {error && <p className="error">{error}</p>}

      <div className="tabs">
        <button className={`tab ${tab === 'overview' ? 'active' : ''}`} onClick={() => setTab('overview')}>Overview</button>
        <button className={`tab ${tab === 'logs' ? 'active' : ''}`} onClick={() => setTab('logs')}>Logs</button>
        <button className={`tab ${tab === 'alerts' ? 'active' : ''}`} onClick={() => setTab('alerts')}>Alerts</button>
      </div>

      {tab === 'overview' && metrics && (
        <>
          <h2 className="section-title">System</h2>
          <div className="stats-grid">
            <div className="stat-card"><span className="stat-count">{metrics.system.memory_mb.toFixed(1)}</span><span className="stat-label">Memory MB</span></div>
            <div className="stat-card"><span className="stat-count">{metrics.system.goroutines}</span><span className="stat-label">Goroutines</span></div>
            <div className="stat-card"><span className="stat-count">{fmtUptime(metrics.system.uptime_seconds)}</span><span className="stat-label">Uptime</span></div>
          </div>

          <h2 className="section-title">Requests</h2>
          <div className="stats-grid">
            <div className="stat-card"><span className="stat-count">{metrics.requests.total}</span><span className="stat-label">Total</span></div>
            <div className="stat-card"><span className="stat-count">{metrics.requests.avg_latency_ms.toFixed(1)}ms</span><span className="stat-label">Avg Latency</span></div>
          </div>

          {Object.keys(metrics.requests.by_endpoint).length > 0 && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Endpoint</th><th>Count</th><th>Avg Latency</th><th>Status Codes</th></tr>
                </thead>
                <tbody>
                  {Object.entries(metrics.requests.by_endpoint).map(([path, ep]) => (
                    <tr key={path}>
                      <td><code>{path}</code></td>
                      <td>{ep.count}</td>
                      <td>{ep.avg_latency_ms.toFixed(1)}ms</td>
                      <td>{Object.entries(ep.status_count).map(([code, count]) => (
                        <span key={code} className={`status-badge ${code.startsWith('2') ? 'status-ok' : code.startsWith('4') ? 'status-warn' : 'status-err'}`}>{code}: {count}</span>
                      ))}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {tab === 'logs' && (
        <>
          <div className="card">
            <form onSubmit={e => { e.preventDefault(); fetchLogs(logQuery) }} className="inline-form">
              <input placeholder="Search logs..." value={logQuery} onChange={e => setLogQuery(e.target.value)} />
              <button type="submit" className="create-btn">Search</button>
            </form>
          </div>
          {logs.length === 0 ? <p className="dim">No logs.</p> : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Level</th><th>Message</th><th>Time</th></tr>
                </thead>
                <tbody>
                  {logs.map(l => (
                    <tr key={l.id}>
                      <td><span className={`status-badge ${levelClass(l.level)}`}>{l.level}</span></td>
                      <td>{l.message}</td>
                      <td>{new Date(l.created_at).toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {tab === 'alerts' && (
        <>
          <button className="create-btn" onClick={() => setShowAlertForm(!showAlertForm)}>{showAlertForm ? 'Cancel' : 'New Alert'}</button>
          {showAlertForm && (
            <div className="card">
              <form onSubmit={handleCreateAlert} className="fn-form">
                <input placeholder="Name *" value={alertForm.name} onChange={e => setAlertForm(p => ({ ...p, name: e.target.value }))} required />
                <input placeholder="Metric *" value={alertForm.metric} onChange={e => setAlertForm(p => ({ ...p, metric: e.target.value }))} required />
                <input placeholder="Threshold" type="number" step="0.1" value={alertForm.threshold} onChange={e => setAlertForm(p => ({ ...p, threshold: +e.target.value }))} />
                <select value={alertForm.operator} onChange={e => setAlertForm(p => ({ ...p, operator: e.target.value }))}>
                  <option value="gt">Greater Than</option>
                  <option value="lt">Less Than</option>
                  <option value="eq">Equals</option>
                </select>
                <label className="checkbox-label">
                  <input type="checkbox" checked={alertForm.enabled} onChange={e => setAlertForm(p => ({ ...p, enabled: e.target.checked }))} />
                  Enabled
                </label>
                <button type="submit" className="create-btn">Create</button>
              </form>
            </div>
          )}
          {alerts.length === 0 ? <p className="dim">No alerts.</p> : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Name</th><th>Metric</th><th>Threshold</th><th>Operator</th><th>Status</th></tr>
                </thead>
                <tbody>
                  {alerts.map(a => (
                    <tr key={a.id}>
                      <td>{a.name}</td>
                      <td><code>{a.metric}</code></td>
                      <td>{a.threshold}</td>
                      <td>{a.operator}</td>
                      <td><span className={`status-badge ${a.enabled ? 'status-ok' : 'status-warn'}`}>{a.enabled ? 'Enabled' : 'Disabled'}</span></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  )
}
