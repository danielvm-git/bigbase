import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { PageHeader } from './PageHeader'

function renderAt(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <Routes>
        <Route path="*" element={<PageHeader title="Dashboard" />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('PageHeader', () => {
  it('renders the title', () => {
    renderAt('/')
    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
  })

  it('renders subtitle when provided', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="*" element={<PageHeader title="Sites" subtitle="Manage your deployments" />} />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('Manage your deployments')).toBeInTheDocument()
    expect(screen.getByText('Manage your deployments').className).toContain('page-subtitle')
  })

  it('omits subtitle element when subtitle is absent', () => {
    const { container } = renderAt('/')
    expect(container.querySelector('.page-subtitle')).toBeNull()
  })

  it('renders children in actions area', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route
            path="*"
            element={
              <PageHeader title="With Action">
                <button type="button">Add</button>
              </PageHeader>
            }
          />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByRole('button', { name: 'Add' })).toBeInTheDocument()
  })

  it('renders a "You are here" location indicator on top-level pages (WCAG 2.4.8)', () => {
    renderAt('/monitoring')
    const location = screen.getByRole('navigation', { name: 'You are here' })
    expect(location).toBeInTheDocument()
    expect(screen.getByText('DevOps')).toBeInTheDocument()
    expect(screen.getByText('Monitoring')).toHaveAttribute('aria-current', 'page')
  })

  it('omits the location indicator on detail pages that render their own breadcrumb', () => {
    renderAt('/deploy/acme-prod')
    expect(screen.queryByRole('navigation', { name: 'You are here' })).toBeNull()
  })
})
