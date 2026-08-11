import { type ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { Icon, type IconName } from './Icon'

export interface SidebarNavItem {
  to: string
  label: string
  icon: IconName
  end?: boolean
}

interface SidebarSectionProps {
  title: string
  items: SidebarNavItem[]
}

interface SidebarProps {
  id: string
  open: boolean
  children: ReactNode
  footer?: ReactNode
  className?: string
}

export function SidebarSection({ title, items }: SidebarSectionProps) {
  return (
    <div className="sidebar-section">
      <div className="sidebar-section-title">{title}</div>
      <ul className="sidebar-nav">
        {items.map(item => (
          <li key={item.to}>
            <NavLink to={item.to} end={item.end} aria-label={item.label}>
              <Icon name={item.icon} size={18} />
              <span>{item.label}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  )
}

export function Sidebar({ id, open, children, footer, className = '' }: SidebarProps) {
  return (
    <nav
      id={id}
      className={`sidebar${open ? ' sidebar-open' : ''} ${className}`.trim()}
    >
      {children}
      <div className="sidebar-spacer" />
      {footer}
    </nav>
  )
}
