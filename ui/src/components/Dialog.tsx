import { useEffect, useId, useRef, type ReactNode } from 'react'

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

const FOCUSABLE =
  'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'

function getFocusable(dialog: HTMLDivElement): HTMLElement[] {
  return Array.from(dialog.querySelectorAll<HTMLElement>(FOCUSABLE))
    .filter(el => !el.hasAttribute('disabled'))
}

function trapTabKey(e: KeyboardEvent, dialog: HTMLDivElement) {
  if (e.key !== 'Tab') return
  const focusable = getFocusable(dialog)
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement
  if (!dialog.contains(active)) {
    e.preventDefault()
    first.focus()
    return
  }
  if (e.shiftKey && active === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && active === last) {
    e.preventDefault()
    first.focus()
  }
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
  const dialogRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<Element | null>(null)
  const onCloseRef = useRef(onClose)
  useEffect(() => { onCloseRef.current = onClose })

  useEffect(() => {
    if (!open) return
    triggerRef.current = document.activeElement
    const dialog = dialogRef.current

    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCloseRef.current()
        return
      }
      if (dialog) trapTabKey(e, dialog)
    }

    document.addEventListener('keydown', onKey)
    const focusable = dialog ? getFocusable(dialog) : []
    if (focusable.length > 0) {
      focusable[0].focus()
    } else {
      dialog?.focus()
    }

    return () => {
      document.removeEventListener('keydown', onKey)
      if (triggerRef.current instanceof HTMLElement) {
        triggerRef.current.focus()
      }
    }
  }, [open])

  if (!open) return null

  return (
    <div className="dialog-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={`dialog ${className}`.trim()}
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
