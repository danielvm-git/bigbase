import { useState, useRef, useEffect, isValidElement, type KeyboardEvent, type ReactNode } from 'react'

export interface DropdownItem {
  id: string
  label?: string
  icon?: ReactNode
  danger?: boolean
  disabled?: boolean
  divider?: true
}

interface DropdownMenuProps {
  trigger: ReactNode
  items: DropdownItem[]
  onSelect?: (id: string) => void
  className?: string
}

export function DropdownMenu({ trigger, items, onSelect, className = '' }: DropdownMenuProps) {
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node) &&
          triggerRef.current && !triggerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  function handleItemClick(item: DropdownItem) {
    if (item.disabled || item.divider) return
    setOpen(false)
    onSelect?.(item.id)
  }

  function handleTriggerKeyDown(e: KeyboardEvent<HTMLSpanElement>) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      setOpen(o => !o)
    }
  }

  function handleKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === 'Escape') setOpen(false)
    if (e.key === 'ArrowDown') {
      const items = menuRef.current?.querySelectorAll('[role="menuitem"]:not([disabled])')
      const first = items?.[0] as HTMLElement
      first?.focus()
    }
  }

  // If the trigger child is already a native control (button/link) it is the
  // accessible element and the wrapper span stays non-interactive. Otherwise
  // the span itself must be keyboard-accessible (WCAG 2.1.1).
  const triggerIsInteractive =
    isValidElement(trigger) &&
    typeof trigger.type === 'string' &&
    (trigger.type === 'button' || trigger.type === 'a')

  return (
    <div className={`dropdown ${className}`.trim()}>
      <span
        ref={triggerRef}
        onClick={() => setOpen(o => !o)}
        {...(triggerIsInteractive
          ? {}
          : {
              role: 'button' as const,
              tabIndex: 0,
              'aria-haspopup': 'menu' as const,
              'aria-expanded': open,
              onKeyDown: handleTriggerKeyDown,
            })}
      >
        {trigger}
      </span>
      {open && (
        <div
          ref={menuRef}
          role="menu"
          className="dropdown-menu"
          onKeyDown={handleKeyDown}
        >
          {items.map(item => {
            if (item.divider) {
              return <div key={item.id} className="dropdown-divider" role="separator" />
            }
            return (
              <button
                key={item.id}
                role="menuitem"
                type="button"
                disabled={item.disabled}
                className={`dropdown-item${item.danger ? ' danger' : ''}${item.disabled ? ' disabled' : ''}`}
                onClick={() => handleItemClick(item)}
              >
                {item.icon && <span className="dropdown-item-icon">{item.icon}</span>}
                {item.label}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
