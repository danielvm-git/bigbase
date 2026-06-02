import { createContext } from 'react'

type ToastVariant = 'info' | 'success' | 'error'

export interface Toast {
  id: number
  message: string
  variant: ToastVariant
}

export interface ToastCtx {
  show: (message: string, variant?: ToastVariant) => void
}

export const ToastContext = createContext<ToastCtx>({ show: () => {} })
