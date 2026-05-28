import type { ReactNode } from 'react'

interface PageHeaderProps {
  title: string
  children?: ReactNode
}

export function PageHeader({ title, children }: PageHeaderProps) {
  return (
    <div className="page-header">
      <h1 className="page-title">{title}</h1>
      {children}
    </div>
  )
}
