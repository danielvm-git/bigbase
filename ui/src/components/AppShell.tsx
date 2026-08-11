import { type ReactNode } from 'react'

interface AppShellProps {
  sidebar: ReactNode
  children: ReactNode
  sidebarOpen: boolean
  onToggleSidebar: () => void
  className?: string
}

export function AppShell({ sidebar, children, sidebarOpen, onToggleSidebar, className = '' }: AppShellProps) {
  return (
    <div className={`layout ${className}`.trim()}>
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      <button
        type="button"
        className="sidebar-toggle"
        onClick={onToggleSidebar}
        aria-label={sidebarOpen ? 'Close sidebar' : 'Open sidebar'}
        aria-expanded={sidebarOpen}
        aria-controls="sidebar-nav"
      >
        {sidebarOpen ? '✕' : '☰'}
      </button>
      {sidebar}
      <div className="layout-body">
        {children}
      </div>
    </div>
  )
}
