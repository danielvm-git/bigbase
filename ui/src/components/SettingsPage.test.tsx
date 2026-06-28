import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SettingsPage, SettingsSection } from './SettingsPage'

describe('SettingsPage', () => {
  it('renders title', () => {
    render(<SettingsPage title="Settings"><p>x</p></SettingsPage>)
    expect(screen.getByRole('heading', { name: 'Settings' })).toBeInTheDocument()
  })

  it('renders children', () => {
    render(<SettingsPage title="Settings"><p>Settings content</p></SettingsPage>)
    expect(screen.getByText('Settings content')).toBeInTheDocument()
  })

  it('renders tabs when provided', () => {
    const tabs = <nav aria-label="Settings tabs"><a href="#account">Account</a></nav>
    render(<SettingsPage title="Settings" tabs={tabs}><p>x</p></SettingsPage>)
    expect(screen.getByRole('navigation', { name: 'Settings tabs' })).toBeInTheDocument()
  })
})

describe('SettingsSection', () => {
  it('renders section title', () => {
    render(<SettingsSection title="Account"><p>section content</p></SettingsSection>)
    expect(screen.getByRole('heading', { name: 'Account' })).toBeInTheDocument()
  })

  it('renders children', () => {
    render(<SettingsSection title="Account"><p>Form fields</p></SettingsSection>)
    expect(screen.getByText('Form fields')).toBeInTheDocument()
  })

  it('renders description when provided', () => {
    render(<SettingsSection title="Security" description="Manage your security settings"><p>x</p></SettingsSection>)
    expect(screen.getByText('Manage your security settings')).toBeInTheDocument()
  })

  it('renders actions when provided', () => {
    render(<SettingsSection title="Account" actions={<button>Save</button>}><p>x</p></SettingsSection>)
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })
})
