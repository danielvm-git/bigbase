import { useEffect, useState } from 'react'

interface Repo {
  id: string
  name: string
  owner_id: number
  private: boolean
  default_branch: string
  description: string
  created_at: string
}

export default function GitReposPage() {
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', private: false })

  const fetchRepos = async () => {
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/api/git/repos')
      const d = await res.json()
      if (!res.ok) { setError(d.error || `error: ${res.status}`); setRepos([]) }
      else { setRepos((d as { data: Repo[] }).data || []) }
    } catch { setError('network error') }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchRepos() }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/git/repos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (!res.ok) { const d = await res.json(); setError(d.error || 'create failed'); return }
      setShowForm(false)
      setForm({ name: '', description: '', private: false })
      fetchRepos()
    } catch { setError('network error') }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this repo?')) return
    try {
      const res = await fetch(`/api/git/repos/${id}`, { method: 'DELETE' })
      if (!res.ok) { const d = await res.json(); setError(d.error || 'delete failed'); return }
      setRepos(prev => prev.filter(r => r.id !== id))
    } catch { setError('network error') }
  }

  if (loading) return <p className="loading">Loading repos...</p>

  return (
    <div className="page">
      <div className="page-header">
        <h1>Git Repos</h1>
        <button className="refresh-btn" onClick={fetchRepos}>Refresh</button>
        <button className="create-btn" onClick={() => setShowForm(!showForm)}>{showForm ? 'Cancel' : 'New Repo'}</button>
      </div>
      {error && <p className="error">{error}</p>}

      {showForm && (
        <div className="card">
          <form onSubmit={handleCreate} className="inline-form">
            <input placeholder="Name *" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} required />
            <input placeholder="Description" value={form.description} onChange={e => setForm(p => ({ ...p, description: e.target.value }))} />
            <label className="checkbox-label">
              <input type="checkbox" checked={form.private} onChange={e => setForm(p => ({ ...p, private: e.target.checked }))} />
              Private
            </label>
            <button type="submit" className="create-btn">Create</button>
          </form>
        </div>
      )}

      {repos.length === 0 && !error && <p className="dim">No repos yet.</p>}
      {repos.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Branch</th>
                <th>Description</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {repos.map(r => (
                <tr key={r.id}>
                  <td><code>{r.name}</code></td>
                  <td>{r.default_branch}</td>
                  <td className="dim">{r.description || '—'}</td>
                  <td>{new Date(r.created_at).toLocaleString()}</td>
                  <td>
                    <button className="delete-btn" onClick={() => handleDelete(r.id)}>Delete</button>
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
