import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader, Button, Input } from '../components'

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
  const navigate = useNavigate()
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

  if (loading) return <div className="loading">Loading functions...</div>

  return (
    <div>
      <PageHeader title="Functions">
        <Button variant="secondary" size="sm" onClick={fetchFns}>Refresh</Button>
        <Button variant="primary" size="sm" onClick={openCreate}>New Function</Button>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}

      {showForm && (
        <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
          <h3 style={{ marginBottom: 'var(--space-6)', fontSize: 'var(--text-m)', fontWeight: 600 }}>
            {editing ? 'Edit Function' : 'New Function'}
          </h3>
          <form onSubmit={handleSave} className="fn-form">
            <Input placeholder="Name *" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} required />
            <Input as="select" value={form.runtime} onChange={e => setForm(p => ({ ...p, runtime: e.target.value }))}>
              <option value="javascript">JavaScript</option>
            </Input>
            <Input as="select" value={form.trigger} onChange={e => setForm(p => ({ ...p, trigger: e.target.value }))}>
              <option value="http">HTTP</option>
              <option value="schedule">Schedule</option>
              <option value="event">Event</option>
            </Input>
            {form.trigger === 'schedule' && <Input placeholder="Cron schedule" value={form.schedule} onChange={e => setForm(p => ({ ...p, schedule: e.target.value }))} />}
            <Input placeholder='Env JSON ({"KEY":"val"})' value={form.env} onChange={e => setForm(p => ({ ...p, env: e.target.value }))} />
            <Input placeholder="Timeout (seconds)" type="number" value={form.timeout} onChange={e => setForm(p => ({ ...p, timeout: +e.target.value }))} />
            <Input as="textarea" placeholder="Source code *" value={form.source} onChange={e => setForm(p => ({ ...p, source: e.target.value }))} required rows={8} className="code-textarea" />
            <div className="form-actions">
              <Button type="submit">{editing ? 'Update' : 'Create'}</Button>
              <Button type="button" variant="secondary" onClick={() => setShowForm(false)}>Cancel</Button>
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
                    <Button variant="secondary" size="sm" onClick={() => handleRun(fn.id)}>Run</Button>
                    <Button variant="secondary" size="sm" onClick={() => navigate(`/functions/${fn.id}/logs`)}>Logs</Button>
                    <Button variant="secondary" size="sm" onClick={() => openEdit(fn)}>Edit</Button>
                    <Button variant="danger" size="sm" onClick={() => handleDelete(fn.id)}>Delete</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {Object.entries(runResult).map(([id, result]) => (
        <div key={id} className="card" style={{ marginTop: 'var(--space-8)', maxHeight: 400, overflow: 'auto' }}>
          <h3 style={{ marginBottom: 'var(--space-4)' }}>
            Run Result — <code>{fns.find(f => f.id === id)?.name || id}</code>
          </h3>
          {result.logs?.length > 0 && (
            <div className="code-output">
              {result.logs.map((l, i) => <pre key={i}>{l}</pre>)}
            </div>
          )}
          {result.error && <p className="input-error-text">{result.error}</p>}
          {result.result !== undefined && (
            <div className="code-output" style={{ marginTop: 'var(--space-4)' }}>
              <pre>{JSON.stringify(result.result, null, 2)}</pre>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
