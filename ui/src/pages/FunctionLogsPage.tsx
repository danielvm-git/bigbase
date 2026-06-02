import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'

interface ExecutionEntry {
  id: string
  status: string
  logs: string[]
  error: string
  created_at: string
}

export default function FunctionLogsPage() {
  const { id } = useParams<{ id: string }>()
  const [executions, setExecutions] = useState<ExecutionEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    fetch(`/api/functions/${id}/logs`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then((data: { data: ExecutionEntry[] }) => {
        setExecutions(data.data)
        setError(null)
      })
      .catch(err => {
        setError('Failed to load execution logs')
        console.error('function logs:', err)
      })
  }, [id])

  if (error) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">⚠</div>
        <div className="empty-state-title">Failed to load</div>
        <div className="empty-state-text">{error}</div>
      </div>
    )
  }

  if (!executions) {
    return (
      <div className="loading">
        <div className="spinner" />
        Loading execution logs...
      </div>
    )
  }

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Function Execution History</h1>
        <span className="badge badge-neutral">{executions.length} executions</span>
      </div>

      {executions.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">📋</div>
          <div className="empty-state-title">No executions yet</div>
          <div className="empty-state-text">Run this function to see execution logs here.</div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-8)' }}>
          {executions.map(exec => (
            <div key={exec.id} className="card" style={{ padding: 'var(--space-8)' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 'var(--space-6)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)' }}>
                  <span className={`badge ${exec.status === 'success' ? 'badge-success' : 'badge-error'}`}>
                    {exec.status}
                  </span>
                  <span style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
                    {exec.created_at}
                  </span>
                </div>
              </div>

              {exec.error && (
                <div style={{
                  padding: 'var(--space-4) var(--space-6)',
                  marginBottom: 'var(--space-6)',
                  borderRadius: 'var(--radius-xs)',
                  background: 'var(--error-bg)',
                  color: 'var(--error-fg)',
                  fontSize: 'var(--text-s)',
                  fontFamily: 'var(--font-mono)',
                }}>
                  {exec.error}
                </div>
              )}

              {exec.logs.length > 0 && (
                <div className="code-output">
                  <pre>
                    {exec.logs.map((log, i) => (
                      <div key={i} style={{ lineHeight: '1.6' }}>
                        {log.startsWith('[error]') ? (
                          <span style={{ color: '#f87171' }}>{log}</span>
                        ) : log.startsWith('[warn]') ? (
                          <span style={{ color: '#fbbf24' }}>{log}</span>
                        ) : (
                          <span>{log}</span>
                        )}
                      </div>
                    ))}
                  </pre>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
