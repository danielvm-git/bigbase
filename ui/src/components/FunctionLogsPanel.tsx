import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { statusBadgeVariant } from '../components/Badge'

interface ExecutionEntry {
  id: string
  status: string
  logs: string[]
  error: string
  created_at: string
}

interface FunctionLogsPanelProps {
  functionId: string
  showMonitoringLink?: boolean
}

export function FunctionLogsPanel({ functionId, showMonitoringLink }: FunctionLogsPanelProps) {
  const [executions, setExecutions] = useState<ExecutionEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!functionId) return
    fetch(`/api/functions/${functionId}/logs`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then((data: { data: ExecutionEntry[] }) => {
        setExecutions(data.data)
        setError(null)
      })
      .catch(() => {
        setError('Failed to load execution logs')
      })
  }, [functionId])

  if (error) {
    return <p className="input-error-text" role="alert">{error}</p>
  }

  if (!executions) {
    return <div className="loading" role="status" aria-busy="true">Loading execution logs...</div>
  }

  return (
    <div>
      {showMonitoringLink && (
        <p style={{ marginBottom: 'var(--space-6)' }}>
          <Link to="/monitoring" className="btn btn-link">View all in Monitoring →</Link>
        </p>
      )}
      {executions.length === 0 ? (
        <p className="dim">No executions yet. Run this function to see logs here.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-8)' }}>
          {executions.map(exec => (
            <div key={exec.id} className="card" style={{ padding: 'var(--space-8)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)', marginBottom: 'var(--space-6)' }}>
                <span className={`badge badge-${statusBadgeVariant(exec.status)}`}>{exec.status}</span>
                <span className="dim">{exec.created_at}</span>
              </div>
              {exec.error && <p className="input-error-text">{exec.error}</p>}
              {exec.logs.length > 0 && (
                <div className="code-output">
                  <pre>{exec.logs.join('\n')}</pre>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
