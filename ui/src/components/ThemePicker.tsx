import { useEffect, useRef, useState } from 'react'
import type { AccentId } from '../context/accentThemes'
import { ACCENT_THEMES } from '../context/accentThemes'

interface ThemePickerProps {
  value: AccentId
  onChange: (id: AccentId) => void
  label?: string
}

export function ThemePicker({ value, onChange, label = 'Accent theme' }: ThemePickerProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([])
  const current = ACCENT_THEMES.find(t => t.id === value) ?? ACCENT_THEMES[0]
  const [activeIndex, setActiveIndex] = useState(() =>
    Math.max(0, ACCENT_THEMES.findIndex(t => t.id === value)),
  )

  const close = (returnFocus: boolean) => {
    setOpen(false)
    // a11y: return focus to the trigger so keyboard users don't get
    // dropped on <body> when the popover (or the option they
    // selected) is unmounted.
    if (returnFocus) triggerRef.current?.focus()
  }

  // Move focus into the menu, landing on the currently selected item,
  // whenever the menu opens. Imperative only — activeIndex is synced in
  // the trigger's onClick (user event context) to satisfy the
  // react-hooks/set-state-in-effect rule.
  useEffect(() => {
    if (!open) return
    const idx = Math.max(0, ACCENT_THEMES.findIndex(t => t.id === value))
    itemRefs.current[idx]?.focus()
  }, [open, value])

  useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close(true)
    }
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close(true)
    }
    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKey)
    }
  }, [open])

  const moveTo = (index: number) => {
    setActiveIndex(index)
    itemRefs.current[index]?.focus()
  }

  const selectItem = (index: number) => {
    onChange(ACCENT_THEMES[index].id)
    close(true)
  }

  // ARIA menu keyboard pattern: roving focus with Arrow/Home/End,
  // Enter/Space activates the focused item.
  const handleMenuKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    const count = ACCENT_THEMES.length
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        moveTo((activeIndex + 1) % count)
        break
      case 'ArrowUp':
        e.preventDefault()
        moveTo((activeIndex - 1 + count) % count)
        break
      case 'Home':
        e.preventDefault()
        moveTo(0)
        break
      case 'End':
        e.preventDefault()
        moveTo(count - 1)
        break
      case 'Enter':
      case ' ':
        e.preventDefault()
        selectItem(activeIndex)
        break
    }
  }

  // Close the popover when focus leaves it entirely (e.g. Tab past the
  // last item) — do NOT yank focus back, the user is moving on.
  const handleMenuBlur = (e: React.FocusEvent<HTMLDivElement>) => {
    if (!ref.current?.contains(e.relatedTarget as Node | null)) {
      setOpen(false)
    }
  }

  const dotStyle = (rgb: string): React.CSSProperties => ({
    width: 12,
    height: 12,
    borderRadius: 'var(--radius-full)',
    background: `rgb(${rgb})`,
    flexShrink: 0,
  })

  return (
    <div className="theme-picker" ref={ref}>
      <button
        ref={triggerRef}
        type="button"
        className="theme-trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={label}
        onClick={() => {
          if (!open) {
            const idx = Math.max(0, ACCENT_THEMES.findIndex(t => t.id === value))
            setActiveIndex(idx)
          }
          setOpen(o => !o)
        }}
      >
        <span className="theme-dot" style={dotStyle(current.brand500)} />
        <span className="theme-trigger-label">{current.label}</span>
        <span className="theme-trigger-chev" aria-hidden="true">▾</span>
      </button>
      {open && (
        <div
          className="theme-menu"
          role="menu"
          aria-label={label}
          onKeyDown={handleMenuKeyDown}
          onBlur={handleMenuBlur}
        >
          {ACCENT_THEMES.map((t, i) => (
            <button
              type="button"
              key={t.id}
              ref={el => {
                itemRefs.current[i] = el
              }}
              role="menuitemradio"
              aria-checked={t.id === value}
              tabIndex={i === activeIndex ? 0 : -1}
              className={`theme-menu-item${t.id === value ? ' active' : ''}`}
              onClick={() => selectItem(i)}
            >
              <span className="theme-dot" style={dotStyle(t.brand500)} />
              <span className="theme-menu-name">{t.label}</span>
              {t.id === value && (
                <span className="theme-menu-check" aria-hidden="true">✓</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
