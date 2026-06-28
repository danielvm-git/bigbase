import { type ReactNode } from 'react'

type AlertVariant = 'info' | 'success' | 'warning' | 'error'

interface AlertProps {
  children: ReactNode
  variant?: AlertVariant
  title?: string
  dismissible?: boolean
  onDismiss?: () => void
  className?: string
}

export function Alert({ children, variant = 'info', title, dismissible = false, onDismiss, className = '' }: AlertProps) {
  return (
    <div role="alert" className={`alert alert-${variant} ${className}`.trim()}>
      <div className="alert-body">
        {title && <strong className="alert-title">{title}</strong>}
        <span className="alert-message">{children}</span>
      </div>
      {dismissible && (
        <button
          type="button"
          className="btn btn-ghost btn-sm alert-dismiss"
          onClick={onDismiss}
          aria-label="Dismiss alert"
        >
          ✕
        </button>
      )}
    </div>
  )
}
