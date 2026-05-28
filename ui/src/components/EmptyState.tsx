import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon?: string
  title: string
  description?: string
  children?: ReactNode
}

export function EmptyState({ icon = '—', title, description, children }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <span className="empty-state-icon">{icon}</span>
      <span className="empty-state-title">{title}</span>
      {description && <span className="empty-state-text">{description}</span>}
      {children}
    </div>
  )
}
