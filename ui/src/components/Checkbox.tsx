import { useRef, useEffect, type InputHTMLAttributes } from 'react'

interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: string
  error?: string
  indeterminate?: boolean
}

export function Checkbox({ label, error, indeterminate, id, className = '', ...rest }: CheckboxProps) {
  const ref = useRef<HTMLInputElement>(null)
  const inputId = id || `checkbox-${label.toLowerCase().replace(/\s+/g, '-')}`

  useEffect(() => {
    if (ref.current) {
      ref.current.indeterminate = indeterminate ?? false
    }
  }, [indeterminate])

  return (
    <div className={`checkbox-wrapper ${className}`.trim()}>
      <label className="checkbox-label" htmlFor={inputId}>
        <input
          ref={ref}
          id={inputId}
          type="checkbox"
          className={`checkbox${error ? ' checkbox-error' : ''}`}
          aria-invalid={error ? 'true' : undefined}
          aria-describedby={error ? `${inputId}-error` : undefined}
          {...rest}
        />
        <span className="checkbox-text">{label}</span>
      </label>
      {error && (
        <p id={`${inputId}-error`} className="input-error-message" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
