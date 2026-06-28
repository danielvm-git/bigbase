import type { SelectHTMLAttributes } from 'react'

export interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'id'> {
  label: string
  options: SelectOption[]
  error?: string
  hint?: string
  placeholder?: string
  id?: string
}

export function Select({ label, options, error, hint, placeholder, id, className = '', ...rest }: SelectProps) {
  const selectId = id || `select-${label.toLowerCase().replace(/\s+/g, '-')}`

  return (
    <div className="input-wrapper">
      <label className="input-label" htmlFor={selectId}>{label}</label>
      <select
        id={selectId}
        className={`input${error ? ' input-error' : ''} ${className}`.trim()}
        aria-invalid={error ? 'true' : undefined}
        aria-describedby={error ? `${selectId}-error` : hint ? `${selectId}-hint` : undefined}
        {...rest}
      >
        {placeholder && <option value="" disabled>{placeholder}</option>}
        {options.map(opt => (
          <option key={opt.value} value={opt.value} disabled={opt.disabled}>
            {opt.label}
          </option>
        ))}
      </select>
      {error && <p id={`${selectId}-error`} className="input-error-message" role="alert">{error}</p>}
      {hint && !error && <p id={`${selectId}-hint`} className="input-hint">{hint}</p>}
    </div>
  )
}
