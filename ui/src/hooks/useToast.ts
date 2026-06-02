import { useContext } from 'react'
import { ToastContext } from '../context/toastState'

export function useToast() {
  return useContext(ToastContext)
}
