import { useEffect, useState, useCallback } from 'react'

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

  const statusClass = (s: string) => {
    if (s === 'running') return 'status-ok'
    if (s === 'failed') return 'status-err'
    if (s === 'building') return 'status-warn'
    return ''
  }

  if (loading) return <p className="loading">Loading deployments...</p>

  return (
    <div className="page">
      <div className="page-header">
        <h1>Deployments</h1>
        <button className="refresh-btn" onClick={fetchDeployments}>Refresh</button>
        <button className="create-btn" onClick={() => setShowForm(!showForm)}>{showForm ? 'Cancel' : 'New Deployment'}</button>
      </div>
      {error && <p className="error">{error}</p>}

      {showForm && (
        <div className="card">
          <form onSubmit={handleCreate} className="inline-form">
            <select value={selectedRepoId} onChange={e => setSelectedRepoId(e.target.value)} required>
              <option value="">Select repo...</option>
              {repos.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
            </select>
            <input placeholder="Branch" value={branch} onChange={e => setBranch(e.target.value)} />
            <button type="submit" className="create-btn">Deploy</button>
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
              </tr>
            </thead>
            <tbody>
              {deployments.map(d => (
                <tr key={d.id}>
                  <td><span className={`status-badge ${statusClass(d.status)}`}>{d.status}</span></td>
                  <td><code>{d.repo_id.slice(0, 8)}</code></td>
                  <td>{d.branch}</td>
                  <td>{d.app_type || '—'}</td>
                  <td>{d.url ? <a href={d.url} target="_blank" rel="noreferrer">{d.url}</a> : '—'}</td>
                  <td><code>{d.commit_sha ? d.commit_sha.slice(0, 7) : '—'}</code></td>
                  <td>{new Date(d.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
