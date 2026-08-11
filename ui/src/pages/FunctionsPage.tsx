import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader, Button, Input } from '../components'
import {
  type FunctionRecord,
  type FunctionFormFields,
  formEnvFromRecord,
  functionPayloadFromForm,
} from '../lib/functionEnv'

interface RunResult {
  logs: string[]
  result: unknown
  error: string | null
}

const defaultForm: FunctionFormFields = {
  name: '',
  runtime: 'javascript',
  source: '',
  trigger: 'http',
  schedule: '',
  env: '{}',
  timeout: 30,
}

export default function FunctionsPage() {
  const navigate = useNavigate()
  const [fns, setFns] = useState<FunctionRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<FunctionRecord | null>(null)
  const [form, setForm] = useState(defaultForm)
  const [runResult, setRunResult] = useState<Record<string, RunResult>>({})

  const fetchFns = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/functions')
      const d = await res.json()
      if (!res.ok) { setError(d.error || `error: ${res.status}`); setFns([]) }
      else { setFns((d as { data: FunctionRecord[] }).data || []) }
    } catch { setError('network error') }
    finally { setLoading(false) }
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { fetchFns() }, [])

  const openCreate = () => { setEditing(null); setForm(defaultForm); setShowForm(true) }
  const openEdit = (fn: FunctionRecord) => {
    setEditing(fn)
    setForm({
      name: fn.name,
      runtime: fn.runtime,
      source: fn.source,
      trigger: fn.trigger,
      schedule: fn.schedule,
      env: formEnvFromRecord(fn.env),
      timeout: fn.timeout,
    })
    setShowForm(true)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    const built = functionPayloadFromForm(form)
    if (!built.ok) {
      setError(built.error)
      return
    }
    const isNew = !editing
    const url = isNew ? '/api/functions' : `/api/functions/${editing.id}`
    const method = isNew ? 'POST' : 'PUT'
    try {
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(built.payload),
      })
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
        <Button variant="primary" size="sm" onClick={openCreate}>Create function</Button>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}

      {showForm && (
        <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
          <h2 style={{ marginBottom: 'var(--space-6)', fontSize: 'var(--text-m)', fontWeight: 600 }}>
            {editing ? 'Edit Function' : 'New Function'}
          </h2>
          <form onSubmit={handleSave} className="fn-form">
            <Input label="Name" placeholder="Name *" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} required />
            <Input as="select" aria-label="Runtime" value={form.runtime} onChange={e => setForm(p => ({ ...p, runtime: e.target.value }))}>
              <option value="javascript">JavaScript</option>
            </Input>
            <Input as="select" aria-label="Trigger" value={form.trigger} onChange={e => setForm(p => ({ ...p, trigger: e.target.value }))}>
              <option value="http">HTTP</option>
              <option value="schedule">Schedule</option>
              <option value="event">Event</option>
            </Input>
            {form.trigger === 'schedule' && <Input label="Cron schedule" placeholder="Cron schedule" value={form.schedule} onChange={e => setForm(p => ({ ...p, schedule: e.target.value }))} />}
            <Input label="Env JSON" placeholder='Env JSON ({"KEY":"val"})' value={form.env} onChange={e => setForm(p => ({ ...p, env: e.target.value }))} />
            <Input label="Timeout (seconds)" placeholder="Timeout (seconds)" type="number" value={form.timeout} onChange={e => setForm(p => ({ ...p, timeout: +e.target.value }))} />
            <Input as="textarea" aria-label="Source code" placeholder="Source code *" value={form.source} onChange={e => setForm(p => ({ ...p, source: e.target.value }))} required rows={8} className="code-textarea" />
            <div className="form-actions">
              <Button type="submit">{editing ? 'Update' : 'Create'}</Button>
              <Button type="button" variant="secondary" onClick={() => setShowForm(false)}>Cancel</Button>
            </div>
          </form>
        </div>
      )}

      {fns.length === 0 && !error && <p className="dim">No functions yet.</p>}
      {fns.length > 0 && (
        <div className="function-grid">
          {fns.map(fn => (
            <article key={fn.id} className="function-card">
              <div className="function-card-name">{fn.name}</div>
              <div className="function-card-meta">
                {fn.runtime} · {fn.trigger} trigger
              </div>
              <div className="function-card-meta dim">
                Created {new Date(fn.created_at).toLocaleDateString()}
              </div>
              <div className="function-card-actions">
                <Button variant="primary" size="sm" onClick={() => navigate(`/functions/${fn.id}`)}>Open</Button>
                <Button variant="secondary" size="sm" onClick={() => handleRun(fn.id)}>Run</Button>
                <Button variant="ghost" size="sm" onClick={() => navigate(`/functions/${fn.id}?tab=logs`)}>Logs</Button>
                <Button variant="ghost" size="sm" onClick={() => openEdit(fn)}>Edit</Button>
                <Button variant="danger" size="sm" onClick={() => handleDelete(fn.id)}>Delete</Button>
              </div>
            </article>
          ))}
        </div>
      )}

      {Object.entries(runResult).map(([id, result]) => (
        <div key={id} className="card" style={{ marginTop: 'var(--space-8)', maxHeight: 400, overflow: 'auto' }}>
          <h2 style={{ marginBottom: 'var(--space-4)', fontSize: 'var(--text-m)', fontWeight: 600 }}>
            Run Result — <code>{fns.find(f => f.id === id)?.name || id}</code>
          </h2>
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
