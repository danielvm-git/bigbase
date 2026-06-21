import { useEffect, useState, useCallback } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  PageHeader,
  Button,
  Card,
  CardHeader,
  Badge,
  statusBadgeVariant,
  PreviewBanner,
  SitesListSkeleton,
  BuildLogs,
  Tabs,
  RequestLogs,
} from '../components'
import { getSite, getSiteDeployments, deleteSite } from '../lib/sitesData'
import { isPreviewForced, previewQuerySuffix } from '../lib/previewMode'
import { useBuildLogs } from '../hooks/useBuildLogs'
import { useRequestLogs } from '../hooks/useRequestLogs'
import type { Deployment, Site } from '../types/sites'

// Visual states for the status timeline dots
function statusColor(status: string): string {
  if (status === 'live' || status === 'running') return 'var(--success)'
  if (status === 'failed') return 'var(--error)'
  if (status === 'deploying') return 'var(--warning)'
  if (status === 'building') return 'var(--brand-500)'
  return 'var(--border-default)' // pending
}

function statusIndicator(status: string): string {
  if (status === 'building') return 'var(--brand-500)'
  if (status === 'deploying') return 'var(--warning)'
  return statusColor(status)
}

const STATUS_STEPS = ['pending', 'building', 'deploying', 'running', 'live']

function StatusTimeline({ status, history }: { status: string; history?: Array<{ from: string; to: string; timestamp: string }> }) {
  const currentIdx = STATUS_STEPS.indexOf(status)
  if (currentIdx < 0 && !history) return null

  // If we have history, render it as a chronological list
  if (history && history.length > 0) {
    return (
      <div style={{ padding: 'var(--space-3) 0' }}>
        {history.map((tr, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', marginBottom: 'var(--space-2)' }}>
            <div style={{
              width: 10, height: 10, borderRadius: '50%', flexShrink: 0,
              background: statusColor(tr.to),
              animation: tr.to === 'building' || tr.to === 'deploying' ? 'status-pulse 1.5s ease-in-out infinite' : 'none',
            }} />
            <div style={{ fontSize: 'var(--text-xs)', display: 'flex', gap: 'var(--space-2)', alignItems: 'baseline' }}>
              <span style={{ fontWeight: i === history.length - 1 ? 600 : 400, textTransform: 'capitalize' }}>{tr.to}</span>
              <span style={{ color: 'var(--fg-tertiary)' }}>
                {new Date(tr.timestamp).toLocaleTimeString()}
              </span>
            </div>
          </div>
        ))}
      </div>
    )
  }

  // Fall back to step-based timeline for deployments without history
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', padding: 'var(--space-3) 0' }}>
      {STATUS_STEPS.map((step, i) => {
        const dotColor = i <= currentIdx ? statusIndicator(status) : 'var(--border-default)'
        const labelColor = i <= currentIdx ? 'var(--fg-primary)' : 'var(--fg-tertiary)'
        const isCurrent = i === currentIdx
        return (
          <div key={step} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', flex: 1 }}>
            <div style={{
              width: 10, height: 10, borderRadius: '50%', flexShrink: 0,
              background: dotColor,
              transition: 'background var(--duration-medium) var(--ease-standard)',
              animation: (step === 'building' || step === 'deploying') && isCurrent ? 'status-pulse 1.5s ease-in-out infinite' : 'none',
            }}>
              {(step === 'live' || step === 'running') && isCurrent && (
                <span style={{ position: 'absolute', fontSize: '8px' }}>✓</span>
              )}
              {status === 'failed' && isCurrent && (
                <span style={{ position: 'absolute', fontSize: '8px' }}>✗</span>
              )}
            </div>
            <span style={{
              fontSize: 'var(--text-xs)', textTransform: 'capitalize', whiteSpace: 'nowrap',
              color: labelColor,
              fontWeight: isCurrent ? 600 : 400,
            }}>{step}</span>
            {i < STATUS_STEPS.length - 1 && (
              <div style={{ flex: 1, height: 2, background: i < currentIdx ? statusIndicator(status) : 'var(--border-default)', minWidth: 16 }} />
            )}
          </div>
        )
      })}
    </div>
  )
}

export default function SiteDetailPage() {
  const { siteId = '' } = useParams()
  const [site, setSite] = useState<Site | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [loading, setLoading] = useState(true)
  const [previewMode, setPreviewMode] = useState(false)
  const [redeployError, setRedeployError] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deletingSite, setDeletingSite] = useState(false)
  const [deleteSiteError, setDeleteSiteError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('deployments')
  const pq = previewQuerySuffix()

  const latestFromDeployments = deployments[0]
  const { lines, loading: logsLoading, error: logsError } = useBuildLogs(latestFromDeployments?.id || site?.latest_deployment?.id || '')

  const {
    logs, loading: reqLogsLoading, error: reqLogsError,
    pathPrefix, setPathPrefix, statusClass, setStatusClass, refresh: refreshReqLogs
  } = useRequestLogs(siteId)

  const tabs = [
    { id: 'deployments', label: 'Deployments' },
    { id: 'logs', label: 'Build Logs' },
    { id: 'request-logs', label: 'Request Logs' },
  ]

  const load = useCallback(async () => {
    setLoading(true)
    const s = await getSite(siteId)
    setSite(s.data)
    setPreviewMode(s.previewMode || isPreviewForced())
    const d = await getSiteDeployments(siteId)
    setDeployments(d.data)
    setPreviewMode(p => p || d.previewMode)
    setLoading(false)
  }, [siteId])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const active = deployments.some(d =>
      d.status === 'pending' || d.status === 'building' || d.status === 'deploying'
    )
    if (!active || previewMode) return
    const t = setInterval(load, 3000)
    return () => clearInterval(t)
  }, [deployments, previewMode, load])

  const handleDelete = async (id: string) => {
    if (!window.confirm('Delete this deployment? This cannot be undone.')) return
    if (previewMode) {
      setDeployments(prev => prev.filter(d => d.id !== id))
      return
    }
    setDeletingId(id)
    setDeleteError(null)
    try {
      const res = await fetch(`/api/deployments/${id}`, { method: 'DELETE' })
      if (!res.ok) {
        const j = await res.json()
        setDeleteError(j.error ?? `Delete failed (HTTP ${res.status})`)
      } else {
        load()
      }
    } finally {
      setDeletingId(null)
    }
  }

  const handleDeleteSite = async () => {
    const name = site?.name || ''
    if (!window.confirm(`Delete site "${name}"?\n\nThis will remove all deployments, domains, and logs. This cannot be undone.`)) return
    if (previewMode) {
      window.location.href = `/admin/#/deploy${pq}`
      return
    }
    setDeletingSite(true)
    setDeleteSiteError(null)
    const result = await deleteSite(site!.id)
    if (result.ok) {
      window.location.href = `/admin/#/deploy${pq}`
    } else {
      setDeleteSiteError(result.error || 'Delete failed')
      setDeletingSite(false)
    }
  }

  const handleRedeploy = async () => {
    if (!site || previewMode) return
    setRedeployError(null)
    try {
      const res = await fetch(`/api/sites/${site.id}/deploy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ branch: site.production_branch }),
      })
      if (!res.ok) {
        const fallback = await fetch('/api/deploy', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ repo_id: site.git_repo_id, branch: site.production_branch }),
        })
        if (!fallback.ok) {
          setRedeployError(`Redeploy failed (HTTP ${fallback.status})`)
          return
        }
      }
      load()
    } catch {
      setRedeployError('Redeploy failed — network error')
    }
  }

  if (loading) return <SitesListSkeleton />

  if (!site) {
    return (
      <div>
        <PageHeader title="Site not found" />
        <Link to={`/deploy${pq}`}>Back to sites</Link>
      </div>
    )
  }

  const latest = deployments[0] ?? site.latest_deployment

  return (
    <div>
      <PageHeader title={site.name}>
        <Button variant="secondary" size="sm" onClick={() => load()}>Refresh</Button>
        <Button variant="primary" size="sm" onClick={handleRedeploy}>Redeploy</Button>
      </PageHeader>

      {redeployError && (
        <p className="input-error-text" style={{ marginBottom: 'var(--space-6)' }}>{redeployError}</p>
      )}

      {(previewMode || isPreviewForced()) && <PreviewBanner />}

      <p className="section-subtitle">
        <code>{site.full_name}</code> · branch <strong>{site.production_branch}</strong>
      </p>

      <div className="dash-grid" style={{ marginBottom: 'var(--space-12)' }}>
        <Card>
          <CardHeader title="Status" />
          {latest ? (
            <>
              <div style={{ marginBottom: 'var(--space-4)' }}>
                <Badge variant={statusBadgeVariant(latest.status)}>{latest.status}</Badge>
                {site.production_branch && <code style={{ marginLeft: 'var(--space-4)' }}>{site.production_branch}</code>}
              </div>
              <StatusTimeline status={latest.status} history={latest.status_history} />
              {latest.url && (
                <p style={{ marginTop: 'var(--space-6)' }}>
                  <Button as="a" href={latest.url} target="_blank" rel="noreferrer" variant="primary" size="sm">
                    Open app
                  </Button>
                </p>
              )}
            </>
          ) : (
            <p className="dim">No deployments yet.</p>
          )}
        </Card>
        <Card>
          <CardHeader title="Configuration" />
          <p className="dim">Root: <code>{site.root_path}</code></p>
          <p className="dim">Type: {latest?.app_type || '—'}</p>
        </Card>
      </div>

      <Tabs tabs={tabs} active={activeTab} onChange={setActiveTab} />

      {activeTab === 'deployments' && (
        <>
          <h2 className="section-title">Deployment History</h2>
          {deployments.length === 0 && <p className="dim">No deployment history.</p>}
          {deployments.length > 0 && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Status</th>
                    <th>Branch</th>
                    <th>Commit</th>
                    <th>URL</th>
                    <th>Created</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {deployments.map(d => (
                    <tr key={d.id}>
                      <td><Badge variant={statusBadgeVariant(d.status)}>{d.status}</Badge></td>
                      <td>{d.branch}</td>
                      <td><code>{d.commit_sha ? d.commit_sha.slice(0, 7) : '—'}</code></td>
                      <td>
                        {d.url ? (
                          <a href={d.url} target="_blank" rel="noreferrer">{d.url}</a>
                        ) : '—'}
                      </td>
                      <td>{new Date(d.created_at).toLocaleString()}</td>
                      <td>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={deletingId === d.id || d.status === 'pending' || d.status === 'building'}
                          onClick={() => handleDelete(d.id)}
                        >
                          {deletingId === d.id ? '…' : 'Delete'}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {activeTab === 'logs' && (
        <div>
          <h2 className="section-title">Build Logs</h2>
          <BuildLogs lines={lines} loading={logsLoading} error={logsError} />
        </div>
      )}

      {activeTab === 'request-logs' && (
        <div>
          <h2 className="section-title">Request Logs</h2>
          <RequestLogs
            logs={logs}
            loading={reqLogsLoading}
            error={reqLogsError}
            pathPrefix={pathPrefix}
            onPathPrefixChange={setPathPrefix}
            statusClass={statusClass}
            onStatusClassChange={setStatusClass}
            onRefresh={refreshReqLogs}
          />
        </div>
      )}

      {deleteError && (
        <p className="input-error-text" style={{ marginTop: 'var(--space-4)' }}>{deleteError}</p>
      )}

      <Card style={{ marginTop: 'var(--space-12)', borderColor: 'var(--error)' }}>
        <CardHeader title="Danger Zone" />
        <p className="dim" style={{ marginBottom: 'var(--space-6)' }}>
          Permanently delete this site and all associated deployments, domains, and logs.
          This cannot be undone.
        </p>
        {deleteSiteError && (
          <p className="input-error-text" style={{ marginBottom: 'var(--space-4)' }}>{deleteSiteError}</p>
        )}
        <Button
          variant="primary"
          size="sm"
          disabled={deletingSite}
          onClick={handleDeleteSite}
          style={{ background: 'var(--error)', borderColor: 'var(--error)' }}
        >
          {deletingSite ? 'Deleting…' : `Delete ${site.name}`}
        </Button>
      </Card>

      <p style={{ marginTop: 'var(--space-12)' }}>
        <Link to={`/deploy${pq}`}>← All sites</Link>
      </p>
    </div>
  )
}
