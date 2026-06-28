import type { LabelHTMLAttributes, ReactNode } from 'react'

interface LabelProps extends LabelHTMLAttributes<HTMLLabelElement> {
  children: ReactNode
  required?: boolean
}

export function Label({ children, required, className = '', ...rest }: LabelProps) {
  return (
    <label className={`input-label ${className}`.trim()} {...rest}>
      {children}
      {required && <span className="label-required" aria-hidden="true"> *</span>}
    </label>
  )
}
