import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon?: string
  title: string
  description?: string
  children?: ReactNode
}

export function EmptyState({ icon = '○', title, description, children }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <span className="empty-state-icon" aria-hidden="true">{icon}</span>
      <h2 className="empty-state-title">{title}</h2>
      {description && <span className="empty-state-text">{description}</span>}
      {children}
    </div>
  )
}
