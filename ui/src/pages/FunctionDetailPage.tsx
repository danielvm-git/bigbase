import { useEffect, useState } from 'react'
import { useParams, useSearchParams, Navigate } from 'react-router-dom'
import { PageHeader, Button, Input, Tabs, Breadcrumb, FunctionLogsPanel } from '../components'

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

const detailTabs = [
  { id: 'code', label: 'Code' },
  { id: 'triggers', label: 'Triggers' },
  { id: 'variables', label: 'Variables' },
  { id: 'logs', label: 'Logs' },
]

export default function FunctionDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') || 'code'
  const [fn, setFn] = useState<Fn | null>(null)
  const [error, setError] = useState('')
  const [source, setSource] = useState('')
  const [env, setEnv] = useState('{}')

  useEffect(() => {
    if (!id) return
    fetch('/api/functions')
      .then(r => r.json())
      .then(d => {
        const list = (d as { data: Fn[] }).data || []
        const found = list.find(f => f.id === id)
        if (!found) {
          setError('Function not found')
          return
        }
        setFn(found)
        setSource(found.source)
        setEnv(found.env)
      })
      .catch(() => setError('Failed to load function'))
  }, [id])

  const setTab = (next: string) => setSearchParams({ tab: next })

  if (!id) return null
  if (error) return <p className="input-error-text">{error}</p>
  if (!fn) return <div className="loading">Loading function...</div>

  return (
    <div>
      <Breadcrumb items={[
        { label: 'Functions', to: '/functions' },
        { label: fn.name },
      ]} />
      <PageHeader title={fn.name}>
        <Button variant="secondary" size="sm" onClick={() => window.history.back()}>Back</Button>
      </PageHeader>

      <Tabs tabs={detailTabs} active={tab} onChange={setTab} />

      {tab === 'code' && (
        <div className="card">
          <Input as="textarea" value={source} onChange={e => setSource(e.target.value)} rows={16} className="code-textarea" />
          <div className="form-actions" style={{ marginTop: 'var(--space-6)' }}>
            <Button size="sm" onClick={async () => {
              const res = await fetch(`/api/functions/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ...fn, source }),
              })
              if (!res.ok) setError('Save failed')
            }}>Save code</Button>
          </div>
        </div>
      )}

      {tab === 'triggers' && (
        <div className="card">
          <p><strong>Trigger:</strong> {fn.trigger}</p>
          {fn.trigger === 'schedule' && <p><strong>Schedule:</strong> {fn.schedule || '—'}</p>}
          <p className="dim">Runtime: {fn.runtime} · Timeout: {fn.timeout}s</p>
        </div>
      )}

      {tab === 'variables' && (
        <div className="card">
          <Input as="textarea" value={env} onChange={e => setEnv(e.target.value)} rows={8} className="code-textarea" />
          <div className="form-actions" style={{ marginTop: 'var(--space-6)' }}>
            <Button size="sm" onClick={async () => {
              const res = await fetch(`/api/functions/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ...fn, env }),
              })
              if (!res.ok) setError('Save failed')
            }}>Save variables</Button>
          </div>
        </div>
      )}

      {tab === 'logs' && <FunctionLogsPanel functionId={id} showMonitoringLink />}
    </div>
  )
}

export function FunctionLogsRedirect() {
  const { id } = useParams<{ id: string }>()
  if (!id) return null
  return <Navigate to={`/functions/${id}?tab=logs`} replace />
}
