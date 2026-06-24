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
} from '../components'
import { getSite, getSiteDeployments, deleteSite, getSiteManifest, saveSiteManifest } from '../lib/sitesData'
import { isPreviewForced, previewQuerySuffix } from '../lib/previewMode'
import { useRequestLogs } from '../hooks/useRequestLogs'
import type { Deployment, Site } from '../types/sites'

const STATUS_STEPS = ['pending', 'building', 'deploying', 'running']

function StatusTimeline({ status }: { status: string }) {
  const currentIdx = STATUS_STEPS.indexOf(status)
  if (currentIdx < 0) return null
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', padding: 'var(--space-3) 0' }}>
      {STATUS_STEPS.map((step, i) => (
        <div key={step} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', flex: 1 }}>
          <div style={{
            width: 10, height: 10, borderRadius: '50%', flexShrink: 0,
            background: i <= currentIdx ? (status === 'failed' && i === currentIdx ? 'var(--error)' : 'var(--brand-500)') : 'var(--border-default)',
            transition: 'background var(--duration-medium) var(--ease-standard)',
          }} />
          <span style={{
            fontSize: 'var(--text-xs)', textTransform: 'capitalize', whiteSpace: 'nowrap',
            color: i <= currentIdx ? 'var(--fg-primary)' : 'var(--fg-tertiary)',
            fontWeight: i === currentIdx ? 600 : 400,
          }}>{step}</span>
          {i < STATUS_STEPS.length - 1 && (
            <div style={{ flex: 1, height: 2, background: i < currentIdx ? 'var(--brand-500)' : 'var(--border-default)', minWidth: 16 }} />
          )}
        </div>
      ))}
    </div>
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

  useEffect(() => {
    loadManifest()
  }, [loadManifest])

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
    setLoading(false)
  }, [siteId])

  useEffect(() => { load() }, [load])

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
              <StatusTimeline status={latest.status} />
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

      {activeTab === 'manifest' && (
        <SiteManifest site={site} latestDeployment={latest} />
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
