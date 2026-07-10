import { useEffect, useState } from 'react'
import { PageHeader, Button, Input, Tabs } from '../components'

interface Repo { id: string; name: string }
interface Issue { id: string; repo_id: string; title: string; description: string; status: string; labels: string; created_at: string; updated_at: string }
interface Label { id: string; repo_id: string; name: string; color: string }
interface Board { open: Issue[]; in_progress: Issue[]; review: Issue[]; closed: Issue[] }
interface Comment { id: string; issue_id: string; content: string; created_at: string }

export default function ForgePage() {
  const [repos, setRepos] = useState<Repo[]>([])
  const [repoId, setRepoId] = useState('')
  const [tab, setTab] = useState('issues')
  const [issues, setIssues] = useState<Issue[]>([])
  const [labels, setLabels] = useState<Label[]>([])
  const [board, setBoard] = useState<Board | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ title: '', description: '', status: 'open', labels: '' })
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [commentText, setCommentText] = useState('')

  useEffect(() => {
    fetch('/api/git/repos')
      .then(r => r.json())
      .then(d => setRepos((d as { data: Repo[] }).data || []))
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (!repoId) return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true); setError('')
    Promise.all([
      fetch(`/api/forge/issues?repo_id=${repoId}`).then(r => r.json()),
      fetch(`/api/forge/labels?repo_id=${repoId}`).then(r => r.json()),
      fetch(`/api/forge/board?repo_id=${repoId}`).then(r => r.json()),
    ]).then(([iD, lD, bD]) => {
      setIssues((iD as { data: Issue[] }).data || [])
      setLabels((lD as { data: Label[] }).data || [])
      setBoard(bD as Board)
    }).catch(() => setError('failed to load'))
    .finally(() => setLoading(false))
  }, [repoId])

  const loadComments = async (issueId: string) => {
    try {
      const res = await fetch(`/api/forge/issues/${issueId}/comments`)
      if (res.ok) {
        const d = await res.json()
        setComments(d as Comment[])
      }
    } catch { /* ignore network error for comments */ }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    try {
      const res = await fetch('/api/forge/issues', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, repo_id: repoId }),
      })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'create failed'); return }
      setShowForm(false); setForm({ title: '', description: '', status: 'open', labels: '' })
      const iRes = await fetch(`/api/forge/issues?repo_id=${repoId}`)
      const iD = await iRes.json()
      setIssues((iD as { data: Issue[] }).data || [])
    } catch { /* ignore network error */ }
  }

  const handleStatusChange = async (id: string, status: string) => {
    try {
      await fetch(`/api/forge/issues/${id}`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      })
      const iRes = await fetch(`/api/forge/issues?repo_id=${repoId}`)
      const iD = await iRes.json()
      setIssues((iD as { data: Issue[] }).data || [])
      if (board) {
        const bRes = await fetch(`/api/forge/board?repo_id=${repoId}`)
        setBoard(await bRes.json() as Board)
      }
    } catch { /* ignore */ }
  }

  const handleComment = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedIssue || !commentText.trim()) return
    try {
      const res = await fetch(`/api/forge/issues/${selectedIssue.id}/comments`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: commentText }),
      })
      if (res.ok) { setCommentText(''); loadComments(selectedIssue.id) }
    } catch { /* ignore */ }
  }

  if (!repoId) return (
    <div>
      <PageHeader title="Forge" />
      <p className="dim">Select a repo to view issues.</p>
      <select value={repoId} onChange={e => setRepoId(e.target.value)} className="input" style={{ maxWidth: 240, marginTop: 'var(--space-4)' }}>
        <option value="">Select repo...</option>
        {repos.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
      </select>
    </div>
  )

  const labelColor = (name: string) => labels.find(l => l.name === name)?.color || '#888'

  if (loading) return <div className="loading">Loading...</div>

  const forgeTabs = [
    { id: 'issues', label: 'Issues' },
    { id: 'board', label: 'Board' },
  ]

  return (
    <div>
      <PageHeader title="Forge">
        <select value={repoId} onChange={e => setRepoId(e.target.value)} className="input" style={{ maxWidth: 200 }}>
          {repos.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
        <Button variant="primary" size="sm" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'New Issue'}
        </Button>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}

      <Tabs tabs={forgeTabs} active={tab} onChange={setTab} />

      {showForm && (
        <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
          <h3 style={{ marginBottom: 'var(--space-6)', fontSize: 'var(--text-m)', fontWeight: 600 }}>New Issue</h3>
          <form onSubmit={handleCreate} className="fn-form">
            <Input placeholder="Title *" value={form.title} onChange={e => setForm(p => ({ ...p, title: e.target.value }))} required />
            <Input as="textarea" placeholder="Description" value={form.description} onChange={e => setForm(p => ({ ...p, description: e.target.value }))} rows={4} />
            <Input placeholder="Labels (comma-separated)" value={form.labels} onChange={e => setForm(p => ({ ...p, labels: e.target.value }))} />
            <Button type="submit">Create</Button>
          </form>
        </div>
      )}

      {tab === 'issues' && (
        <>
          {issues.length === 0 ? <p className="dim">No issues.</p> : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Title</th>
                    <th>Status</th>
                    <th>Labels</th>
                    <th>Updated</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {issues.map(issue => (
                    <tr key={issue.id}>
                      <td>
                        <button className="btn-link" onClick={() => { setSelectedIssue(issue); loadComments(issue.id) }}>{issue.title}</button>
                      </td>
                      <td>
                        <select value={issue.status} onChange={e => handleStatusChange(issue.id, e.target.value)} className="status-select">
                          <option value="open">Open</option>
                          <option value="in_progress">In Progress</option>
                          <option value="review">Review</option>
                          <option value="closed">Closed</option>
                        </select>
                      </td>
                      <td>{issue.labels ? issue.labels.split(',').map(l => <span key={l} className="label-badge" style={{ background: labelColor(l.trim()) }}>{l.trim()}</span>) : '—'}</td>
                      <td>{new Date(issue.updated_at).toLocaleString()}</td>
                      <td>
                        <Button variant="secondary" size="sm" onClick={() => handleStatusChange(issue.id, issue.status === 'closed' ? 'open' : 'closed')}>
                          {issue.status === 'closed' ? 'Reopen' : 'Close'}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {selectedIssue && (
            <div className="card issue-detail">
              <div className="issue-detail-header">
                <h3>{selectedIssue.title}</h3>
                <Button variant="secondary" size="sm" onClick={() => setSelectedIssue(null)}>Close</Button>
              </div>
              <p className="dim">{selectedIssue.description || 'No description.'}</p>
              {comments.length > 0 && (
                <div className="comments">
                  <h4 style={{ marginBottom: 'var(--space-4)' }}>Comments</h4>
                  {comments.map(c => (
                    <div key={c.id} className="comment">
                      <p>{c.content}</p>
                      <span className="dim">{new Date(c.created_at).toLocaleString()}</span>
                    </div>
                  ))}
                </div>
              )}
              <form onSubmit={handleComment} className="form-row" style={{ marginTop: 'var(--space-4)' }}>
                <Input placeholder="Add a comment..." value={commentText} onChange={e => setCommentText(e.target.value)} />
                <Button type="submit" size="sm">Comment</Button>
              </form>
            </div>
          )}
        </>
      )}

      {tab === 'board' && board && (
        <div className="board">
          {(['open', 'in_progress', 'review', 'closed'] as const).map(col => (
            <div key={col} className="board-col">
              <h3 className="board-col-title">{col.replace('_', ' ')}</h3>
              {(board[col] || []).map(issue => (
                <div key={issue.id} className="board-card" onClick={() => { setSelectedIssue(issue); setTab('issues'); loadComments(issue.id) }}>
                  <strong>{issue.title}</strong>
                  {issue.labels && <div className="board-labels">{issue.labels.split(',').map(l => <span key={l} className="label-badge" style={{ background: labelColor(l.trim()) }}>{l.trim()}</span>)}</div>}
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
