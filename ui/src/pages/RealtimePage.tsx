import { useEffect, useState } from 'react'

interface ConnectionInfo {
  user_id: number
  rooms: string[]
}

interface HubStatus {
  total_connections: number
  total_rooms: number
  connections: ConnectionInfo[]
}

export default function RealtimePage() {
  const [status, setStatus] = useState<HubStatus | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/realtime/status')
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then((data: HubStatus) => {
        setStatus(data)
        setError(null)
      })
      .catch(() => {
        setError('Failed to load realtime status')
      })
  }, [])

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Realtime</h1>
      </div>

      {error ? (
        <div className="empty-state">
          <div className="empty-state-icon" aria-hidden="true">⚠</div>
          <div className="empty-state-title">Failed to load</div>
          <div className="empty-state-text">{error}</div>
        </div>
      ) : !status ? (
        <div className="loading">
          <div className="spinner" />
          Loading realtime status...
        </div>
      ) : status.total_connections === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon" aria-hidden="true">🔌</div>
          <div className="empty-state-title">No active connections</div>
          <div className="empty-state-text">
            WebSocket connections will appear here when clients connect.
          </div>
        </div>
      ) : (
        <>
          <div style={{ display: 'flex', gap: 'var(--space-8)' }}>
            <span className="badge badge-success">
              {status.total_connections} active connections
            </span>
            <span className="badge badge-neutral">
              {status.total_rooms} rooms
            </span>
          </div>

          <div className="card" style={{ overflow: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: '2px solid var(--border-default)' }}>
                  <th style={{ textAlign: 'left', padding: 'var(--space-4) var(--space-6)', fontSize: 'var(--text-xs)', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-tertiary)', fontWeight: 600 }}>User</th>
                  <th style={{ textAlign: 'left', padding: 'var(--space-4) var(--space-6)', fontSize: 'var(--text-xs)', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-tertiary)', fontWeight: 600 }}>Channels Subscribed</th>
                  <th style={{ textAlign: 'left', padding: 'var(--space-4) var(--space-6)', fontSize: 'var(--text-xs)', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-tertiary)', fontWeight: 600 }}>Subscription Count</th>
                </tr>
              </thead>
              <tbody>
                {status.connections.map((conn, i) => (
                  <tr key={conn.user_id} style={{ borderBottom: i < status.connections.length - 1 ? '1px solid var(--border-default)' : 'none' }}>
                    <td style={{ padding: 'var(--space-6)' }}>
                      <span style={{ fontWeight: 600, color: 'var(--fg-primary)' }}>
                        User #{conn.user_id}
                      </span>
                    </td>
                    <td style={{ padding: 'var(--space-6)' }}>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
                        {conn.rooms.length === 0
                          ? <span style={{ color: 'var(--fg-tertiary)', fontStyle: 'italic' }}>No subscriptions</span>
                          : conn.rooms.map(room => (
                              <span key={room} className="tag" style={{
                                display: 'inline-block',
                                padding: 'var(--space-1) var(--space-3)',
                                borderRadius: 'var(--radius-xs)',
                                fontSize: 'var(--text-xs)',
                                fontWeight: 500,
                                background: 'var(--bg-subtle)',
                                color: 'var(--fg-secondary)',
                              }}>
                                {room}
                              </span>
                            ))
                        }
                      </div>
                    </td>
                    <td style={{ padding: 'var(--space-6)' }}>
                      <span style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        minWidth: '24px',
                        height: '24px',
                        padding: '0 var(--space-3)',
                        borderRadius: 'var(--radius-full)',
                        fontSize: 'var(--text-xs)',
                        fontWeight: 600,
                        background: 'var(--bg-accent)',
                        color: 'var(--fg-on-accent)',
                      }}>
                        {conn.rooms.length}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}
