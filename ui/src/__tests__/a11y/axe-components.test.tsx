/**
 * Structural accessibility tests for shared components.
 * Verifies ARIA roles, labels, attributes, and landmark structure.
 * (Automated axe-core integration requires jest-axe — tracked for follow-up.)
 */
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { Button } from '../../components/Button'
import { Checkbox } from '../../components/Checkbox'
import { Spinner } from '../../components/Spinner'
import { Switch } from '../../components/Switch'
import { Select } from '../../components/Select'
import { Label } from '../../components/Label'
import { Table, TableHead, TableBody, TableRow, TableCell } from '../../components/Table'
import { AppFooter } from '../../components/AppFooter'
import { Sidebar } from '../../components/Sidebar'
import { Page } from '../../components/Page'
import { ListPage } from '../../components/ListPage'
import { DetailPage } from '../../components/DetailPage'
import { SettingsPage, SettingsSection } from '../../components/SettingsPage'

const opts = [{ value: 'a', label: 'Alpha' }, { value: 'b', label: 'Beta' }]

describe('Button a11y', () => {
  it('has accessible text', () => {
    render(<Button>Save</Button>)
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })
})

describe('Checkbox a11y', () => {
  it('has accessible label', () => {
    render(<Checkbox label="Accept terms" />)
    expect(screen.getByLabelText('Accept terms')).toBeInTheDocument()
  })

  it('error has role=alert', () => {
    render(<Checkbox label="Field" error="Required" />)
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })
})

describe('Spinner a11y', () => {
  it('has role=status', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('has accessible label', () => {
    render(<Spinner aria-label="Saving" />)
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Saving')
  })
})

describe('Switch a11y', () => {
  it('has role=switch', () => {
    render(<Switch label="Enable" />)
    expect(screen.getByRole('switch')).toBeInTheDocument()
  })

  it('has aria-checked', () => {
    render(<Switch label="Enable" checked={true} onChange={() => {}} />)
    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'true')
  })
})

describe('Select a11y', () => {
  it('is labelled', () => {
    render(<Select options={opts} label="Region" />)
    expect(screen.getByLabelText('Region')).toBeInTheDocument()
  })

  it('error has role=alert', () => {
    render(<Select options={opts} label="Region" error="Required" />)
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })
})

describe('Label a11y', () => {
  it('renders as label element', () => {
    render(<Label htmlFor="x">Name</Label>)
    expect(screen.getByText('Name').closest('label')).toBeInTheDocument()
  })
})

describe('Table a11y', () => {
  it('has role=table', () => {
    render(
      <Table>
        <TableHead><TableRow><TableCell as="th">H</TableCell></TableRow></TableHead>
        <TableBody><TableRow><TableCell>D</TableCell></TableRow></TableBody>
      </Table>
    )
    expect(screen.getByRole('table')).toBeInTheDocument()
  })

  it('headers have role=columnheader', () => {
    render(
      <Table>
        <TableHead><TableRow><TableCell as="th">Name</TableCell></TableRow></TableHead>
        <TableBody><TableRow><TableCell>Row</TableCell></TableRow></TableBody>
      </Table>
    )
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument()
  })
})

describe('AppFooter a11y', () => {
  it('has landmark element (footer)', () => {
    const { container } = render(<AppFooter />)
    expect(container.querySelector('footer')).toBeInTheDocument()
  })

  it('GitHub link has accessible name', () => {
    render(<AppFooter />)
    expect(screen.getByRole('link', { name: 'GitHub' })).toBeInTheDocument()
  })
})

describe('Sidebar a11y', () => {
  it('is a navigation landmark', () => {
    render(<MemoryRouter><Sidebar id="nav" open={false}><p>nav</p></Sidebar></MemoryRouter>)
    expect(screen.getByRole('navigation')).toBeInTheDocument()
  })
})

describe('Page templates a11y', () => {
  it('Page has one h1 when title provided', () => {
    const { container } = render(<Page title="Dashboard"><p>x</p></Page>)
    expect(container.querySelectorAll('h1')).toHaveLength(1)
  })

  it('ListPage has one h1', () => {
    const { container } = render(<ListPage title="Users"><p>x</p></ListPage>)
    expect(container.querySelectorAll('h1')).toHaveLength(1)
  })

  it('DetailPage has one h1', () => {
    const { container } = render(<DetailPage title="Site"><p>x</p></DetailPage>)
    expect(container.querySelectorAll('h1')).toHaveLength(1)
  })

  it('SettingsPage has one h1', () => {
    const { container } = render(<SettingsPage title="Settings"><p>x</p></SettingsPage>)
    expect(container.querySelectorAll('h1')).toHaveLength(1)
  })

  it('SettingsSection uses h2 (no heading skip)', () => {
    const { container } = render(
      <SettingsPage title="Settings">
        <SettingsSection title="Account"><p>x</p></SettingsSection>
      </SettingsPage>
    )
    expect(container.querySelector('h1')).toBeInTheDocument()
    expect(container.querySelector('h2')).toBeInTheDocument()
    expect(container.querySelector('h3')).not.toBeInTheDocument()
  })
})
