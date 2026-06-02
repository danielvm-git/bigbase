import { useState } from 'react'
import { PageHeader, Tabs } from '../components'

export default function SettingsPage() {
  const [tab, setTab] = useState('account')

  const settingsTabs = [
    { id: 'account', label: 'Account' },
    { id: 'workspace', label: 'Workspace' },
    { id: 'billing', label: 'Billing' },
  ]

  return (
    <div>
      <PageHeader title="Settings" />
      <Tabs tabs={settingsTabs} active={tab} onChange={setTab} />
      {tab === 'account' && (
        <p className="dim" style={{ marginTop: 'var(--space-8)' }}>
          Email, password, and two-factor settings will appear here (e17s17).
        </p>
      )}
      {tab === 'workspace' && (
        <p className="dim" style={{ marginTop: 'var(--space-8)' }}>
          Workspace name, members, and roles will appear here (e17s17).
        </p>
      )}
      {tab === 'billing' && (
        <p className="dim" style={{ marginTop: 'var(--space-8)' }}>
          Billing and plan details (stub).
        </p>
      )}
    </div>
  )
}
