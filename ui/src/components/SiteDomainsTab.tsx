import { useState, useEffect } from 'react'
import { Button, Card } from '.'
import { getDomains, addDomain, verifyDomain, deleteDomain } from '../lib/sitesData'
import type { SiteDomain } from '../types/sites'

const DOMAIN_RE = /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$/

function DomainRow({
  d,
  onVerify,
  onDelete,
  verifying,
}: {
  d: SiteDomain
  onVerify: (domain: string) => void
  onDelete: (domain: string) => void
  verifying: boolean
}) {
  const isVerified = !!d.verified_at
  const certColor = d.cert_status === 'valid' ? 'var(--success)' :
    d.cert_status === 'expiring_soon' ? 'var(--warning)' :
    d.cert_status === 'expired' ? 'var(--error)' :
    'var(--fg-tertiary)'
  const certLabel = d.cert_status === 'valid' ? 'Valid' :
    d.cert_status === 'expiring_soon' ? 'Expiring' :
    d.cert_status === 'expired' ? 'Expired' :
    d.cert_status === 'provisioning' ? 'Provisioning…' :
    d.cert_status === 'failed' ? 'Failed' :
    '—'

  return (
    <tr>
      <td>
        <code style={{ fontSize: 'var(--text-sm)' }}>{d.domain}</code>
        {!isVerified && d.verify_token && (
          <div style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)', marginTop: 'var(--space-1)' }}>
            TXT: <code>bigbase-verify={d.verify_token}</code>
          </div>
        )}
      </td>
      <td>
        {isVerified
          ? <span style={{ color: 'var(--success)', fontSize: 'var(--text-sm)' }}>✓ Verified</span>
          : <span style={{ color: 'var(--warning)', fontSize: 'var(--text-sm)' }}>Pending</span>}
      </td>
      <td>
        <span style={{ color: certColor, fontSize: 'var(--text-sm)' }}>{certLabel}</span>
        {d.cert_expires_at && (
          <div style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)', marginTop: 'var(--space-1)' }}>
            exp {new Date(d.cert_expires_at).toLocaleDateString()}
          </div>
        )}
      </td>
      <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
        {d.created_at ? new Date(d.created_at).toLocaleDateString() : '—'}
      </td>
      <td>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <Button
            variant="ghost"
            size="sm"
            disabled={verifying || isVerified}
            onClick={() => onVerify(d.domain)}
          >
            {verifying ? '…' : isVerified ? 'Verified' : 'Verify'}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            style={{ color: 'var(--error)' }}
            onClick={() => onDelete(d.domain)}
          >
            Remove
          </Button>
        </div>
      </td>
    </tr>
  )
}

function AddDomainForm({
  onAdd,
  error,
  adding,
}: {
  onAdd: (domain: string) => void
  error?: string
  adding: boolean
}) {
  const [domain, setDomain] = useState('')
  const [validationError, setValidationError] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setValidationError('')
    const trimmed = domain.trim().toLowerCase()
    if (!trimmed) {
      setValidationError('Domain is required')
      return
    }
    if (!DOMAIN_RE.test(trimmed)) {
      setValidationError('Enter a valid domain (e.g. myapp.com)')
      return
    }
    onAdd(trimmed)
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 'var(--space-2)', alignItems: 'flex-start' }}>
      <div style={{ flex: 1 }}>
        <input
          type="text"
          placeholder="myapp.com"
          value={domain}
          onChange={e => { setDomain(e.target.value); setValidationError('') }}
          style={{
            width: '100%', padding: 'var(--space-2) var(--space-3)', fontSize: 'var(--text-sm)',
            border: '1px solid var(--border-default)', borderRadius: 'var(--radius-s)',
            background: 'var(--surface)', color: 'var(--fg-primary)',
          }}
        />
        {validationError && (
          <p style={{ color: 'var(--error)', fontSize: 'var(--text-xs)', marginTop: 'var(--space-1)' }}>
            {validationError}
          </p>
        )}
      </div>
      <Button type="submit" variant="primary" size="sm" disabled={adding}>
        {adding ? 'Adding…' : 'Add Domain'}
      </Button>
      {error && <p style={{ color: 'var(--error)', fontSize: 'var(--text-xs)' }}>{error}</p>}
    </form>
  )
}

export function SiteDomainsTab({ siteId, id }: { siteId: string; id?: string }) {
  const [domains, setDomains] = useState<SiteDomain[]>([])
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [addError, setAddError] = useState<string | undefined>()
  const [verifyingDomain, setVerifyingDomain] = useState<string | null>(null)
  const [error, setError] = useState<string | undefined>()

  const loadDomains = async () => {
    setLoading(true)
    const result = await getDomains(siteId)
    setDomains(result)
    setLoading(false)
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect, react-hooks/exhaustive-deps
  useEffect(() => { loadDomains() }, [siteId])

  const handleAdd = async (domain: string) => {
    setAdding(true)
    setAddError(undefined)
    const result = await addDomain(siteId, domain)
    if (result.ok) {
      await loadDomains()
    } else {
      setAddError(result.error)
    }
    setAdding(false)
  }

  const handleVerify = async (domain: string) => {
    setVerifyingDomain(domain)
    const result = await verifyDomain(siteId, domain)
    if (result.ok) {
      await loadDomains()
    }
    setVerifyingDomain(null)
  }

  const handleDelete = async (domain: string) => {
    if (!window.confirm(`Remove domain "${domain}"?`)) return
    const result = await deleteDomain(siteId, domain)
    if (result.ok) {
      await loadDomains()
    } else {
      setError(result.error)
    }
  }

  if (loading) return <p className="dim">Loading domains…</p>

  return (
    <div id={id ?? 'panel-domains'} role="tabpanel" aria-labelledby="tab-domains" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
      <Card>
        <h3 style={{ fontSize: 'var(--text-sm)', fontWeight: 600, marginBottom: 'var(--space-3)' }}>
          Add Custom Domain
        </h3>
        <AddDomainForm onAdd={handleAdd} error={addError} adding={adding} />
        <div style={{
          marginTop: 'var(--space-3)', padding: 'var(--space-3)', background: 'var(--surface-secondary)',
          borderRadius: 'var(--radius-s)', fontSize: 'var(--text-xs)', color: 'var(--fg-secondary)',
        }}>
          <strong>DNS Setup (two records):</strong>
          <ol style={{ margin: 'var(--space-1) 0 0', paddingLeft: 'var(--space-4)' }}>
            <li>
              <strong>Route traffic</strong> — add a CNAME (or A) record pointing your
              domain to <code>yoursite.bigbase.click</code>.
            </li>
            <li>
              <strong>Prove ownership</strong> — add the TXT record shown next to the
              domain below (<code>bigbase-verify=…</code>), then click <strong>Verify</strong>.
            </li>
          </ol>
          Once verified, an SSL certificate is provisioned automatically.
        </div>
      </Card>

      {error && <p style={{ color: 'var(--error)', fontSize: 'var(--text-sm)' }}>{error}</p>}

      {domains.length === 0 ? (
        <p className="dim">No custom domains configured. Add one above.</p>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Domain</th>
                <th>Verification</th>
                <th>SSL</th>
                <th>Added</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {domains.map(d => (
                <DomainRow
                  key={d.id}
                  d={d}
                  onVerify={handleVerify}
                  onDelete={handleDelete}
                  verifying={verifyingDomain === d.domain}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
