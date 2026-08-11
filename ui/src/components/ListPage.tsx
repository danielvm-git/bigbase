import { type ReactNode } from 'react'
import { Spinner } from './Spinner'

interface ListPageProps {
  title: string
  children: ReactNode
  actions?: ReactNode
  filters?: ReactNode
  pagination?: ReactNode
  loading?: boolean
  loadingMessage?: string
  error?: string
  empty?: boolean
  emptyMessage?: string
  className?: string
}

export function ListPage({
  title,
  children,
  actions,
  filters,
  pagination,
  loading,
  loadingMessage,
  error,
  empty,
  emptyMessage = 'No items found.',
  className = '',
}: ListPageProps) {
  return (
    <div className={`page list-page ${className}`.trim()}>
      <div className="page-header">
        <h1 className="page-title">{title}</h1>
        {actions && <div className="page-actions">{actions}</div>}
      </div>
      {filters && <div className="list-page-filters">{filters}</div>}
      <div className="list-page-content">
        {loading && (loadingMessage
          ? <div className="loading" role="status" aria-busy="true">{loadingMessage}</div>
          : <Spinner aria-label="Loading" />
        )}
        {!loading && error && <div role="alert" className="error-banner">{error}</div>}
        {!loading && !error && empty && <p className="empty-message">{emptyMessage}</p>}
        {!loading && !error && !empty && children}
      </div>
      {pagination && <div className="list-page-pagination">{pagination}</div>}
    </div>
  )
}
