import { type ReactNode } from 'react'
import { Spinner } from './Spinner'

interface DetailPageProps {
  title: string
  children: ReactNode
  onBack?: () => void
  tabs?: ReactNode
  actions?: ReactNode
  loading?: boolean
  error?: string
  className?: string
}

export function DetailPage({
  title,
  children,
  onBack,
  tabs,
  actions,
  loading,
  error,
  className = '',
}: DetailPageProps) {
  return (
    <div className={`page detail-page ${className}`.trim()}>
      <div className="page-header">
        <div className="page-header-left">
          {onBack && (
            <button type="button" className="btn btn-ghost btn-sm" onClick={onBack} aria-label="Back">
              ← Back
            </button>
          )}
          <h1 className="page-title">{title}</h1>
        </div>
        {actions && <div className="page-actions">{actions}</div>}
      </div>
      {tabs && <div className="detail-page-tabs">{tabs}</div>}
      <div className="detail-page-content">
        {loading && <Spinner aria-label="Loading" />}
        {!loading && error && <div role="alert" className="error-banner">{error}</div>}
        {!loading && !error && children}
      </div>
    </div>
  )
}
