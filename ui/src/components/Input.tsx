import type { InputHTMLAttributes, TextareaHTMLAttributes, SelectHTMLAttributes, ReactNode } from 'react'

interface InputBase {
  label?: string
  error?: string
  hint?: string
}

type InputAsInput = InputBase &
  InputHTMLAttributes<HTMLInputElement> & { as?: 'input' }

type InputAsTextarea = InputBase &
  TextareaHTMLAttributes<HTMLTextAreaElement> & { as: 'textarea' }

type InputAsSelect = InputBase &
  SelectHTMLAttributes<HTMLSelectElement> & { as: 'select'; children: ReactNode }

type InputProps = InputAsInput | InputAsTextarea | InputAsSelect

export function Input(props: InputProps) {
  const { label, error, hint, className = '', ...rest } = props
  const inputClass = `input ${error ? 'input-error' : ''} ${className}`.trim()
  const id = props.id || props.name

  const inputElement =
    props.as === 'textarea' ? (
      <textarea id={id} className={inputClass} {...(rest as InputAsTextarea)} />
    ) : props.as === 'select' ? (
      <select id={id} className={inputClass} {...(rest as InputAsSelect)}>
        {(rest as InputAsSelect).children}
      </select>
    ) : (
      <input id={id} className={inputClass} {...(rest as InputAsInput)} />
    )

  return (
    <div className="input-group">
      {label && (
        <label htmlFor={id} className="input-label">
          {label}
        </label>
      )}
      {inputElement}
      {hint && !error && <span className="input-hint">{hint}</span>}
      {error && <span className="input-error-text">{error}</span>}
    </div>
  )
}
