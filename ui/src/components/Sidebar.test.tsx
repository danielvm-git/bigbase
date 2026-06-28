import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { Sidebar, SidebarSection } from './Sidebar'

const items = [
  { to: '/home', label: 'Home', icon: 'layout-dashboard' as const },
  { to: '/data', label: 'Data', icon: 'database' as const },
]

function wrap(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

describe('SidebarSection', () => {
  it('renders section title', () => {
    wrap(<SidebarSection title="Build" items={items} />)
    expect(screen.getByText('Build')).toBeInTheDocument()
  })

  it('renders nav items', () => {
    wrap(<SidebarSection title="Build" items={items} />)
    expect(screen.getByText('Home')).toBeInTheDocument()
    expect(screen.getByText('Data')).toBeInTheDocument()
  })

  it('renders links for each item', () => {
    wrap(<SidebarSection title="Build" items={items} />)
    expect(screen.getByRole('link', { name: /Home/ })).toHaveAttribute('href', '/home')
  })
})

describe('Sidebar', () => {
  it('renders nav element with id', () => {
    wrap(<Sidebar id="sidebar-nav" open={false}><p>content</p></Sidebar>)
    expect(screen.getByRole('navigation')).toHaveAttribute('id', 'sidebar-nav')
  })

  it('has sidebar-open class when open', () => {
    wrap(<Sidebar id="sidebar-nav" open={true}><p>content</p></Sidebar>)
    expect(screen.getByRole('navigation').className).toContain('sidebar-open')
  })

  it('does not have sidebar-open class when closed', () => {
    wrap(<Sidebar id="sidebar-nav" open={false}><p>content</p></Sidebar>)
    expect(screen.getByRole('navigation').className).not.toContain('sidebar-open')
  })

  it('renders children', () => {
    wrap(<Sidebar id="sidebar-nav" open={false}><p>Sidebar content</p></Sidebar>)
    expect(screen.getByText('Sidebar content')).toBeInTheDocument()
  })

  it('renders footer slot', () => {
    wrap(
      <Sidebar id="sidebar-nav" open={false} footer={<div>Footer slot</div>}>
        <p>nav</p>
      </Sidebar>
    )
    expect(screen.getByText('Footer slot')).toBeInTheDocument()
  })
})
