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
  TerminalLogViewer,
  Tabs,
  RequestLogs,
  SiteEnvVarsTab,
  SiteCacheTab,
  SiteDomainsTab,
  SiteDeployKeysTab,
  Page,
} from '../components'
import { getSite, getSiteDeployments, deleteSite, getSiteManifest, saveSiteManifest, rollbackDeployment, getRollbackEvents } from '../lib/sitesData'
import { isPreviewForced, previewQuerySuffix } from '../lib/previewMode'
import { useRequestLogs } from '../hooks/useRequestLogs'
import type { Deployment, RollbackEvent, Site } from '../types/sites'

const STATUS_STEPS = ['pending', 'building', 'deploying', 'running', 'draining', 'stopped']

interface HealthSummaryData {
  probe_count?: number
  avg_response_time_ms?: number
  first_failure_reason?: string
}

function parseHealthSummary(summary: string | undefined | null): HealthSummaryData | null {
  if (!summary) return null
  try {
    return JSON.parse(summary) as HealthSummaryData
  } catch {
    return null
  }
}

function StatusTimeline({ status, healthSummary }: { status: string; healthSummary?: string | null }) {
  const currentIdx = STATUS_STEPS.indexOf(status) // Default as fallback
  const hs = parseHealthSummary(healthSummary)
  const isTerminal = status === 'running' || status === 'failed' || status === 'stopped'
  if (currentIdx < 0) return null

  const dotStyle = (step: string, i: number): React.CSSProperties => {
    const isPast = i < currentIdx
    const isCurrent = i === currentIdx
    if (isCurrent && status === 'failed') {
      return { width: 16, height: 16, borderRadius: '50%', flexShrink: 0, background: 'var(--error)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#fff', lineHeight: '16px' }
    }
    if (isCurrent && status === 'running') {
      return { width: 16, height: 16, borderRadius: '50%', flexShrink: 0, background: 'var(--brand-500)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#fff', lineHeight: '16px' }
    }
    const bg = isPast || (isCurrent && isTerminal) ? 'var(--brand-500)' : isCurrent && step === 'building' ? 'var(--brand-500)' : (isCurrent && step === 'deploying') || (isCurrent && step === 'draining') ? 'rgba(255, 183, 0, 0.8)' : 'var(--border-default)'
    return { width: 10, height: 10, borderRadius: '50%', flexShrink: 0, background: bg }
  }

  const dotClass = (step: string, i: number): string => {
    if (i !== currentIdx) return ''
    switch (step) {
      case 'building': return 'status-dot-spin'
      case 'deploying': return 'status-dot-pulse'
      case 'draining': return 'status-dot-pulse'
      case 'running': return 'status-dot-live'
      case 'failed': return 'status-dot-failed'
      default: return ''
    }
  }

  const label = (step: string, i: number): string => {
    if (i === currentIdx && step === 'running') return 'Live'
    if (i === currentIdx && step === 'draining') return 'Draining…'
    if (i === currentIdx && step === 'deployed') return 'Deployed'
    return step
  }

  return (
    <>
    <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', padding: 'var(--space-3) 0' }}>
      {STATUS_STEPS.map((step, i) => (
        <div key={step} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', flex: 1 }}>
          <div className={dotClass(step, i)} style={dotStyle(step, i)} />
          <span style={{
            fontSize: 'var(--text-xs)', textTransform: 'capitalize', whiteSpace: 'nowrap',
            color: i <= currentIdx ? 'var(--fg-primary)' : 'var(--fg-tertiary)',
            fontWeight: i === currentIdx ? 600 : 400,
          }}>
            {label(step, i)}
            {i === currentIdx && step === 'running' && <span style={{ marginLeft: 'var(--space-1)', color: 'var(--brand-500)', fontWeight: 600, fontSize: 'var(--text-xs)' }}>●</span>}
          </span>
          {i < STATUS_STEPS.length - 1 && (
            <div style={{ flex: 1, height: 2, background: i < currentIdx ? 'var(--brand-500)' : 'var(--border-default)', minWidth: 16 }} />
          )}
        </div>
      ))}
    </div>
    {hs && hs.probe_count != null && hs.probe_count > 0 && (
      <div style={{
        display: 'flex', alignItems: 'center', gap: 'var(--space-2)',
        padding: 'var(--space-2) 0 var(--space-1)', fontSize: 'var(--text-xs)',
      }}>
        <span style={{ fontWeight: 600, marginRight: 'var(--space-1)' }}>Health:</span>
        {hs.first_failure_reason ? (
          <span style={{ color: 'var(--error)' }}>
            ✗ {hs.first_failure_reason}
            {hs.probe_count != null && <span style={{ marginLeft: 'var(--space-1)', opacity: 0.7 }}>({hs.probe_count} probes)</span>}
          </span>
        ) : (
          <span style={{ color: 'var(--brand-500)' }}>
            ✓ Passed
            {hs.avg_response_time_ms != null && <span style={{ marginLeft: 'var(--space-1)', opacity: 0.7 }}>({hs.avg_response_time_ms}ms avg)</span>}
            {hs.probe_count != null && <span style={{ marginLeft: 'var(--space-1)', opacity: 0.7 }}>{hs.probe_count} probe{hs.probe_count !== 1 ? 's' : ''}</span>}
          </span>
        )}
      </div>
    )}
    </>
  )
}

function SiteManifest({ site, latestDeployment }: { site: Site; latestDeployment?: Deployment }) {
  const [exists, setExists] = useState(false)
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [editedContent, setEditedContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saveSuccess, setSaveSuccess] = useState(false)

  const loadManifest = useCallback(async () => {
    setLoading(true)
    setError(null)
    const result = await getSiteManifest(site.id)
    setExists(result.data.exists)
    setContent(result.data.content)
    setEditedContent(result.data.content)
    setLoading(false)
  }, [site.id])

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    loadManifest()
  }, [loadManifest])
  /* eslint-enable react-hooks/set-state-in-effect */

  const handleCreate = () => {
    const appType = latestDeployment?.app_type || 'static'
    let template = `version: 1
framework: static
build:
  command: "npm run build"
start:
  command: "serve"
  port: 3000
`
    if (appType === 'node') {
      template = `version: 1
framework: node
build:
  command: "npm run build"
start:
  command: "npm run start"
  port: 3000
`
    } else if (appType === 'go') {
      template = `version: 1
framework: go
build:
  command: "go build -o app ."
start:
  command: "./app"
  port: 8080
`
    } else if (appType === 'python') {
      template = `version: 1
framework: python
build:
  command: "pip install -r requirements.txt"
start:
  command: "python app.py"
  port: 8080
`
    }
    setEditedContent(template)
    setIsEditing(true)
  }

  const handleSave = async () => {
    setSaving(true)
    setSaveError(null)
    setSaveSuccess(false)
    const res = await saveSiteManifest(site.id, editedContent)
    setSaving(false)
    if (res.ok) {
      setContent(editedContent)
      setExists(true)
      setIsEditing(false)
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } else {
      setSaveError(res.error || 'Failed to save manifest')
    }
  }

  if (loading) return <p className="dim">Loading manifest...</p>

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-4)' }}>
        <h2 className="section-title" style={{ margin: 0 }}>App Manifest (bigbase.yaml)</h2>
        {!loading && !isEditing && exists && (
          <Button variant="secondary" size="sm" onClick={() => setIsEditing(true)}>
            Edit Manifest
          </Button>
        )}
      </div>

      {error && <p className="input-error-text">{error}</p>}
      {saveError && <p className="input-error-text" style={{ marginBottom: 'var(--space-4)' }}>{saveError}</p>}
      {saveSuccess && <p style={{ color: 'var(--brand-500)', marginBottom: 'var(--space-4)', fontSize: 'var(--text-sm)' }}>Manifest saved successfully!</p>}

      {!exists && !isEditing ? (
        <Card style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
          <p className="dim" style={{ marginBottom: 'var(--space-6)' }}>
            No <code>bigbase.yaml</code> manifest file found in the repository root. Auto-detection is currently active (Framework: <strong>{latestDeployment?.app_type || 'unknown'}</strong>).
          </p>
          <Button variant="primary" size="sm" onClick={handleCreate}>
            Create bigbase.yaml
          </Button>
        </Card>
      ) : isEditing ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
          <textarea
            value={editedContent}
            onChange={(e) => setEditedContent(e.target.value)}
            style={{
              fontFamily: 'monospace',
              fontSize: 'var(--text-sm)',
              width: '100%',
              height: '350px',
              padding: 'var(--space-4)',
              background: 'var(--bg-secondary)',
              color: 'var(--fg-primary)',
              border: '1px solid var(--border-default)',
              borderRadius: 'var(--radius-md)',
              outline: 'none',
              resize: 'vertical',
            }}
          />
          <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
            <Button variant="primary" size="sm" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving...' : 'Save'}
            </Button>
            <Button variant="secondary" size="sm" onClick={() => { setIsEditing(false); setEditedContent(content); setSaveError(null); }} disabled={saving}>
              Cancel
            </Button>
          </div>
        </div>
      ) : (
        <pre
          style={{
            whiteSpace: 'pre-wrap',
            padding: 'var(--space-4)',
            background: 'var(--bg-secondary)',
            color: 'var(--fg-primary)',
            borderRadius: 'var(--radius-md)',
            fontFamily: 'monospace',
            fontSize: 'var(--text-sm)',
            border: '1px solid var(--border-default)',
            margin: 0,
          }}
        >
          {content}
        </pre>
      )}
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
  const [rollbackTarget, setRollbackTarget] = useState<Deployment | null>(null)
  const [rollbacking, setRollbacking] = useState(false)
  const [rollbackError, setRollbackError] = useState<string | null>(null)
  const [rollbackEvents, setRollbackEvents] = useState<RollbackEvent[]>([])
  const [activeTab, setActiveTab] = useState('deployments')
  const pq = previewQuerySuffix()

  const latestFromDeployments = deployments[0]

  const {
    logs, loading: reqLogsLoading, error: reqLogsError,
    pathPrefix, setPathPrefix, statusClass, setStatusClass, refresh: refreshReqLogs
  } = useRequestLogs(siteId)

  const tabs = [
    { id: 'deployments', label: 'Deployments' },
    { id: 'logs', label: 'Build Logs' },
    { id: 'request-logs', label: 'Request Logs' },
    { id: 'env-vars', label: 'Env Vars' },
    { id: 'domains', label: 'Domains' },
    { id: 'deploy-keys', label: 'Deploy Keys' },
    { id: 'cache', label: 'Cache' },
    { id: 'manifest', label: 'Manifest' },
  ]

  const load = useCallback(async () => {
    setLoading(true)
    const s = await getSite(siteId)
    setSite(s.data)
    setPreviewMode(s.previewMode || isPreviewForced())
    const d = await getSiteDeployments(siteId)
    setDeployments(d.data)
    setPreviewMode(p => p || d.previewMode)
    const evRes = await getRollbackEvents(siteId)
    if (evRes.ok) setRollbackEvents(evRes.events)
    setLoading(false)
  }, [siteId])

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => { load() }, [load])
  /* eslint-enable react-hooks/set-state-in-effect */

  useEffect(() => {
    const active = deployments.some(d => d.status === 'pending' || d.status === 'building' || d.status === 'deploying')
    if (!active || previewMode) return
    const t = setInterval(load, 2000)
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

  const handleRollbackClick = (dep: Deployment) => {
    setRollbackTarget(dep)
    setRollbackError(null)
  }

  const handleRollbackConfirm = async () => {
    if (!rollbackTarget) return
    setRollbacking(true)
    setRollbackError(null)
    const result = await rollbackDeployment(rollbackTarget.id)
    setRollbacking(false)
    setRollbackTarget(null)
    if (result.ok) {
      load()
      // Fetch updated rollback events
      const evRes = await getRollbackEvents(siteId)
      if (evRes.ok) setRollbackEvents(evRes.events)
    } else {
      setRollbackError(result.error || 'Rollback failed')
    }
  }

  const canRollback = (dep: Deployment): boolean => {
    if (dep.status !== 'running') return false
    // Can only rollback if there's at least one previous deployment
    return deployments.some(d => d.id !== dep.id && (d.status === 'stopped' || d.status === 'replaced' || d.status === 'rolled_back'))
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
    <Page className="site-detail-page">
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
              <StatusTimeline status={latest.status} healthSummary={latest.health_summary} />
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
                      <td style={{ display: 'flex', gap: 'var(--space-1)' }}>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={deletingId === d.id || d.status === 'pending' || d.status === 'building'}
                          onClick={() => handleDelete(d.id)}
                        >
                          {deletingId === d.id ? '…' : 'Delete'}
                        </Button>
                        {d.status === 'running' && canRollback(d) && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleRollbackClick(d)}
                          >
                            Rollback
                          </Button>
                        )}
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
          <TerminalLogViewer deploymentId={latestFromDeployments?.id || site?.latest_deployment?.id || ''} />
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

      {activeTab === 'env-vars' && (
        <SiteEnvVarsTab siteId={siteId} />
      )}

      {activeTab === 'domains' && (
        <SiteDomainsTab siteId={siteId} />
      )}

      {activeTab === 'deploy-keys' && (
        <SiteDeployKeysTab siteId={siteId} />
      )}

      {activeTab === 'cache' && (
        <SiteCacheTab siteId={siteId} />
      )}

      {activeTab === 'manifest' && (
        <SiteManifest site={site} latestDeployment={latest} />
      )}

      {activeTab === 'deployments' && rollbackEvents.length > 0 && (
        <div style={{ marginTop: 'var(--space-8)' }}>
          <h2 className="section-title">Rollback History</h2>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Timestamp</th>
                  <th>Rolled Back From</th>
                  <th>Rolled Back To</th>
                  <th>Details</th>
                </tr>
              </thead>
              <tbody>
                {rollbackEvents.map(ev => (
                  <tr key={ev.id}>
                    <td>{new Date(ev.created_at).toLocaleString()}</td>
                    <td><code>{ev.rolled_back_from.slice(0, 8)}</code></td>
                    <td><code>{ev.rolled_back_to.slice(0, 8)}</code></td>
                    <td>
                      Rolled back from{' '}
                      <Link to={`/deploy/deployment/${ev.rolled_back_from}${pq}`}>{ev.rolled_back_from.slice(0, 8)}</Link>
                      {' → '}
                      <Link to={`/deploy/deployment/${ev.rolled_back_to}${pq}`}>{ev.rolled_back_to.slice(0, 8)}</Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Rollback confirmation dialog */}
      {rollbackTarget && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 1000,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(0,0,0,0.5)',
        }} onClick={() => setRollbackTarget(null)}>
          <Card style={{ maxWidth: 480, width: '90vw', padding: 'var(--space-6)' }} onClick={e => e.stopPropagation()}>
            <h3 style={{ margin: '0 0 var(--space-4)' }}>Rollback Deployment?</h3>
            <p style={{ marginBottom: 'var(--space-4)', fontSize: 'var(--text-sm)' }}>
              This will stop the current live deployment and restore the previous
              successful deployment using its build artifacts. The current deployment
              will be marked as <strong>rolled_back</strong>.
            </p>
            <p className="dim" style={{ marginBottom: 'var(--space-6)', fontSize: 'var(--text-sm)' }}>
              Commit: <code>{rollbackTarget.commit_sha ? rollbackTarget.commit_sha.slice(0, 7) : '—'}</code>
              {' · '}Deployed: {new Date(rollbackTarget.created_at).toLocaleString()}
            </p>
            {rollbackError && (
              <p className="input-error-text" style={{ marginBottom: 'var(--space-4)' }}>{rollbackError}</p>
            )}
            <div style={{ display: 'flex', gap: 'var(--space-2)', justifyContent: 'flex-end' }}>
              <Button variant="secondary" size="sm" onClick={() => setRollbackTarget(null)} disabled={rollbacking}>
                Cancel
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={handleRollbackConfirm}
                disabled={rollbacking}
                style={{ background: 'var(--error)', borderColor: 'var(--error)' }}
              >
                {rollbacking ? 'Rolling back…' : 'Confirm Rollback'}
              </Button>
            </div>
          </Card>
        </div>
      )}

      {rollbackError && !rollbackTarget && (
        <p className="input-error-text" style={{ marginTop: 'var(--space-4)' }}>{rollbackError}</p>
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
    </Page>
  )
}
