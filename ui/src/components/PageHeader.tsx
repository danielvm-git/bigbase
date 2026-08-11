import type { ReactNode } from 'react'
import { useLocation } from 'react-router-dom'

interface PageHeaderProps {
  title: string
  subtitle?: string
  children?: ReactNode
}

/**
 * Route map for the depth-1 location indicator (WCAG 2.4.8).
 * Only top-level pages are listed — detail pages render their own
 * Breadcrumb, so PageHeader adds no indicator there.
 */
const PAGE_LOCATIONS: Record<string, { section: string; page: string }> = {
  '/': { section: 'Overview', page: 'Dashboard' },
  '/deploy': { section: 'Build', page: 'Sites' },
  '/functions': { section: 'Build', page: 'Functions' },
  '/data': { section: 'Data', page: 'Data Studio' },
  '/sql': { section: 'Data', page: 'SQL Editor' },
  '/storage': { section: 'Data', page: 'Storage' },
  '/users': { section: 'Auth', page: 'Users' },
  '/messaging': { section: 'Engage', page: 'Messaging' },
  '/repos': { section: 'DevOps', page: 'Git Repos' },
  '/cici': { section: 'DevOps', page: 'CI/CD' },
  '/monitoring': { section: 'DevOps', page: 'Monitoring' },
  '/forge': { section: 'DevOps', page: 'Forge' },
  '/realtime': { section: 'DevOps', page: 'Realtime' },
  '/events': { section: 'DevOps', page: 'Events' },
  '/settings': { section: 'Admin', page: 'Settings' },
}

export function PageHeader({ title, subtitle, children }: PageHeaderProps) {
  const { pathname } = useLocation()
  const location = PAGE_LOCATIONS[pathname] ?? null

  return (
    <div className="page-header">
      {location && (
        <nav className="breadcrumb page-location" aria-label="You are here">
          <ol>
            <li className="location-section">{location.section}</li>
            <li className="breadcrumb-current location-page">
              <span aria-current="page">{location.page}</span>
            </li>
          </ol>
        </nav>
      )}
      <div>
        <h1 className="page-title">{title}</h1>
        {subtitle && <p className="page-subtitle">{subtitle}</p>}
      </div>
      {children && <div className="page-header-actions">{children}</div>}
    </div>
  )
}
