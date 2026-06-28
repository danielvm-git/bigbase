import { type ReactNode } from 'react'

interface PageProps {
  children: ReactNode
  title?: string
  subtitle?: string
  actions?: ReactNode
  className?: string
}

export function Page({ children, title, subtitle, actions, className = '' }: PageProps) {
  return (
    <div className={`page ${className}`.trim()}>
      {(title || actions) && (
        <div className="page-header">
          <div className="page-header-text">
            {title && <h1 className="page-title">{title}</h1>}
            {subtitle && <p className="page-subtitle">{subtitle}</p>}
          </div>
          {actions && <div className="page-actions">{actions}</div>}
        </div>
      )}
      <div className="page-content">
        {children}
      </div>
    </div>
  )
}
