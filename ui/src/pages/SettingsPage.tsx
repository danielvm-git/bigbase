import { useState, type FormEvent } from 'react'
import { PageHeader, Tabs, Card, CardHeader, Input, Button, Badge } from '../components'
import { useCurrentUser } from '../hooks/useAuth'
import { useWorkspace, useMembers } from '../hooks/useWorkspace'
import { useBilling, useUsage } from '../hooks/useBilling'

export default function SettingsPage() {
  const [tab, setTab] = useState('account')

  return (
    <div>
      <PageHeader
        title="Settings"
        subtitle="Manage your account, workspace, and billing"
      />
      <Tabs
        tabs={[
          { id: 'account', label: 'Account' },
          { id: 'workspace', label: 'Workspace' },
          { id: 'billing', label: 'Billing' },
        ]}
        active={tab}
        onChange={setTab}
      />
      {tab === 'account' && <AccountSection />}
      {tab === 'workspace' && <WorkspaceSection />}
      {tab === 'billing' && <BillingSection />}
    </div>
  )
}

function AccountSection() {
  const { data: user } = useCurrentUser()
  return (
    <Card className="settings-section">
      <CardHeader title="Account" />
      <div className="settings-row">
        <label className="dim">Email</label>
        <span>{user?.email}</span>
      </div>
      <div className="settings-row">
        <label className="dim">Name</label>
        <span>{user?.name}</span>
      </div>
      <div className="settings-row">
        <label className="dim">Two-factor authentication</label>
        <Badge variant="info">Not configured</Badge>
      </div>
      <hr className="settings-divider" />
      <h3 className="settings-subhead">Change password</h3>
      <ChangePasswordForm />
    </Card>
  )
}

function ChangePasswordForm() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [status, setStatus] = useState<string | null>(null)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (next.length < 8) {
      setStatus('Password must be at least 8 characters')
      return
    }
    setStatus('Password updated (mock — wire to /api/auth in production)')
    setCurrent('')
    setNext('')
  }

  return (
    <form onSubmit={handleSubmit} className="settings-form">
      <Input
        as="input"
        type="password"
        label="Current password"
        value={current}
        onChange={e => setCurrent(e.target.value)}
        name="current"
      />
      <Input
        as="input"
        type="password"
        label="New password"
        value={next}
        onChange={e => setNext(e.target.value)}
        name="next"
        hint="At least 8 characters"
      />
      <div className="settings-form-actions">
        <Button type="submit" variant="primary">
          Update password
        </Button>
        {status && <span className="dim">{status}</span>}
      </div>
    </form>
  )
}

function WorkspaceSection() {
  const { data: ws } = useWorkspace()
  const { data: members } = useMembers()
  const [name, setName] = useState(ws?.name ?? '')

  return (
    <>
      <Card className="settings-section">
        <CardHeader title="Workspace" />
        <Input
          as="input"
          label="Workspace name"
          value={name}
          onChange={e => setName(e.target.value)}
          name="workspace-name"
        />
        <div className="settings-form-actions">
          <Button variant="primary">Save</Button>
        </div>
      </Card>
      <Card className="settings-section">
        <CardHeader title={`Members (${members?.length ?? 0})`} />
        <ul className="settings-member-list">
          {members?.map(m => (
            <li key={m.id} className="settings-member">
              <span>{m.email}</span>
              <Badge variant={m.role === 'admin' ? 'accent' : 'neutral'}>{m.role}</Badge>
            </li>
          ))}
        </ul>
      </Card>
    </>
  )
}

function BillingSection() {
  const { data: billing } = useBilling()
  const { data: usage } = useUsage()
  return (
    <Card className="settings-section">
      <CardHeader title="Billing" />
      <div className="settings-row">
        <label className="dim">Current plan</label>
        <Badge variant="success">{billing?.plan}</Badge>
      </div>
      <div className="settings-row">
        <label className="dim">Renews</label>
        <span>{billing?.renews}</span>
      </div>
      <hr className="settings-divider" />
      <h3 className="settings-subhead">Usage this month</h3>
      <div className="settings-usage">
        <div className="settings-usage-cell">
          <span className="dim">Functions</span>
          <strong>{usage?.functions}</strong>
        </div>
        <div className="settings-usage-cell">
          <span className="dim">Storage</span>
          <strong>{usage?.storage_mb} MB</strong>
        </div>
        <div className="settings-usage-cell">
          <span className="dim">Sites</span>
          <strong>{usage?.sites}</strong>
        </div>
      </div>
    </Card>
  )
}
