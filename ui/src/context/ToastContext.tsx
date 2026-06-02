import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'

type ToastVariant = 'info' | 'success' | 'error'

interface Toast {
  id: number
  message: string
  variant: ToastVariant
}

interface ToastCtx {
  show: (message: string, variant?: ToastVariant) => void
}

const ToastContext = createContext<ToastCtx>({ show: () => {} })

let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const show = useCallback((message: string, variant: ToastVariant = 'info') => {
    const id = nextId++
    setToasts(prev => [...prev, { id, message, variant }])
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id))
    }, 4000)
  }, [])

  const dismiss = (id: number) => setToasts(prev => prev.filter(t => t.id !== id))

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      <div style={{
        position: 'fixed', bottom: 'var(--space-8)', right: 'var(--space-8)',
        display: 'flex', flexDirection: 'column', gap: 'var(--space-3)',
        zIndex: 2000, maxWidth: 360,
      }}>
        {toasts.map(t => (
          <div
            key={t.id}
            onClick={() => dismiss(t.id)}
            style={{
              padding: 'var(--space-6) var(--space-8)',
              borderRadius: 'var(--radius-s)',
              fontSize: 'var(--text-s)',
              fontWeight: 500,
              cursor: 'pointer',
              boxShadow: 'var(--shadow-l)',
              animation: 'slideIn 0.2s var(--ease-out)',
              ...(t.variant === 'success' ? { background: 'var(--success-bg)', color: 'var(--success-fg)', border: '1px solid var(--success)' } :
                 t.variant === 'error' ? { background: 'var(--error-bg)', color: 'var(--error-fg)', border: '1px solid var(--error)' } :
                 { background: 'var(--info-bg)', color: 'var(--info-fg)', border: '1px solid var(--info)' }),
            }}
          >
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  return useContext(ToastContext)
}
