import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader, Button, Input, PreviewBanner, SitesListSkeleton, BuildCachePanel } from '../components'
import { SiteCard } from '../components/SiteCard'
import { Icon } from '../components/Icon'
import { getSites, deleteSite } from '../lib/sitesData'
import { isPreviewForced, previewQuerySuffix } from '../lib/previewMode'
import type { Site } from '../types/sites'

type EnvFilter = 'all' | 'production' | 'preview'

export default function DeployPage() {
  const navigate = useNavigate()
  const [sites, setSites] = useState<Site[]>([])
  const [loading, setLoading] = useState(true)
  const [previewMode, setPreviewMode] = useState(false)
  const [filter, setFilter] = useState<EnvFilter>('all')
  const [search, setSearch] = useState('')
  const [deletingSiteId, setDeletingSiteId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    const result = await getSites()
    setSites(result.data)
    setPreviewMode(result.previewMode || isPreviewForced())
    setLoading(false)
  }, [])

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    load()
  }, [load])
  /* eslint-enable react-hooks/set-state-in-effect */

  const pq = previewQuerySuffix()

  const handleDeleteSite = async (site: Site) => {
    const name = site.name || site.full_name || site.id.slice(0, 8)
    if (!window.confirm(`Delete site "${name}"?\n\nThis will remove all deployments, domains, and logs. This cannot be undone.`)) return
    if (previewMode) {
      setSites(prev => prev.filter(s => s.id !== site.id))
      return
    }
    setDeletingSiteId(site.id)
    const result = await deleteSite(site.id)
    if (result.ok) {
      load()
    } else {
      alert(result.error || 'Delete failed')
    }
    setDeletingSiteId(null)
  }

  const shown = sites.filter(s => {
    const env = s.production_branch === 'main' ? 'production' : 'preview'
    const matchesFilter = filter === 'all' || env === filter
    const q = search.toLowerCase()
    const matchesSearch =
      !q ||
      s.name.toLowerCase().includes(q) ||
      (s.full_name || '').toLowerCase().includes(q)
    return matchesFilter && matchesSearch
  })

  if (loading) {
    return (
      <div>
        <PageHeader title="Sites" />
        <SitesListSkeleton />
      </div>
    )
  }

  return (
    <div>
      {(previewMode || isPreviewForced()) && <PreviewBanner />}

      <PageHeader title="Sites">
        <Button variant="primary" size="sm" onClick={() => navigate(`/deploy/new${pq}`)}>
          <Icon name="plus" size={16} />
          Create site
        </Button>
      </PageHeader>
      <p className="page-subtitle" style={{ marginTop: 'calc(-1 * var(--space-8))', marginBottom: 'var(--space-12)' }}>
        Deploy and host web apps straight from Git.
      </p>

      {sites.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <Icon name="rocket" size={28} />
          </div>
          <div className="empty-state-title">Create your first site</div>
          <div className="empty-state-text">
            Connect a Git repository and BigBase builds, deploys, and serves it with a live preview URL —
            auto-redeploying on every push.
          </div>
          <Button variant="primary" size="sm" onClick={() => navigate(`/deploy/new${pq}`)}>
            <Icon name="plus" size={16} />
            Create site
          </Button>
        </div>
      ) : (
        <>
          <div className="sites-toolbar">
            <div className="segmented-control" role="tablist" aria-label="Environment filter">
              {(['all', 'production', 'preview'] as const).map(f => (
                <button
                  key={f}
                  type="button"
                  role="tab"
                  aria-selected={filter === f}
                  className={filter === f ? 'segmented-control--active' : ''}
                  onClick={() => setFilter(f)}
                >
                  {f}
                </button>
              ))}
            </div>
            <div className="sites-search input-with-prefix">
              <span className="input-prefix" aria-hidden>
                <Icon name="search" size={14} />
              </span>
              <Input
                placeholder="Search sites"
                value={search}
                onChange={e => setSearch(e.target.value)}
                aria-label="Search sites"
              />
            </div>
          </div>

          {shown.length === 0 ? (
            <p className="dim">No sites match your filters.</p>
          ) : (
            <div className="site-grid">
              {shown.map(s => (
                <div key={s.id} style={{ position: 'relative' }}>
                  <SiteCard site={s} />
                  <Button
                    variant="ghost"
                    size="sm"
                    style={{
                      position: 'absolute',
                      top: 'var(--space-2)',
                      right: 'var(--space-2)',
                      opacity: 0.7,
                    }}
                    disabled={deletingSiteId === s.id}
                    aria-label={`Delete site ${s.name}`}
                    onClick={(e: React.MouseEvent) => {
                      e.preventDefault()
                      e.stopPropagation()
                      handleDeleteSite(s)
                    }}
                  >
                    {deletingSiteId === s.id ? '…' : <Icon name="trash-2" size={14} />}
                  </Button>
                </div>
              ))}
            </div>
          )}

          <BuildCachePanel />
        </>
      )}
    </div>
  )
}
