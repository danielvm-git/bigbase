import { useState, useEffect } from 'react'
import { Button, Modal, CopyButton, EmptyState } from '.'
import { getDeployKeys, createDeployKey, revokeDeployKey } from '../lib/sitesData'
import type { SiteDeployKey } from '../types/sites'

function DeployKeyRow({
  k,
  onRevoke,
}: {
  k: SiteDeployKey
  onRevoke: (id: string) => void
}) {
  return (
    <tr>
      <td><code style={{ fontSize: 'var(--text-sm)' }}>{k.name}</code></td>
      <td><code style={{ fontSize: 'var(--text-sm)', color: 'var(--fg-tertiary)' }}>{k.prefix}…</code></td>
      <td style={{ fontSize: 'var(--text-sm)', color: 'var(--fg-secondary)' }}>
        {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : 'Never'}
      </td>
      <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
        {new Date(k.created_at).toLocaleDateString()}
      </td>
      <td>
        <Button
          variant="ghost"
          size="sm"
          style={{ color: 'var(--error)' }}
          onClick={() => onRevoke(k.key_id)}
        >
          Revoke
        </Button>
      </td>
    </tr>
  )
}

function GenerateModal({
  open,
  siteId,
  onClose,
  onGenerated,
}: {
  open: boolean
  siteId: string
  onClose: () => void
  onGenerated: () => void
}) {
  const [name, setName] = useState('')
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState<string | undefined>()
  const [result, setResult] = useState<{ name: string; key: string } | null>(null)

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    if (!open) {
      setName('')
      setError(undefined)
      setResult(null)
      setGenerating(false)
    }
  }, [open])
  /* eslint-enable react-hooks/set-state-in-effect */

  const handleGenerate = async () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setError('Key name is required')
      return
    }
    if (trimmed.length > 100) {
      setError('Key name must be 100 characters or less')
      return
    }
    setGenerating(true)
    setError(undefined)
    const res = await createDeployKey(siteId, trimmed)
    if (res.ok && res.data) {
      setResult({ name: res.data.name, key: res.data.key })
      onGenerated()
    } else {
      const msg = res.error || 'Failed to create key'
      if (msg.includes('rate') || msg.includes('limit') || msg.includes('429')) {
        setError('Rate limited. Please try again later.')
      } else {
        setError(msg)
      }
      setGenerating(false)
    }
  }

  const handleClose = () => {
    setResult(null)
    setName('')
    setError(undefined)
    setGenerating(false)
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose} title="Generate Deploy Key">
      {result ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
          <p style={{ color: 'var(--warning)', fontWeight: 600, fontSize: 'var(--text-sm)' }}>
            This key will not be shown again. Copy it now.
          </p>
          <div style={{
            display: 'flex', gap: 'var(--space-2)', alignItems: 'center',
            padding: 'var(--space-3)', background: 'var(--surface-secondary)',
            borderRadius: 'var(--radius-s)', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-sm)',
            wordBreak: 'break-all',
          }}>
            <code style={{ flex: 1 }}>{result.key}</code>
            <CopyButton value={result.key} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button variant="primary" size="sm" onClick={handleClose}>
              Done
            </Button>
          </div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          <div>
            <label style={{ display: 'block', fontSize: 'var(--text-sm)', fontWeight: 500, marginBottom: 'var(--space-1)' }}>
              Key Name
            </label>
            <input
              type="text"
              placeholder="e.g. ci-bot"
              value={name}
              onChange={e => { setName(e.target.value); setError(undefined) }}
              style={{
                width: '100%', padding: 'var(--space-2) var(--space-3)', fontSize: 'var(--text-sm)',
                border: '1px solid var(--border-default)', borderRadius: 'var(--radius-s)',
                background: 'var(--surface)', color: 'var(--fg-primary)',
              }}
              disabled={generating}
              autoFocus
            />
            {error && (
              <p style={{ color: 'var(--error)', fontSize: 'var(--text-xs)', marginTop: 'var(--space-1)' }}>
                {error}
              </p>
            )}
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 'var(--space-2)' }}>
            <Button variant="ghost" size="sm" onClick={handleClose} disabled={generating}>
              Cancel
            </Button>
            <Button variant="primary" size="sm" onClick={handleGenerate} disabled={generating}>
              {generating ? 'Generating…' : 'Generate Key'}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}

function RevokeConfirmModal({
  open,
  keyName,
  onConfirm,
  onClose,
  revoking,
}: {
  open: boolean
  keyName: string
  onConfirm: () => void
  onClose: () => void
  revoking: boolean
}) {
  return (
    <Modal open={open} onClose={onClose} title="Revoke Deploy Key">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
        <p style={{ fontSize: 'var(--text-sm)' }}>
          Are you sure you want to revoke <strong>{keyName}</strong>?
          Any services using this key will lose access immediately.
        </p>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 'var(--space-2)' }}>
          <Button variant="ghost" size="sm" onClick={onClose} disabled={revoking}>
            Cancel
          </Button>
          <Button
            variant="primary"
            size="sm"
            style={{ background: 'var(--error)', borderColor: 'var(--error)' }}
            onClick={onConfirm}
            disabled={revoking}
          >
            {revoking ? 'Revoking…' : 'Revoke'}
          </Button>
        </div>
      </div>
    </Modal>
  )
}

export function SiteDeployKeysTab({ siteId }: { siteId: string }) {
  const [keys, setKeys] = useState<SiteDeployKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | undefined>()
  const [showGenerate, setShowGenerate] = useState(false)
  const [revokeTarget, setRevokeTarget] = useState<{ id: string; name: string } | null>(null)
  const [revoking, setRevoking] = useState(false)
  const [rateLimited, setRateLimited] = useState(false)

  const loadKeys = async () => {
    setLoading(true)
    setError(undefined)
    setRateLimited(false)
    try {
      const result = await getDeployKeys(siteId)
      setKeys(result)
    } catch {
      setError('Failed to load deploy keys')
    }
    setLoading(false)
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect, react-hooks/exhaustive-deps
  useEffect(() => { loadKeys() }, [siteId])

  const handleGenerated = () => {
    setShowGenerate(false)
    loadKeys()
  }

  const handleRevoke = async () => {
    if (!revokeTarget) return
    setRevoking(true)
    const res = await revokeDeployKey(siteId, revokeTarget.id)
    if (res.ok) {
      setRevokeTarget(null)
      loadKeys()
    } else {
      const msg = res.error || 'Failed to revoke key'
      if (msg.includes('rate') || msg.includes('limit') || msg.includes('429')) {
        setError('Rate limited. Please try again later.')
      } else {
        setError(msg)
      }
    }
    setRevoking(false)
  }

  if (loading) return <p className="dim">Loading deploy keys…</p>

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 style={{ fontSize: 'var(--text-sm)', fontWeight: 600, margin: 0 }}>
          Deploy Keys
        </h3>
        <Button variant="primary" size="sm" onClick={() => setShowGenerate(true)}>
          Generate Key
        </Button>
      </div>

      <GenerateModal
        open={showGenerate}
        siteId={siteId}
        onClose={() => { setShowGenerate(false); setError(undefined) }}
        onGenerated={handleGenerated}
      />

      <RevokeConfirmModal
        open={revokeTarget !== null}
        keyName={revokeTarget?.name || ''}
        onConfirm={handleRevoke}
        onClose={() => setRevokeTarget(null)}
        revoking={revoking}
      />

      {error && <p style={{ color: 'var(--error)', fontSize: 'var(--text-sm)' }}>{error}</p>}

      {rateLimited && (
        <div style={{
          padding: 'var(--space-3)', background: 'var(--warning-bg, #fff3cd)',
          borderRadius: 'var(--radius-s)', fontSize: 'var(--text-sm)', color: 'var(--warning-text, #856404)',
        }}>
          Rate limited. Too many requests. Please wait before trying again.
        </div>
      )}

      {keys.length === 0 ? (
        <EmptyState
          title="No deploy keys"
          description="Generate a deploy key to use with CI/CD pipelines or the BigBase CLI."
        >
          <Button variant="primary" size="sm" onClick={() => setShowGenerate(true)}>
            Generate Key
          </Button>
        </EmptyState>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Key</th>
                <th>Last Used</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <DeployKeyRow key={k.key_id} k={k} onRevoke={(id) => {
                  const found = keys.find(x => x.key_id === id)
                  setRevokeTarget(found ? { id: found.key_id, name: found.name } : null)
                }} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
