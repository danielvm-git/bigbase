import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader, Button, Input, Badge, statusBadgeVariant } from '../components'

interface Deployment {
  id: string
  repo_id: string
  branch: string
  commit_sha: string
  status: string
  url: string
  port: number
  app_type: string
  created_at: string
}

interface Repo {
  id: string
  name: string
}

export default function DeployPage() {
  const navigate = useNavigate()
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [selectedRepoId, setSelectedRepoId] = useState('')
  const [branch, setBranch] = useState('main')

  const fetchDeployments = useCallback(async () => {
    try {
      const res = await fetch('/api/deploy')
      const d = await res.json()
      if (!res.ok) { setError(d.error || `error: ${res.status}`) }
      else { setDeployments((d as { data: Deployment[] }).data || []) }
    } catch { setError('network error') }
    finally { setLoading(false) }
  }, [])

  const fetchRepos = useCallback(async () => {
    try {
      const res = await fetch('/api/git/repos')
      const d = await res.json()
      if (res.ok) { setRepos((d as { data: Repo[] }).data || []) }
    } catch {}
  }, [])

  useEffect(() => {
    fetchDeployments()
    fetchRepos()
  }, [fetchDeployments, fetchRepos])

  useEffect(() => {
    const hasActive = deployments.some(d => d.status === 'pending' || d.status === 'building')
    if (!hasActive) return
    const timer = setInterval(fetchDeployments, 3000)
    return () => clearInterval(timer)
  }, [deployments, fetchDeployments])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedRepoId) return
    setError('')
    try {
      const res = await fetch('/api/deploy', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_id: selectedRepoId, branch }),
      })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'create failed'); return }
      setShowForm(false)
      setSelectedRepoId('')
      setBranch('main')
      fetchDeployments()
    } catch { setError('network error') }
  }

  if (loading) return <div className="loading">Loading deployments...</div>

  return (
    <div>
      <PageHeader title="Deployments">
        <Button variant="secondary" size="sm" onClick={fetchDeployments}>Refresh</Button>
        <Button variant="primary" size="sm" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'New Deployment'}
        </Button>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}

      {showForm && (
        <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
          <form onSubmit={handleCreate} className="form-row">
            <select className="input" value={selectedRepoId} onChange={e => setSelectedRepoId(e.target.value)} required style={{ flex: 1, minWidth: 140 }}>
              <option value="">Select repo...</option>
              {repos.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
            </select>
            <Input placeholder="Branch" value={branch} onChange={e => setBranch(e.target.value)} />
            <Button type="submit" size="sm">Deploy</Button>
          </form>
        </div>
      )}

      {deployments.length === 0 && !error && <p className="dim">No deployments yet.</p>}
      {deployments.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>Repo</th>
                <th>Branch</th>
                <th>Type</th>
                <th>URL</th>
                <th>Commit</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {deployments.map(d => (
                <tr key={d.id}>
                  <td><Badge variant={statusBadgeVariant(d.status)}>{d.status}</Badge></td>
                  <td><code>{d.repo_id.slice(0, 8)}</code></td>
                  <td>{d.branch}</td>
                  <td>{d.app_type || '—'}</td>
                  <td>{d.url ? <a href={d.url} target="_blank" rel="noreferrer">{d.url}</a> : '—'}</td>
                  <td><code>{d.commit_sha ? d.commit_sha.slice(0, 7) : '—'}</code></td>
                  <td>{new Date(d.created_at).toLocaleString()}</td>
                  <td>
                    <Button variant="secondary" size="sm" onClick={() => navigate(`/deploy/${d.id}`)}>Logs</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
