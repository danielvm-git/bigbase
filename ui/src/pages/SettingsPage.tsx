import { useState, type FormEvent } from 'react'
import { Tabs, Card, CardHeader, Input, Button, Badge, SettingsPage as SettingsPageTemplate } from '../components'
import { useCurrentUser } from '../hooks/useAuth'
import { useWorkspace, useMembers } from '../hooks/useWorkspace'
import { useBilling, useUsage } from '../hooks/useBilling'

export default function SettingsPage() {
  const [tab, setTab] = useState('account')

  const tabs = (
    <Tabs
      tabs={[
        { id: 'account', label: 'Account' },
        { id: 'workspace', label: 'Workspace' },
        { id: 'billing', label: 'Billing' },
      ]}
      active={tab}
      onChange={setTab}
    />
  )

  return (
    <SettingsPageTemplate
      title="Settings"
      subtitle="Manage your account, workspace, and billing"
      tabs={tabs}
    >
      <div id={`panel-${tab}`} role="tabpanel" aria-labelledby={`tab-${tab}`}>
        {tab === 'account' && <AccountSection />}
        {tab === 'workspace' && <WorkspaceSection />}
        {tab === 'billing' && <BillingSection />}
      </div>
    </SettingsPageTemplate>
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
      <h2 className="settings-subhead">Change password</h2>
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
    // Stub: the form is demo-only. Do NOT signal success to the user
    // and never leak internal API paths. Real submit lands in a
    // follow-up batch that wires /api/auth.
    console.info('[SettingsPage] ChangePasswordForm submit (stub):', {
      currentLen: current.length,
      nextLen: next.length,
    })
    setStatus('Demo mode — password change is not persisted')
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

  return (
    <>
      <Card className="settings-section">
        <CardHeader title="Workspace" />
        {/* `key` remounts the form when the workspace name changes,
            which is the React-blessed way to "reset" local form state
            from a prop without a setState-in-useEffect pattern. */}
        <WorkspaceNameForm key={ws?.name ?? 'empty'} initialName={ws?.name ?? ''} />
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

function WorkspaceNameForm({ initialName }: { initialName: string }) {
  const [name, setName] = useState(initialName)
  return (
    <>
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
      <h2 className="settings-subhead">Usage this month</h2>
      <div className="settings-usage">
        <div className="settings-usage-cell" data-testid="usage-functions">
          <span className="dim">Functions</span>
          <strong>{usage?.functions}</strong>
        </div>
        <div className="settings-usage-cell" data-testid="usage-storage">
          <span className="dim">Storage</span>
          <strong>{usage?.storage_mb} MB</strong>
        </div>
        <div className="settings-usage-cell" data-testid="usage-sites">
          <span className="dim">Sites</span>
          <strong>{usage?.sites}</strong>
        </div>
      </div>
    </Card>
  )
}
