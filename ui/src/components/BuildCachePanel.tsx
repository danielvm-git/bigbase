import { useState, useEffect } from 'react'
import { Button, Card, CardHeader } from '.'
import { getCacheStats, clearAllCache, pruneCache, setCacheMaxSize } from '../lib/sitesData'
import { fmtBytes } from '../lib/format'
import type { CacheStats } from '../types/sites'

const GIB = 1024 * 1024 * 1024
const PRUNE_DAYS = 7

export function BuildCachePanel() {
  const [stats, setStats] = useState<CacheStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [maxGB, setMaxGB] = useState('')

  const load = async () => {
    setLoading(true)
    const s = await getCacheStats()
    setStats(s)
    setMaxGB(s.max_size_bytes ? String(Math.round((s.max_size_bytes / GIB) * 10) / 10) : '')
    setLoading(false)
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { void load() }, [])

  const run = async (fn: () => Promise<{ ok: boolean; error?: string }>, successMsg: string) => {
    setBusy(true); setError(null); setNotice(null)
    const res = await fn()
    setBusy(false)
    if (!res.ok) { setError(res.error ?? 'Operation failed'); return }
    setNotice(successMsg)
    void load()
  }

  const handleClearAll = () => {
    if (!window.confirm('Clear ALL build caches across every site? Each site will rebuild dependencies from scratch on its next deploy.')) return
    void run(clearAllCache, 'All caches cleared.')
  }

  const handlePrune = () => {
    void run(async () => {
      const res = await pruneCache(PRUNE_DAYS)
      return res
    }, `Pruned entries older than ${PRUNE_DAYS} days.`)
  }

  const handleSaveMaxSize = () => {
    const gb = parseFloat(maxGB)
    if (!Number.isFinite(gb) || gb <= 0) { setError('Max size must be a positive number of GB'); return }
    void run(() => setCacheMaxSize(Math.round(gb * GIB)), 'Max cache size updated.')
  }

  if (loading) return null

  const usedPct = stats && stats.max_size_bytes > 0
    ? Math.min(100, Math.round((stats.total_size_bytes / stats.max_size_bytes) * 100))
    : 0

  return (
    <Card style={{ marginTop: 'var(--space-12)' }}>
      <CardHeader title="Build Cache" />
      <p className="dim" style={{ marginBottom: 'var(--space-4)' }}>
        Cached dependencies (node_modules) skip <code>npm install</code> when a lockfile is unchanged.
      </p>

      <div style={{ display: 'flex', gap: 'var(--space-6)', marginBottom: 'var(--space-4)', flexWrap: 'wrap' }}>
        <div>
          <span className="dim" style={{ fontSize: 'var(--text-xs)' }}>Usage</span>
          <p style={{ margin: 0, fontWeight: 600 }}>
            {fmtBytes(stats?.total_size_bytes ?? 0)} / {fmtBytes(stats?.max_size_bytes ?? 0)} ({usedPct}%)
          </p>
        </div>
        <div>
          <span className="dim" style={{ fontSize: 'var(--text-xs)' }}>Entries</span>
          <p style={{ margin: 0, fontWeight: 600 }}>{stats?.total_entries ?? 0}</p>
        </div>
      </div>

      {error && <p className="input-error-text" style={{ marginBottom: 'var(--space-3)' }}>{error}</p>}
      {notice && <p style={{ color: 'var(--brand-500)', marginBottom: 'var(--space-3)', fontSize: 'var(--text-sm)' }}>{notice}</p>}

      <div style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center', flexWrap: 'wrap' }}>
        <Button variant="secondary" size="sm" onClick={handlePrune} disabled={busy}>
          {busy ? '…' : `Prune >${PRUNE_DAYS}d`}
        </Button>
        <Button variant="secondary" size="sm" onClick={handleClearAll} disabled={busy} style={{ color: 'var(--error)' }}>
          Clear all
        </Button>
        <div style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center', marginLeft: 'auto' }}>
          <label className="dim" style={{ fontSize: 'var(--text-sm)' }}>Max size (GB)</label>
          <input
            className="input"
            type="number"
            min="0.1"
            step="0.1"
            value={maxGB}
            onChange={e => setMaxGB(e.target.value)}
            disabled={busy}
            style={{ width: 90 }}
            aria-label="Max cache size in GB"
          />
          <Button variant="primary" size="sm" onClick={handleSaveMaxSize} disabled={busy}>Save</Button>
        </div>
      </div>
    </Card>
  )
}
