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
  const current = ACCENT_THEMES.find(t => t.id === value) ?? ACCENT_THEMES[0]

  useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKey)
    }
  }, [open])

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
        type="button"
        className="theme-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={label}
        onClick={() => setOpen(o => !o)}
      >
        <span className="theme-dot" style={dotStyle(current.brand500)} />
        <span className="theme-trigger-label">{current.label}</span>
        <span className="theme-trigger-chev" aria-hidden="true">▾</span>
      </button>
      {open && (
        <div className="theme-menu" role="listbox" aria-label={label}>
          {ACCENT_THEMES.map(t => (
            <button
              type="button"
              key={t.id}
              role="option"
              aria-selected={t.id === value}
              className={`theme-menu-item${t.id === value ? ' active' : ''}`}
              onClick={() => {
                onChange(t.id)
                setOpen(false)
              }}
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
