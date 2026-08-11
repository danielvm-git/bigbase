import { useState, useEffect } from 'react'
import { Button, Card } from '.'
import { getSiteCache, clearSiteCache } from '../lib/sitesData'
import { fmtBytes } from '../lib/format'
import type { SiteCacheStatus } from '../types/sites'

export function SiteCacheTab({ siteId, id }: { siteId: string; id?: string }) {
  const [status, setStatus] = useState<SiteCacheStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [clearing, setClearing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    const res = await getSiteCache(siteId)
    setStatus(res.status)
    if (!res.ok) setError('Could not load cache status. Check that the server is reachable.')
    setLoading(false)
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect, react-hooks/exhaustive-deps
  useEffect(() => { void load() }, [siteId])

  const handleClear = async () => {
    if (!window.confirm('Clear the build cache for this site? The next deploy will rebuild dependencies from scratch.')) return
    setClearing(true)
    setError(null)
    const res = await clearSiteCache(siteId)
    setClearing(false)
    if (!res.ok) { setError(res.error ?? 'Failed to clear cache'); return }
    void load()
  }

  if (loading) return <p className="dim" role="status" aria-busy="true">Loading…</p>

  const entries = status?.entries ?? []

  return (
    <div id={id ?? 'panel-cache'} role="tabpanel" aria-labelledby="tab-cache">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-4)' }}>
        <h2 className="section-title" style={{ margin: 0 }}>Build Cache</h2>
        {entries.length > 0 && (
          <Button variant="secondary" size="sm" onClick={handleClear} disabled={clearing} style={{ color: 'var(--error)' }}>
            {clearing ? 'Clearing…' : 'Clear cache'}
          </Button>
        )}
      </div>

      {error && <p className="input-error-text" role="alert" style={{ marginBottom: 'var(--space-3)' }}>{error}</p>}

      {entries.length === 0 ? (
        <Card style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
          <p className="dim">
            No cached dependencies yet. The build cache is populated after the first successful deploy that installs dependencies.
          </p>
        </Card>
      ) : (
        <>
          <div style={{ display: 'flex', gap: 'var(--space-6)', marginBottom: 'var(--space-4)' }}>
            <div>
              <span className="dim" style={{ fontSize: 'var(--text-xs)' }}>Cache size</span>
              <p style={{ margin: 0, fontWeight: 600 }}>{fmtBytes(status?.total_size_bytes ?? 0)}</p>
            </div>
            <div>
              <span className="dim" style={{ fontSize: 'var(--text-xs)' }}>Total hits</span>
              <p style={{ margin: 0, fontWeight: 600 }}>{status?.total_hits ?? 0}</p>
            </div>
            <div>
              <span className="dim" style={{ fontSize: 'var(--text-xs)' }}>Entries</span>
              <p style={{ margin: 0, fontWeight: 600 }}>{entries.length}</p>
            </div>
          </div>

          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Key</th>
                  <th>Branch</th>
                  <th>Size</th>
                  <th style={{ textAlign: 'center' }}>Hits</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {entries.map(e => (
                  <tr key={e.key}>
                    <td><code style={{ fontSize: 'var(--text-sm)' }}>{e.key.slice(0, 12)}</code></td>
                    <td>{e.branch || '—'}</td>
                    <td>{fmtBytes(e.size)}</td>
                    <td style={{ textAlign: 'center' }}>{e.hit_count}</td>
                    <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
                      {new Date(e.created_at).toLocaleDateString()}
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
