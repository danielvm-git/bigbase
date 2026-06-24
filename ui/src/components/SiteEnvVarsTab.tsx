import { useState, useEffect, useRef } from 'react'
import { Button, Card } from '.'
import { getEnvVars, createEnvVar, updateEnvVar, deleteEnvVar } from '../lib/sitesData'
import type { EnvVar } from '../types/sites'

const KEY_RE = /^[A-Z][A-Z0-9_]*$/

function maskValue(value: string): string {
  if (value.length <= 4) return '••••'
  return '••••' + value.slice(-4)
}

function parseEnvFile(text: string): Array<{ key: string; value: string }> {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(line => line && !line.startsWith('#'))
    .flatMap(line => {
      const eq = line.indexOf('=')
      if (eq < 1) return []
      const key = line.slice(0, eq).trim()
      const raw = line.slice(eq + 1).trim()
      const value = raw.startsWith('"') && raw.endsWith('"') ? raw.slice(1, -1) : raw
      return [{ key, value }]
    })
}

function EnvVarRow({
  ev,
  onEdit,
  onDelete,
}: {
  ev: EnvVar
  onEdit: (ev: EnvVar) => void
  onDelete: (key: string) => void
}) {
  return (
    <tr>
      <td><code style={{ fontSize: 'var(--text-sm)' }}>{ev.key}</code></td>
      <td>
        <code style={{ fontSize: 'var(--text-sm)', color: 'var(--fg-secondary)' }}>
          {maskValue(ev.value)}
        </code>
      </td>
      <td style={{ textAlign: 'center' }}>{ev.is_build_time ? '✓' : '—'}</td>
      <td style={{ textAlign: 'center' }}>{ev.is_runtime ? '✓' : '—'}</td>
      <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
        {new Date(ev.created_at).toLocaleDateString()}
      </td>
      <td>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <Button variant="ghost" size="sm" onClick={() => onEdit(ev)}>Edit</Button>
          <Button
            variant="ghost"
            size="sm"
            style={{ color: 'var(--error)' }}
            onClick={() => onDelete(ev.key)}
          >
            Delete
          </Button>
        </div>
      </td>
    </tr>
  )
}

function AddEditForm({
  initial,
  onSave,
  onCancel,
  error,
  saving,
}: {
  initial?: EnvVar
  onSave: (data: { key: string; value: string; is_build_time: boolean; is_runtime: boolean }) => void
  onCancel: () => void
  error?: string
  saving: boolean
}) {
  const [key, setKey] = useState(initial?.key ?? '')
  const [value, setValue] = useState(initial?.value ?? '')
  const [showValue, setShowValue] = useState(false)
  const [isBuildTime, setIsBuildTime] = useState(initial?.is_build_time ?? false)
  const [isRuntime, setIsRuntime] = useState(initial?.is_runtime ?? true)
  const [keyError, setKeyError] = useState('')
  const isEdit = !!initial

  const handleSave = () => {
    if (!isEdit) {
      if (!key) { setKeyError('Key is required'); return }
      if (!KEY_RE.test(key)) { setKeyError('Must be uppercase letters, digits, and underscores'); return }
    }
    onSave({ key, value, is_build_time: isBuildTime, is_runtime: isRuntime })
  }

  return (
    <div style={{ background: 'var(--bg-secondary)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', marginBottom: 'var(--space-4)' }}>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-3)', alignItems: 'flex-start' }}>
        {!isEdit && (
          <div style={{ flex: '1 1 180px' }}>
            <label className="label">Key</label>
            <input
              className="input"
              placeholder="DATABASE_URL"
              value={key}
              onChange={e => { setKey(e.target.value.toUpperCase()); setKeyError('') }}
              disabled={saving}
              autoFocus
            />
            {keyError && <p className="input-error-text">{keyError}</p>}
          </div>
        )}
        <div style={{ flex: '2 1 240px' }}>
          <label className="label">Value</label>
          <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
            <input
              className="input"
              type={showValue ? 'text' : 'password'}
              placeholder="Enter value…"
              value={value}
              onChange={e => setValue(e.target.value)}
              disabled={saving}
              style={{ flex: 1 }}
            />
            <Button variant="ghost" size="sm" onClick={() => setShowValue(v => !v)}>
              {showValue ? 'Hide' : 'Show'}
            </Button>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 'var(--space-4)', alignItems: 'center', paddingTop: 'var(--space-5)' }}>
          <label style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center', fontSize: 'var(--text-sm)', cursor: 'pointer' }}>
            <input type="checkbox" checked={isBuildTime} onChange={e => setIsBuildTime(e.target.checked)} />
            Build
          </label>
          <label style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center', fontSize: 'var(--text-sm)', cursor: 'pointer' }}>
            <input type="checkbox" checked={isRuntime} onChange={e => setIsRuntime(e.target.checked)} />
            Runtime
          </label>
        </div>
        <div style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center', paddingTop: 'var(--space-5)' }}>
          <Button variant="primary" size="sm" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving…' : isEdit ? 'Update' : 'Add'}
          </Button>
          <Button variant="secondary" size="sm" onClick={onCancel} disabled={saving}>Cancel</Button>
        </div>
      </div>
      {error && <p className="input-error-text" style={{ marginTop: 'var(--space-2)' }}>{error}</p>}
    </div>
  )
}

export function SiteEnvVarsTab({ siteId }: { siteId: string }) {
  const [envVars, setEnvVars] = useState<EnvVar[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [editEv, setEditEv] = useState<EnvVar | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [importError, setImportError] = useState<string | null>(null)
  const [importing, setImporting] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    const vars = await getEnvVars(siteId)
    setEnvVars(vars)
    setLoading(false)
  }

  useEffect(() => { void load() }, [siteId])

  const handleAdd = async (data: Parameters<typeof createEnvVar>[1]) => {
    setSaving(true); setSaveError(null)
    const res = await createEnvVar(siteId, data)
    setSaving(false)
    if (!res.ok) { setSaveError(res.error ?? 'Failed'); return }
    setShowAdd(false)
    void load()
  }

  const handleUpdate = async (data: { key: string; value: string; is_build_time: boolean; is_runtime: boolean }) => {
    if (!editEv) return
    setSaving(true); setSaveError(null)
    const res = await updateEnvVar(siteId, editEv.key, data)
    setSaving(false)
    if (!res.ok) { setSaveError(res.error ?? 'Failed'); return }
    setEditEv(null)
    void load()
  }

  const handleDelete = async (key: string) => {
    if (!window.confirm(`Delete env var "${key}"? This cannot be undone.`)) return
    const res = await deleteEnvVar(siteId, key)
    if (!res.ok) { setError(res.error ?? 'Delete failed'); return }
    void load()
  }

  const handleExport = () => {
    const lines = envVars.map(ev => `${ev.key}=${ev.value}`).join('\n')
    const blob = new Blob([lines + '\n'], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = '.env'; a.click()
    URL.revokeObjectURL(url)
  }

  const handleImportFile = async (file: File) => {
    setImporting(true); setImportError(null)
    const text = await file.text()
    const pairs = parseEnvFile(text)
    if (pairs.length === 0) { setImportError('No valid KEY=value pairs found'); setImporting(false); return }

    const existingKeys = new Set(envVars.map(e => e.key))
    const overlapping = pairs.filter(p => existingKeys.has(p.key)).map(p => p.key)
    if (overlapping.length > 0) {
      if (!window.confirm(`These keys already exist and will be overwritten:\n${overlapping.join(', ')}\n\nContinue?`)) {
        setImporting(false); return
      }
    }

    for (const { key, value } of pairs) {
      if (existingKeys.has(key)) {
        await updateEnvVar(siteId, key, { value, is_build_time: false, is_runtime: true })
      } else {
        await createEnvVar(siteId, { key, value, is_build_time: false, is_runtime: true })
      }
    }
    setImporting(false)
    void load()
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const file = e.dataTransfer.files[0]
    if (file) void handleImportFile(file)
  }

  if (loading) return <p className="dim">Loading…</p>
  if (error) return <p className="input-error-text">{error}</p>

  return (
    <div
      onDragOver={e => e.preventDefault()}
      onDrop={handleDrop}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-4)' }}>
        <h2 className="section-title" style={{ margin: 0 }}>Environment Variables</h2>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <input ref={fileRef} type="file" accept=".env,text/plain" style={{ display: 'none' }}
            onChange={e => { const f = e.target.files?.[0]; if (f) void handleImportFile(f); e.target.value = '' }}
          />
          <Button variant="secondary" size="sm" onClick={() => fileRef.current?.click()} disabled={importing}>
            {importing ? 'Importing…' : 'Import .env'}
          </Button>
          {envVars.length > 0 && (
            <Button variant="secondary" size="sm" onClick={handleExport}>Export .env</Button>
          )}
          <Button variant="primary" size="sm" onClick={() => { setShowAdd(true); setEditEv(null) }}>
            Add Variable
          </Button>
        </div>
      </div>

      {importError && <p className="input-error-text" style={{ marginBottom: 'var(--space-3)' }}>{importError}</p>}

      {showAdd && !editEv && (
        <AddEditForm
          onSave={handleAdd}
          onCancel={() => { setShowAdd(false); setSaveError(null) }}
          error={saveError ?? undefined}
          saving={saving}
        />
      )}

      {editEv && (
        <AddEditForm
          initial={editEv}
          onSave={handleUpdate}
          onCancel={() => { setEditEv(null); setSaveError(null) }}
          error={saveError ?? undefined}
          saving={saving}
        />
      )}

      {envVars.length === 0 && !showAdd ? (
        <Card style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
          <p className="dim" style={{ marginBottom: 'var(--space-4)' }}>
            No environment variables yet. Add one above or drag & drop a <code>.env</code> file here.
          </p>
        </Card>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Key</th>
                <th>Value</th>
                <th style={{ textAlign: 'center' }}>Build</th>
                <th style={{ textAlign: 'center' }}>Runtime</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {envVars.map(ev => (
                <EnvVarRow
                  key={ev.key}
                  ev={ev}
                  onEdit={ev => { setEditEv(ev); setShowAdd(false); setSaveError(null) }}
                  onDelete={handleDelete}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
