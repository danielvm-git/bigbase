import React, { useRef, useState, useEffect } from 'react';
import { THEME_META } from './tokens';
import { useTheme } from './ThemeContext';

/**
 * ThemePicker — sidebar-footer dropdown that drives the 12 month accent themes.
 * Reads/writes through useTheme(); fully keyboard + screen-reader operable
 * (listbox/option roles, Esc + click-outside to close).
 */
export function ThemePicker() {
  const { accent, setAccent } = useTheme();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false); };
    const onEsc = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onEsc);
    return () => { document.removeEventListener('mousedown', onDown); document.removeEventListener('keydown', onEsc); };
  }, [open]);

  const cur = THEME_META.find((t) => t.id === accent) ?? THEME_META[0];

  return (
    <div className="theme-picker" ref={ref}>
      <button
        type="button"
        className="theme-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label="Accent theme"
        onClick={() => setOpen((o) => !o)}
      >
        <span className="theme-dot" style={{ background: cur.swatch }} />
        <span className="theme-trigger-label">{cur.label}</span>
      </button>
      {open && (
        <div className="theme-menu" role="listbox" aria-label="Accent theme">
          {THEME_META.map((t) => (
            <button
              key={t.id}
              type="button"
              role="option"
              aria-selected={t.id === accent}
              className={`theme-menu-item ${t.id === accent ? 'active' : ''}`}
              onClick={() => { setAccent(t.id); setOpen(false); }}
            >
              <span className="theme-dot" style={{ background: t.swatch }} />
              <span className="theme-menu-name">{t.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
