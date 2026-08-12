import { useEffect, useRef, useState } from 'react'
import { Button, Card, Dialog, Modal } from './index'
import {
  createSecret,
  deleteSecret,
  importEnvSecrets,
  listSecrets,
  listSecretVersions,
  parseEnvFile,
  readSecretValue,
  updateSecret,
  MAX_SECRET_BATCH_KEYS,
} from '../lib/secretsData'
import type { SecretMetadata, SecretValue, SecretVersionMetadata } from '../types/secrets'

// ProjectSecretsTab — the Project → Environment → Folder secret surface
// (e89s05). Hard rules, enforced by shape and state flow:
//
//   - List state holds SecretMetadata only; it never holds plaintext.
//   - Reveal is explicit: it calls the /value route and the returned
//     SecretValue lives only in the reveal modal state, cleared on close
//     (and destroyed with the component on unmount).
//   - Edit forms never prefill a plaintext value (metadata has none).
//   - 401/403 failures render value-free messages.
//   - .env import writes per key, bounded by MAX_SECRET_BATCH_KEYS, and
//     reports failures by key name only — never echoing submitted values.

const KEY_RE = /^[A-Z][A-Z0-9_]*$/

interface RevealState {
  secret: SecretMetadata
  /** Held only while the reveal modal is open; cleared on close/unmount. */
  value: SecretValue | null
  loading: boolean
  error: string | null
}

interface VersionsState {
  secret: SecretMetadata
  items: SecretVersionMetadata[]
  loading: boolean
  error: string | null
}

function SecretForm({
  initial,
  onSave,
  onCancel,
  error,
  saving,
}: {
  initial?: SecretMetadata
  onSave: (data: { key: string; value: string }) => void
  onCancel: () => void
  error?: string
  saving: boolean
}) {
  const [key, setKey] = useState(initial?.key ?? '')
  // Edit never prefills a value: list metadata carries no plaintext, so an
  // edit always requires a fresh replacement value (SC-e89s05-P1-04).
  const [value, setValue] = useState('')
  const [showValue, setShowValue] = useState(false)
  const [keyError, setKeyError] = useState('')
  const isEdit = !!initial

  const handleSave = () => {
    if (!isEdit) {
      if (!key) { setKeyError('Key is required'); return }
      if (!KEY_RE.test(key)) { setKeyError('Must be uppercase letters, digits, and underscores'); return }
    }
    onSave({ key, value })
  }

  return (
    <div
      style={{ background: 'var(--bg-secondary)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', marginBottom: 'var(--space-4)' }}
    >
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-3)', alignItems: 'flex-start' }}>
        {!isEdit && (
          <div style={{ flex: '1 1 180px' }}>
            <label className="label" htmlFor="secret-key">Key</label>
            <input
              className="input"
              id="secret-key"
              placeholder="DATABASE_URL"
              value={key}
              onChange={e => { setKey(e.target.value.toUpperCase()); setKeyError('') }}
              disabled={saving}
              autoFocus
            />
            {keyError && <p className="input-error-text" role="alert">{keyError}</p>}
          </div>
        )}
        <div style={{ flex: '2 1 240px' }}>
          <label className="label" htmlFor="secret-value">Value</label>
          <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
            <input
              className="input"
              id="secret-value"
              type={showValue ? 'text' : 'password'}
              placeholder={isEdit ? 'Enter a replacement value…' : 'Enter value…'}
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
        <div style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'center', paddingTop: 'var(--space-5)' }}>
          <Button variant="primary" size="sm" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving…' : isEdit ? 'Update' : 'Add'}
          </Button>
          <Button variant="secondary" size="sm" onClick={onCancel} disabled={saving}>Cancel</Button>
        </div>
      </div>
      {error && <p className="input-error-text" role="alert" style={{ marginTop: 'var(--space-2)' }}>{error}</p>}
    </div>
  )
}

export function ProjectSecretsTab({
  projectId,
  envId,
  folderName = 'default',
}: {
  projectId: string
  envId: string
  folderName?: string
}) {
  const [secrets, setSecrets] = useState<SecretMetadata[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [editSecret, setEditSecret] = useState<SecretMetadata | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<SecretMetadata | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [reveal, setReveal] = useState<RevealState | null>(null)
  const [versions, setVersions] = useState<VersionsState | null>(null)
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState<string | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    const items = await listSecrets(projectId, envId)
    setSecrets(items)
    setLoading(false)
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect, react-hooks/exhaustive-deps
  useEffect(() => { void load() }, [projectId, envId])

  const handleAdd = async (data: { key: string; value: string }) => {
    setSaving(true); setSaveError(null)
    const res = await createSecret(projectId, envId, data)
    setSaving(false)
    if (!res.ok) { setSaveError(res.error ?? 'Failed'); return }
    setShowAdd(false)
    void load()
  }

  const handleUpdate = async (data: { key: string; value: string }) => {
    if (!editSecret) return
    setSaving(true); setSaveError(null)
    const res = await updateSecret(projectId, envId, editSecret.key, { value: data.value })
    setSaving(false)
    if (!res.ok) { setSaveError(res.error ?? 'Failed'); return }
    setEditSecret(null)
    void load()
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    const target = deleteTarget
    setDeleteTarget(null)
    const res = await deleteSecret(projectId, envId, target.key)
    if (!res.ok) { setError(res.error ?? 'Delete failed'); return }
    void load()
  }

  const openReveal = async (secret: SecretMetadata) => {
    // Only one plaintext-bearing surface at a time.
    setVersions(null)
    setReveal({ secret, value: null, loading: true, error: null })
    const res = await readSecretValue(projectId, envId, secret.key)
    setReveal(prev => {
      if (!prev || prev.secret.id !== secret.id) return prev
      if (res.ok && res.data) return { ...prev, value: res.data, loading: false }
      return { ...prev, loading: false, error: res.error ?? 'Could not read this value.' }
    })
  }

  const closeReveal = () => {
    // Clears the only state that held the plaintext value (SC-e89s05-P1-02).
    setReveal(null)
  }

  const openVersions = async (secret: SecretMetadata) => {
    setReveal(null)
    setVersions({ secret, items: [], loading: true, error: null })
    const items = await listSecretVersions(projectId, envId, secret.key)
    setVersions(prev => (prev && prev.secret.id === secret.id ? { ...prev, items, loading: false } : prev))
  }

  const handleImportFile = async (file: File) => {
    setImporting(true); setImportError(null)
    const text = await file.text()
    const existingKeys = new Set(secrets.map(s => s.key))

    // Overwrite confirmation mirrors the Site env-var tab.
    const pairs = parseEnvFile(text)
    if (pairs.length === 0) { setImportError('No valid KEY=value pairs found'); setImporting(false); return }
    const overlapping = pairs.filter(p => existingKeys.has(p.key)).map(p => p.key)
    if (overlapping.length > 0) {
      if (!window.confirm(`These keys already exist and will be overwritten:\n${overlapping.join(', ')}\n\nContinue?`)) {
        setImporting(false); return
      }
    }

    const result = await importEnvSecrets(projectId, envId, text, existingKeys)
    setImporting(false)
    if (result.failedKeys.length > 0) {
      // Failures are reported by key name only — never with values.
      const noun = result.failedKeys.length === 1 ? 'key failed' : 'keys failed'
      setImportError(`${result.failedKeys.length} ${noun} to import: ${result.failedKeys.join(', ')}`)
    } else {
      setImportError(null)
    }
    void load()
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const file = e.dataTransfer.files[0]
    if (file) void handleImportFile(file)
  }

  if (loading) return <p className="dim">Loading secrets…</p>
  if (error) return <p className="input-error-text" role="alert">{error}</p>

  return (
    <div
      id="panel-project-secrets"
      onDragOver={e => e.preventDefault()}
      onDrop={handleDrop}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-4)' }}>
        <h2 className="section-title" style={{ margin: 0 }}>
          Secrets in <code>{folderName}</code>
        </h2>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <input
            ref={fileRef}
            type="file"
            accept=".env,text/plain"
            style={{ display: 'none' }}
            aria-label="Import .env file"
            onChange={e => { const f = e.target.files?.[0]; if (f) void handleImportFile(f); e.target.value = '' }}
          />
          <Button variant="secondary" size="sm" onClick={() => fileRef.current?.click()} disabled={importing}>
            {importing ? 'Importing…' : 'Import .env'}
          </Button>
          <Button variant="primary" size="sm" onClick={() => { setShowAdd(true); setEditSecret(null); setSaveError(null) }}>
            Add Secret
          </Button>
        </div>
      </div>

      <p className="input-hint" style={{ marginTop: 0, marginBottom: 'var(--space-4)' }}>
        Values are masked until you explicitly reveal one. Import is bounded to {MAX_SECRET_BATCH_KEYS} keys per file.
      </p>

      {importError && <p className="input-error-text" role="alert" style={{ marginBottom: 'var(--space-3)' }}>{importError}</p>}

      {showAdd && !editSecret && (
        <SecretForm
          onSave={handleAdd}
          onCancel={() => { setShowAdd(false); setSaveError(null) }}
          error={saveError ?? undefined}
          saving={saving}
        />
      )}

      {editSecret && (
        <SecretForm
          initial={editSecret}
          onSave={handleUpdate}
          onCancel={() => { setEditSecret(null); setSaveError(null) }}
          error={saveError ?? undefined}
          saving={saving}
        />
      )}

      {secrets.length === 0 && !showAdd ? (
        <Card style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
          <p className="dim" style={{ margin: 0 }}>No secrets in this folder yet.</p>
        </Card>
      ) : (
        <div className="table-wrap">
          <table>
            <caption className="visually-hidden">Project secrets in folder {folderName}</caption>
            <thead>
              <tr>
                <th scope="col">Key</th>
                <th scope="col">Value</th>
                <th scope="col">Version</th>
                <th scope="col">Updated</th>
                <th scope="col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {secrets.map(s => (
                <tr key={s.id}>
                  <td><code style={{ fontSize: 'var(--text-sm)' }}>{s.key}</code></td>
                  <td>
                    <code style={{ fontSize: 'var(--text-sm)', color: 'var(--fg-secondary)' }}>{s.value_preview}</code>
                  </td>
                  <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>v{s.current_version}</td>
                  <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
                    {new Date(s.updated_at).toLocaleDateString()}
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => void openReveal(s)}
                        disabled={reveal?.loading && reveal.secret.id === s.id}
                      >
                        Reveal
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => { setEditSecret(s); setShowAdd(false); setSaveError(null) }}>
                        Edit
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => void openVersions(s)}>
                        Versions
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        style={{ color: 'var(--error)' }}
                        onClick={() => setDeleteTarget(s)}
                      >
                        Delete
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal open={!!reveal} onClose={closeReveal} title={reveal ? `Reveal ${reveal.secret.key}` : 'Reveal'}>
        {reveal?.loading && <p className="dim">Reading value…</p>}
        {reveal?.error && <p className="input-error-text" role="alert">{reveal.error}</p>}
        {reveal?.value && !reveal.loading && !reveal.error && (
          <div>
            <p style={{ fontSize: 'var(--text-s)', color: 'var(--fg-secondary)', margin: 0 }}>
              {reveal.value.key} (version {reveal.value.version})
            </p>
            <pre
              className="code-block secret-value"
              style={{ overflow: 'auto', userSelect: 'text', margin: 'var(--space-4) 0' }}
            >{reveal.value.value}</pre>
            <p className="input-hint" style={{ margin: 0 }}>
              This value is displayed once and cleared when you close this dialog.
            </p>
          </div>
        )}
      </Modal>

      <Modal
        open={!!versions}
        onClose={() => setVersions(null)}
        title={versions ? `Versions — ${versions.secret.key}` : 'Versions'}
      >
        {versions?.loading && <p className="dim">Loading versions…</p>}
        {versions?.error && <p className="input-error-text" role="alert">{versions.error}</p>}
        {versions && !versions.loading && (
          <ul style={{ listStyle: 'none', margin: 0, padding: 0 }}>
            {versions.items.length === 0 && <li className="dim">No version history.</li>}
            {versions.items.map(v => (
              <li
                key={v.id}
                style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-4)', padding: 'var(--space-3) 0', borderBottom: '1px solid var(--border-default)' }}
              >
                <span><code>v{v.version}</code></span>
                <span className="dim" style={{ fontSize: 'var(--text-xs)' }}>
                  {v.algorithm} · {new Date(v.created_at).toLocaleString()}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Modal>

      <Dialog
        open={!!deleteTarget}
        title="Delete secret"
        danger
        confirmLabel="Delete secret"
        cancelLabel="Cancel"
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
      >
        {deleteTarget && (
          <p>
            Delete secret <strong>{deleteTarget.key}</strong>? This permanently removes the secret and all of its
            versions and cannot be undone.
          </p>
        )}
      </Dialog>
    </div>
  )
}
