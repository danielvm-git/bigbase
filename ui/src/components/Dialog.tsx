import { useId, type ReactNode } from 'react'

interface DialogProps {
  open: boolean
  title: string
  children: ReactNode
  onClose: () => void
  onConfirm?: () => void
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  loading?: boolean
  className?: string
}

export function Dialog({
  open,
  title,
  children,
  onClose,
  onConfirm,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  loading = false,
  className = '',
}: DialogProps) {
  const titleId = useId()

  if (!open) return null

  return (
    <div className="dialog-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={`dialog ${className}`.trim()}
        onKeyDown={(e) => { if (e.key === 'Escape') onClose() }}
        tabIndex={-1}
      >
        <div className="dialog-header">
          <h2 id={titleId} className="dialog-title">{title}</h2>
          <button
            type="button"
            className="btn btn-ghost btn-sm dialog-close"
            onClick={onClose}
            aria-label="Close dialog"
          >
            ✕
          </button>
        </div>
        <div className="dialog-body">
          {children}
        </div>
        {(onConfirm || cancelLabel) && (
          <div className="dialog-footer">
            <button type="button" className="btn btn-secondary btn-sm" onClick={onClose} disabled={loading}>
              {cancelLabel}
            </button>
            {onConfirm && (
              <button
                type="button"
                className={`btn ${danger ? 'btn-danger' : 'btn-primary'} btn-sm`}
                onClick={onConfirm}
                disabled={loading}
              >
                {confirmLabel}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
