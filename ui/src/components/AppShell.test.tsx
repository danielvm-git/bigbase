import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { AppShell } from './AppShell'

describe('AppShell', () => {
  it('renders layout class', () => {
    const { container } = render(
      <AppShell sidebar={<nav>Sidebar</nav>} sidebarOpen={false} onToggleSidebar={vi.fn()}>
        <main>Content</main>
      </AppShell>
    )
    expect(container.querySelector('.layout')).toBeInTheDocument()
  })

  it('renders sidebar slot', () => {
    render(
      <AppShell sidebar={<nav>My Sidebar</nav>} sidebarOpen={false} onToggleSidebar={vi.fn()}>
        <main>Content</main>
      </AppShell>
    )
    expect(screen.getByText('My Sidebar')).toBeInTheDocument()
  })

  it('renders children', () => {
    render(
      <AppShell sidebar={<nav>Sidebar</nav>} sidebarOpen={false} onToggleSidebar={vi.fn()}>
        <main>Page Content</main>
      </AppShell>
    )
    expect(screen.getByText('Page Content')).toBeInTheDocument()
  })

  it('renders toggle button', () => {
    const { container } = render(
      <AppShell sidebar={<nav>Sidebar</nav>} sidebarOpen={false} onToggleSidebar={vi.fn()}>
        <main>Content</main>
      </AppShell>
    )
    const btn = container.querySelector('.sidebar-toggle')
    expect(btn).toBeInTheDocument()
    expect(btn).toHaveAttribute('aria-label', 'Open sidebar')
  })

  it('toggle button shows close label when sidebar is open', () => {
    const { container } = render(
      <AppShell sidebar={<nav>Sidebar</nav>} sidebarOpen={true} onToggleSidebar={vi.fn()}>
        <main>Content</main>
      </AppShell>
    )
    expect(container.querySelector('.sidebar-toggle')).toHaveAttribute('aria-label', 'Close sidebar')
  })

  it('calls onToggleSidebar when toggle button clicked', () => {
    const onToggle = vi.fn()
    const { container } = render(
      <AppShell sidebar={<nav>Sidebar</nav>} sidebarOpen={false} onToggleSidebar={onToggle}>
        <main>Content</main>
      </AppShell>
    )
    fireEvent.click(container.querySelector('.sidebar-toggle')!)
    expect(onToggle).toHaveBeenCalledOnce()
  })

  it('has layout-body wrapper', () => {
    const { container } = render(
      <AppShell sidebar={<nav>Sidebar</nav>} sidebarOpen={false} onToggleSidebar={vi.fn()}>
        <main>Content</main>
      </AppShell>
    )
    expect(container.querySelector('.layout-body')).toBeInTheDocument()
  })
})
