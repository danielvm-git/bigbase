import React, {
  createContext, useContext, useState, useEffect, useCallback, ReactNode,
} from 'react';
import type { AccentTheme, ColorScheme } from './tokens';

/**
 * Theme context — drives the accent theme + color scheme by writing
 * `data-accent` / `data-theme` onto <html>, and persisting both to
 * localStorage. Every component reads CSS role tokens, so a change here
 * re-themes the whole app with no per-component work.
 */
export interface ThemeContextValue {
  accent: AccentTheme;
  scheme: ColorScheme;
  setAccent: (a: AccentTheme) => void;
  setScheme: (s: ColorScheme) => void;
  toggleScheme: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

const ACCENT_KEY = 'bigbase-theme';
const SCHEME_KEY = 'bigbase-dark';

function readAccent(): AccentTheme {
  if (typeof localStorage === 'undefined') return 'default';
  return (localStorage.getItem(ACCENT_KEY) as AccentTheme) || 'default';
}
function readScheme(): ColorScheme {
  if (typeof localStorage === 'undefined') return 'light';
  return localStorage.getItem(SCHEME_KEY) === '1' ? 'dark' : 'light';
}

export interface ThemeProviderProps {
  children: ReactNode;
  /** Override the initial accent (otherwise read from localStorage). */
  defaultAccent?: AccentTheme;
  /** Override the initial scheme (otherwise read from localStorage). */
  defaultScheme?: ColorScheme;
}

export function ThemeProvider({ children, defaultAccent, defaultScheme }: ThemeProviderProps) {
  const [accent, setAccentState] = useState<AccentTheme>(defaultAccent ?? readAccent);
  const [scheme, setSchemeState] = useState<ColorScheme>(defaultScheme ?? readScheme);

  useEffect(() => {
    const root = document.documentElement;
    if (accent && accent !== 'default') root.setAttribute('data-accent', accent);
    else root.removeAttribute('data-accent');
    try { localStorage.setItem(ACCENT_KEY, accent); } catch { /* ignore */ }
  }, [accent]);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', scheme);
    try { localStorage.setItem(SCHEME_KEY, scheme === 'dark' ? '1' : '0'); } catch { /* ignore */ }
  }, [scheme]);

  const setAccent = useCallback((a: AccentTheme) => setAccentState(a), []);
  const setScheme = useCallback((s: ColorScheme) => setSchemeState(s), []);
  const toggleScheme = useCallback(
    () => setSchemeState((s) => (s === 'dark' ? 'light' : 'dark')),
    [],
  );

  return (
    <ThemeContext.Provider value={{ accent, scheme, setAccent, setScheme, toggleScheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used within <ThemeProvider>');
  return ctx;
}
