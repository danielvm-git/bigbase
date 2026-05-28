import { useEffect, useState } from 'react'

interface Fn {
  id: string
  name: string
  runtime: string
  source: string
  trigger: string
  schedule: string
  env: string
  timeout: number
  created_at: string
}

interface RunResult {
  logs: string[]
  result: unknown
  error: string | null
}

const defaultFn = { name: '', runtime: 'javascript', source: '', trigger: 'http', schedule: '', env: '{}', timeout: 30 }

export default function FunctionsPage() {
  const [fns, setFns] = useState<Fn[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Fn | null>(null)
  const [form, setForm] = useState(defaultFn)
  const [runResult, setRunResult] = useState<Record<string, RunResult>>({})

  const fetchFns = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/functions')
      const d = await res.json()
      if (!res.ok) { setError(d.error || `error: ${res.status}`); setFns([]) }
      else { setFns((d as { data: Fn[] }).data || []) }
    } catch { setError('network error') }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchFns() }, [])

  const openCreate = () => { setEditing(null); setForm(defaultFn); setShowForm(true) }
  const openEdit = (fn: Fn) => { setEditing(fn); setForm({ name: fn.name, runtime: fn.runtime, source: fn.source, trigger: fn.trigger, schedule: fn.schedule, env: fn.env, timeout: fn.timeout }); setShowForm(true) }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    const isNew = !editing
    const url = isNew ? '/api/functions' : `/api/functions/${editing.id}`
    const method = isNew ? 'POST' : 'PUT'
    try {
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(form) })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'save failed'); return }
      setShowForm(false)
      fetchFns()
    } catch { setError('network error') }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this function?')) return
    try {
      const res = await fetch(`/api/functions/${id}`, { method: 'DELETE' })
      if (!res.ok) { const d = await res.json(); setError(d.error || 'delete failed'); return }
      setFns(prev => prev.filter(f => f.id !== id))
    } catch { setError('network error') }
  }

  const handleRun = async (id: string) => {
    setError('')
    try {
      const res = await fetch(`/api/functions/${id}/run`, { method: 'POST' })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'run failed'); return }
      setRunResult(p => ({ ...p, [id]: d as RunResult }))
    } catch { setError('network error') }
  }

  if (loading) return <p className="loading">Loading functions...</p>

  return (
    <div className="page">
      <div className="page-header">
        <h1>Functions</h1>
        <button className="refresh-btn" onClick={fetchFns}>Refresh</button>
        <button className="create-btn" onClick={openCreate}>New Function</button>
      </div>
      {error && <p className="error">{error}</p>}

      {showForm && (
        <div className="card">
          <h3>{editing ? 'Edit Function' : 'New Function'}</h3>
          <form onSubmit={handleSave} className="fn-form">
            <input placeholder="Name *" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} required />
            <select value={form.runtime} onChange={e => setForm(p => ({ ...p, runtime: e.target.value }))}>
              <option value="javascript">JavaScript</option>
            </select>
            <select value={form.trigger} onChange={e => setForm(p => ({ ...p, trigger: e.target.value }))}>
              <option value="http">HTTP</option>
              <option value="schedule">Schedule</option>
              <option value="event">Event</option>
            </select>
            {form.trigger === 'schedule' && <input placeholder="Cron schedule" value={form.schedule} onChange={e => setForm(p => ({ ...p, schedule: e.target.value }))} />}
            <input placeholder='Env JSON ({"KEY":"val"})' value={form.env} onChange={e => setForm(p => ({ ...p, env: e.target.value }))} />
            <input placeholder="Timeout (seconds)" type="number" value={form.timeout} onChange={e => setForm(p => ({ ...p, timeout: +e.target.value }))} />
            <textarea placeholder="Source code *" value={form.source} onChange={e => setForm(p => ({ ...p, source: e.target.value }))} required rows={8} className="code-textarea" />
            <div className="form-actions">
              <button type="submit" className="create-btn">{editing ? 'Update' : 'Create'}</button>
              <button type="button" className="refresh-btn" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </form>
        </div>
      )}

      {fns.length === 0 && !error && <p className="dim">No functions yet.</p>}
      {fns.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Runtime</th>
                <th>Trigger</th>
                <th>Timeout</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {fns.map(fn => (
                <tr key={fn.id}>
                  <td><code>{fn.name}</code></td>
                  <td>{fn.runtime}</td>
                  <td>{fn.trigger}</td>
                  <td>{fn.timeout}s</td>
                  <td>{new Date(fn.created_at).toLocaleString()}</td>
                  <td className="actions-cell">
                    <button className="action-btn" onClick={() => handleRun(fn.id)}>Run</button>
                    <button className="action-btn" onClick={() => openEdit(fn)}>Edit</button>
                    <button className="delete-btn" onClick={() => handleDelete(fn.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {Object.entries(runResult).map(([id, result]) => (
        <div key={id} className="card run-result">
          <h3>Run Result — <code>{fns.find(f => f.id === id)?.name || id}</code></h3>
          {result.logs?.length > 0 && (
            <div className="log-output">
              {result.logs.map((l, i) => <pre key={i}>{l}</pre>)}
            </div>
          )}
          {result.error && <p className="error">{result.error}</p>}
          {result.result !== undefined && (
            <div className="log-output">
              <pre>{JSON.stringify(result.result, null, 2)}</pre>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
