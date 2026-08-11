import { Badge, Button } from './'
import type { RequestLogEntry } from '../hooks/useRequestLogs'

interface RequestLogsProps {
  logs: RequestLogEntry[]
  loading?: boolean
  error?: string | null
  pathPrefix: string
  onPathPrefixChange: (val: string) => void
  statusClass: string
  onStatusClassChange: (val: string) => void
  onRefresh: () => void
}

export function RequestLogs({
  logs,
  loading,
  error,
  pathPrefix,
  onPathPrefixChange,
  statusClass,
  onStatusClassChange,
  onRefresh
}: RequestLogsProps) {
  if (error) {
    return (
      <div className="card-error" role="alert" style={{ padding: 'var(--space-4)' }}>
        <p className="input-error-text">{error}</p>
      </div>
    )
  }

  return (
    <div className="request-logs">
      <div className="toolbar" style={{ display: 'flex', gap: 'var(--space-4)', marginBottom: 'var(--space-4)', alignItems: 'center' }}>
        <input
          type="text"
          placeholder="Filter by path prefix..."
          className="input"
          aria-label="Filter by path prefix"
          style={{ flex: 1 }}
          value={pathPrefix}
          onChange={(e) => onPathPrefixChange(e.target.value)}
        />
        <select
          className="input"
          aria-label="Filter by status class"
          style={{ width: '120px' }}
          value={statusClass}
          onChange={(e) => onStatusClassChange(e.target.value)}
        >
          <option value="">All Status</option>
          <option value="2xx">2xx OK</option>
          <option value="4xx">4xx Client Error</option>
          <option value="5xx">5xx Server Error</option>
        </select>
        <Button variant="secondary" size="sm" onClick={onRefresh} disabled={loading}>
          {loading ? '…' : 'Refresh'}
        </Button>
      </div>

      <div className="table-wrap">
        <table>
          <caption className="visually-hidden">Request logs</caption>
          <thead>
            <tr>
              <th scope="col">Method</th>
              <th scope="col">Path</th>
              <th scope="col">Status</th>
              <th scope="col">Duration</th>
              <th scope="col">Timestamp</th>
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 && loading && (
              <tr>
                <td colSpan={5} className="dim" style={{ textAlign: 'center', padding: 'var(--space-8)' }} role="status" aria-busy="true">
                  Loading request logs…
                </td>
              </tr>
            )}
            {logs.length === 0 && !loading && (
              <tr>
                <td colSpan={5} className="dim" style={{ textAlign: 'center', padding: 'var(--space-8)' }}>
                  No request logs found.
                </td>
              </tr>
            )}
            {logs.map((l) => (
              <tr key={l.id}>
                <td><strong>{l.method}</strong></td>
                <td><code>{l.path}</code></td>
                <td>
                  <Badge variant={l.status >= 500 ? 'error' : l.status >= 400 ? 'warning' : 'success'}>
                    {l.status}
                  </Badge>
                </td>
                <td className="dim">{l.duration_ms}ms</td>
                <td className="dim" style={{ fontSize: 'var(--text-xs)' }}>
                  {new Date(l.created_at).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
