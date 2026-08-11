import { useEffect, useState } from 'react'
import { PageHeader, Button, Input, Tabs, Badge, statusBadgeVariant } from '../components'

interface Repo { id: string; name: string }
interface Workflow { id: string; repo_id: string; name: string; yaml: string }
interface Run { id: string; workflow_id: string; event: string; status: string; started_at: string; finished_at: string }
interface Log { id: string; run_id: string; step: string; output: string }

export default function CiciPage() {
  const [repos, setRepos] = useState<Repo[]>([])
  const [repoId, setRepoId] = useState('')
  const [workflows, setWorkflows] = useState<Workflow[]>([])
  const [runs, setRuns] = useState<Run[]>([])
  const [logs, setLogs] = useState<Record<string, Log[]>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', yaml: '' })
  const [expandedRun, setExpandedRun] = useState<string | null>(null)

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
      fetch(`/api/cici/${repoId}/workflows`).then(r => r.json()),
      fetch(`/api/cici/runs?repo_id=${repoId}`).then(r => r.json()),
    ]).then(([wD, rD]) => {
      setWorkflows((wD as { data: Workflow[] }).data || [])
      setRuns((rD as { data: Run[] }).data || [])
    }).catch(() => setError('failed to load'))
    .finally(() => setLoading(false))
  }, [repoId])

  const loadLogs = async (runId: string) => {
    if (expandedRun === runId) { setExpandedRun(null); return }
    setExpandedRun(runId)
    try {
      const res = await fetch(`/api/cici/runs/${runId}/logs`)
      if (res.ok) {
        const d = await res.json()
        setLogs(p => ({ ...p, [runId]: (d as { logs: Log[] }).logs || [] }))
      }
    } catch { /* ignore */ }
  }

  const handleCreateWorkflow = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    try {
      const res = await fetch(`/api/cici/${repoId}/workflows`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'save failed'); return }
      setShowForm(false); setForm({ name: '', yaml: '' })
      const wRes = await fetch(`/api/cici/${repoId}/workflows`)
      const wD = await wRes.json()
      setWorkflows((wD as { data: Workflow[] }).data || [])
    } catch { setError('network error') }
  }

  const handleRun = async (wfId: string) => {
    setError('')
    try {
      const res = await fetch(`/api/cici/${repoId}/workflows/${wfId}/run`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ event: 'manual' }),
      })
      if (!res.ok) { const d = await res.json(); setError(d.error || 'run failed'); return }
      const rRes = await fetch(`/api/cici/runs?repo_id=${repoId}`)
      const rD = await rRes.json()
      setRuns((rD as { data: Run[] }).data || [])
    } catch { setError('network error') }
  }

  const ciciTabs = [
    { id: 'workflows', label: 'Workflows' },
    { id: 'runs', label: 'Runs' },
  ]
  const [ciciTab, setCiciTab] = useState('workflows')

  if (!repoId) return (
    <div>
      <PageHeader title="CI/CD" />
      <p className="dim">Select a repo to view workflows.</p>
      <select value={repoId} onChange={e => setRepoId(e.target.value)} className="input" aria-label="Repository" style={{ maxWidth: 240, marginTop: 'var(--space-4)' }}>
        <option value="">Select repo...</option>
        {repos.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
      </select>
    </div>
  )

  if (loading) return <div className="loading">Loading...</div>

  return (
    <div>
      <PageHeader title="CI/CD">
        <select value={repoId} onChange={e => setRepoId(e.target.value)} className="input" aria-label="Repository" style={{ maxWidth: 200 }}>
          {repos.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
        <Button variant="primary" size="sm" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'New Workflow'}
        </Button>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}

      {showForm && (
        <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
          <h2 style={{ marginBottom: 'var(--space-6)', fontSize: 'var(--text-m)', fontWeight: 600 }}>New Workflow</h2>
          <form onSubmit={handleCreateWorkflow} className="fn-form">
            <Input label="Workflow name" placeholder="Workflow name *" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} required />
            <Input as="textarea" aria-label="YAML config" placeholder="YAML config *" value={form.yaml} onChange={e => setForm(p => ({ ...p, yaml: e.target.value }))} required rows={10} className="code-textarea" />
            <Button type="submit">Save</Button>
          </form>
        </div>
      )}

      <Tabs tabs={ciciTabs} active={ciciTab} onChange={setCiciTab} />

      {ciciTab === 'workflows' && (
        <>
          <h2 className="section-title">Workflows</h2>
          {workflows.length === 0 ? <p className="dim">No workflows.</p> : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Name</th><th>Actions</th></tr>
                </thead>
                <tbody>
                  {workflows.map(w => (
                    <tr key={w.id}>
                      <td><code>{w.name}</code></td>
                      <td><Button variant="secondary" size="sm" onClick={() => handleRun(w.id)}>Run</Button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {ciciTab === 'runs' && (
        <>
          <h2 className="section-title">Runs</h2>
          {runs.length === 0 ? <p className="dim">No runs yet.</p> : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>ID</th><th>Event</th><th>Status</th><th>Started</th><th>Finished</th><th>Logs</th></tr>
                </thead>
                <tbody>
                  {runs.map(run => (
                    <tr key={run.id}>
                      <td><code>{run.id.slice(0, 8)}</code></td>
                      <td>{run.event}</td>
                      <td><Badge variant={statusBadgeVariant(run.status)}>{run.status}</Badge></td>
                      <td>{run.started_at ? new Date(run.started_at).toLocaleString() : '—'}</td>
                      <td>{run.finished_at ? new Date(run.finished_at).toLocaleString() : '—'}</td>
                      <td><Button variant="secondary" size="sm" onClick={() => loadLogs(run.id)}>{expandedRun === run.id ? 'Hide' : 'Logs'}</Button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {expandedRun && logs[expandedRun] && (
        <div className="card" style={{ marginTop: 'var(--space-8)' }}>
          <h3 style={{ marginBottom: 'var(--space-4)' }}>Logs — <code>{expandedRun.slice(0, 8)}</code></h3>
          {logs[expandedRun].length === 0 ? <p className="dim">No logs.</p> : (
            <div className="code-output">
              {logs[expandedRun].map((l, i) => (
                <div key={i} className="log-entry">
                  <span className="log-step">[{l.step}]</span>
                  <pre>{l.output}</pre>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
