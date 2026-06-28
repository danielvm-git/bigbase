import { useState, type KeyboardEvent } from 'react'

type SwitchSize = 'sm' | 'md'

interface SwitchProps {
  label: string
  checked?: boolean
  defaultChecked?: boolean
  disabled?: boolean
  size?: SwitchSize
  onChange?: (checked: boolean) => void
  className?: string
}

export function Switch({
  label,
  checked: controlledChecked,
  defaultChecked = false,
  disabled = false,
  size = 'md',
  onChange,
  className = '',
}: SwitchProps) {
  const [internalChecked, setInternalChecked] = useState(defaultChecked)
  const isControlled = controlledChecked !== undefined
  const isChecked = isControlled ? controlledChecked : internalChecked

  function toggle() {
    if (disabled) return
    if (!isControlled) setInternalChecked(v => !v)
    onChange?.(!isChecked)
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      toggle()
    }
  }

  return (
    <label className={`switch-wrapper switch-wrapper-${size} ${className}`.trim()}>
      <button
        type="button"
        role="switch"
        aria-checked={isChecked}
        aria-disabled={disabled || undefined}
        aria-label={label}
        className={`switch switch-${size}${isChecked ? ' switch-on' : ''}${disabled ? ' switch-disabled' : ''}`}
        onClick={toggle}
        onKeyDown={handleKeyDown}
        disabled={disabled}
      >
        <span className="switch-thumb" aria-hidden="true" />
      </button>
      <span className="switch-label">{label}</span>
    </label>
  )
}
