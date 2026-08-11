import { useEffect, useState, useCallback, useRef } from 'react'
import { PageHeader, Button, Input, Tabs, Badge, Sparkline, DonutGauge, BarGauge } from '../components'
import { fetchMonitoringMetricsWarmed } from '../lib/metrics'
import type { HostMetricsSlice, SSEMetricsEvent } from '../lib/metrics'

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
  host?: HostMetricsSlice
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

const MAX_SPARKLINE_SAMPLES = 60

function fmtBytes(b: number): string {
  if (b >= 1e9) return `${(b / 1e9).toFixed(1)} GB`
  if (b >= 1e6) return `${(b / 1e6).toFixed(1)} MB`
  if (b >= 1e3) return `${(b / 1e3).toFixed(1)} KB`
  return `${b} B`
}

export default function MonitoringPage() {
  const [metrics, setMetrics] = useState<Metrics | null>(null)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [tab, setTab] = useState('overview')
  const [logQuery, setLogQuery] = useState('')
  const [showAlertForm, setShowAlertForm] = useState(false)
  const [alertForm, setAlertForm] = useState({ name: '', metric: '', threshold: 0, operator: 'gt', enabled: true })

  // Sparkline history: ring buffers of the last MAX_SPARKLINE_SAMPLES values.
  const [cpuHistory, setCpuHistory] = useState<number[]>([])
  const [netInHistory, setNetInHistory] = useState<number[]>([])
  const [netOutHistory, setNetOutHistory] = useState<number[]>([])
  const prevNetRef = useRef<{ bytesIn: number; bytesOut: number } | null>(null)

  const pushSample = (setter: React.Dispatch<React.SetStateAction<number[]>>, value: number) => {
    setter(prev => {
      const next = [...prev, value]
      return next.length > MAX_SPARKLINE_SAMPLES ? next.slice(-MAX_SPARKLINE_SAMPLES) : next
    })
  }

  // Apply an SSE event to state.
  const applySSEEvent = useCallback((evt: SSEMetricsEvent) => {
    setMetrics(prev => {
      if (!prev) return prev
      return {
        ...prev,
        system: evt.system,
        host: evt.host,
      }
    })
    pushSample(setCpuHistory, evt.host.cpu_percent)

    // Derive per-interval network delta (bytes/s approximation).
    const cur = { bytesIn: evt.host.net_bytes_in, bytesOut: evt.host.net_bytes_out }
    if (prevNetRef.current) {
      const deltaIn = Math.max(cur.bytesIn - prevNetRef.current.bytesIn, 0)
      const deltaOut = Math.max(cur.bytesOut - prevNetRef.current.bytesOut, 0)
      pushSample(setNetInHistory, deltaIn)
      pushSample(setNetOutHistory, deltaOut)
    }
    prevNetRef.current = cur
  }, [])

  const fetchMetrics = useCallback(async () => {
    try {
      const data = await fetchMonitoringMetricsWarmed()
      if (!data.system) return
      const requests = data.requests ?? { total: 0, by_endpoint: {}, by_status: {}, avg_latency_ms: 0 }
      setMetrics({
        system: data.system,
        host: data.host,
        requests: {
          total: requests.total,
          by_endpoint: requests.by_endpoint ?? {},
          by_status: requests.by_status ?? {},
          avg_latency_ms: requests.avg_latency_ms,
        },
      })
    } catch { /* silent */ }
  }, [])

  const fetchLogs = useCallback(async (q?: string) => {
    try {
      const url = q ? `/api/monitoring/logs?q=${encodeURIComponent(q)}` : '/api/monitoring/logs'
      const res = await fetch(url)
      if (res.ok) { const d = await res.json(); setLogs((d as { data: LogEntry[] }).data || []) }
    } catch { /* silent */ }
  }, [])

  const fetchAlerts = useCallback(async () => {
    try {
      const res = await fetch('/api/monitoring/alerts')
      if (res.ok) { const d = await res.json(); setAlerts((d as { data: Alert[] }).data || []) }
    } catch { /* silent */ }
  }, [])

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    setLoading(true)
    Promise.all([fetchMetrics(), fetchLogs(), fetchAlerts()]).finally(() => setLoading(false))
  }, [fetchMetrics, fetchLogs, fetchAlerts])
  /* eslint-enable react-hooks/set-state-in-effect */

  // SSE stream for live host metrics updates.
  useEffect(() => {
    if (!metrics) return
    const es = new EventSource('/api/monitoring/metrics/stream')
    es.onmessage = (e: MessageEvent<string>) => {
      try {
        const evt = JSON.parse(e.data) as SSEMetricsEvent
        applySSEEvent(evt)
      } catch { /* ignore malformed events */ }
    }
    es.onerror = () => { /* reconnect is automatic */ }
    return () => es.close()
  }, [metrics === null, applySSEEvent]) // eslint-disable-line react-hooks/exhaustive-deps

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

  const monitoringTabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'host', label: 'Host' },
    { id: 'logs', label: 'Logs' },
    { id: 'alerts', label: 'Alerts' },
  ]

  if (loading) return <div className="loading" role="status" aria-busy="true">Loading monitoring...</div>

  return (
    <div>
      <PageHeader title="Monitoring">
        <Button variant="secondary" size="sm" onClick={() => { fetchMetrics(); fetchLogs(); fetchAlerts() }}>Refresh</Button>
      </PageHeader>
      {error && <p className="input-error-text" role="alert">{error}</p>}

      <Tabs tabs={monitoringTabs} active={tab} onChange={setTab} />

      {tab === 'overview' && metrics && (
        <>
          <h2 className="section-title">System</h2>
          <div className="stats-grid" aria-live="polite">
            <div className="stat-card"><span className="stat-count">{metrics.system.cpu_percent.toFixed(1)}%</span><span className="stat-label">CPU</span></div>
            <div className="stat-card"><span className="stat-count">{metrics.system.memory_mb.toFixed(1)}</span><span className="stat-label">Heap MB</span></div>
            <div className="stat-card"><span className="stat-count">{metrics.system.goroutines}</span><span className="stat-label">Goroutines</span></div>
            <div className="stat-card"><span className="stat-count">{fmtUptime(metrics.system.uptime_seconds)}</span><span className="stat-label">Uptime</span></div>
          </div>

          <h2 className="section-title">Requests</h2>
          <div className="stats-grid">
            <div className="stat-card"><span className="stat-count">{metrics.requests.total}</span><span className="stat-label">Total</span></div>
            <div className="stat-card"><span className="stat-count">{metrics.requests.avg_latency_ms.toFixed(1)}ms</span><span className="stat-label">Avg Latency</span></div>
          </div>

          {Object.keys(metrics.requests.by_endpoint).length > 0 && (
            <div className="table-wrap" style={{ marginTop: 'var(--space-8)' }}>
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
                        <Badge key={code} variant={code.startsWith('2') ? 'success' : code.startsWith('4') ? 'warning' : 'error'} style={{ marginRight: 'var(--space-2)' }}>
                          {code}: {count}
                        </Badge>
                      ))}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {tab === 'host' && metrics?.host && (
        <>
          <h2 className="section-title">CPU</h2>
          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-6)', padding: 'var(--space-6)' }}>
            <DonutGauge used={metrics.host.cpu_percent} total={100} label="CPU Usage" color="#3b82f6" size={90} />
            <div style={{ flex: 1 }}>
              <p className="stat-label" style={{ marginBottom: 'var(--space-2)' }}>CPU % — last {cpuHistory.length} samples</p>
              <Sparkline values={cpuHistory} width={240} height={48} color="#3b82f6" label="CPU sparkline" />
            </div>
          </div>

          <h2 className="section-title">Memory</h2>
          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-6)', padding: 'var(--space-6)' }}>
            <DonutGauge
              used={metrics.host.mem_used_bytes}
              total={metrics.host.mem_total_bytes}
              label="Memory"
              color="#8b5cf6"
              size={90}
            />
            <div style={{ flex: 1 }}>
              <BarGauge
                used={metrics.host.mem_used_bytes}
                total={metrics.host.mem_total_bytes}
                label="RAM"
                color="#8b5cf6"
                formatValue={fmtBytes}
              />
            </div>
          </div>

          <h2 className="section-title">Disk ( / )</h2>
          <div className="card" style={{ padding: 'var(--space-6)' }}>
            <BarGauge
              used={metrics.host.disk_used_bytes}
              total={metrics.host.disk_total_bytes}
              label="Disk"
              color="#f59e0b"
              formatValue={fmtBytes}
            />
          </div>

          <h2 className="section-title">Network</h2>
          <div className="stats-grid">
            <div className="card" style={{ padding: 'var(--space-4)' }}>
              <p className="stat-label">Bytes In (cumulative)</p>
              <p className="stat-count" style={{ fontSize: 'var(--text-lg)' }}>{fmtBytes(metrics.host.net_bytes_in)}</p>
              <Sparkline values={netInHistory} width={120} height={36} color="#22c55e" label="net in sparkline" />
            </div>
            <div className="card" style={{ padding: 'var(--space-4)' }}>
              <p className="stat-label">Bytes Out (cumulative)</p>
              <p className="stat-count" style={{ fontSize: 'var(--text-lg)' }}>{fmtBytes(metrics.host.net_bytes_out)}</p>
              <Sparkline values={netOutHistory} width={120} height={36} color="#ef4444" label="net out sparkline" />
            </div>
          </div>
        </>
      )}

      {tab === 'logs' && (
        <>
          <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
            <form onSubmit={e => { e.preventDefault(); fetchLogs(logQuery) }} className="form-row">
              <Input placeholder="Search logs..." value={logQuery} onChange={e => setLogQuery(e.target.value)} />
              <Button type="submit" size="sm">Search</Button>
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
                      <td><Badge variant={l.level === 'error' ? 'error' : l.level === 'warn' ? 'warning' : 'success'}>{l.level}</Badge></td>
                      <td className="max">{l.message}</td>
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
          <div style={{ marginBottom: 'var(--space-8)' }}>
            <Button variant="primary" size="sm" onClick={() => setShowAlertForm(!showAlertForm)}>
              {showAlertForm ? 'Cancel' : 'New Alert'}
            </Button>
          </div>
          {showAlertForm && (
            <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
              <form onSubmit={handleCreateAlert} className="fn-form">
                <Input placeholder="Name *" value={alertForm.name} onChange={e => setAlertForm(p => ({ ...p, name: e.target.value }))} required />
                <Input placeholder="Metric *" value={alertForm.metric} onChange={e => setAlertForm(p => ({ ...p, metric: e.target.value }))} required />
                <Input placeholder="Threshold" type="number" step="0.1" value={alertForm.threshold} onChange={e => setAlertForm(p => ({ ...p, threshold: +e.target.value }))} />
                <Input as="select" value={alertForm.operator} onChange={e => setAlertForm(p => ({ ...p, operator: e.target.value }))}>
                  <option value="gt">Greater Than</option>
                  <option value="lt">Less Than</option>
                  <option value="eq">Equals</option>
                </Input>
                <label className="checkbox-label">
                  <input type="checkbox" checked={alertForm.enabled} onChange={e => setAlertForm(p => ({ ...p, enabled: e.target.checked }))} />
                  Enabled
                </label>
                <Button type="submit">Create</Button>
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
                      <td><Badge variant={a.enabled ? 'success' : 'warning'}>{a.enabled ? 'Enabled' : 'Disabled'}</Badge></td>
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
