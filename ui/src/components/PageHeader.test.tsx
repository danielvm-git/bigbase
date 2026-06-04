import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PageHeader } from './PageHeader'

describe('PageHeader', () => {
  it('renders the title', () => {
    render(<PageHeader title="Dashboard" />)
    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
  })

  it('renders subtitle when provided', () => {
    render(<PageHeader title="Sites" subtitle="Manage your deployments" />)
    expect(screen.getByText('Manage your deployments')).toBeInTheDocument()
    expect(screen.getByText('Manage your deployments').className).toContain('page-subtitle')
  })

  it('omits subtitle element when subtitle is absent', () => {
    const { container } = render(<PageHeader title="Only" />)
    expect(container.querySelector('.page-subtitle')).toBeNull()
  })

  it('renders children in actions area', () => {
    render(
      <PageHeader title="With Action">
        <button type="button">Add</button>
      </PageHeader>,
    )
    expect(screen.getByRole('button', { name: 'Add' })).toBeInTheDocument()
  })
})
