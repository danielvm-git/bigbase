/**
 * BigBase Design System — shared token types.
 * String unions mirror the CSS custom properties in colors_and_type.css,
 * so component props can't drift from the token system.
 */

export type AccentTheme =
  | 'default'
  | 'january' | 'february' | 'march' | 'april' | 'may' | 'june'
  | 'july' | 'august' | 'september' | 'october' | 'november' | 'december';

export type ColorScheme = 'light' | 'dark';

export type StatusKind = 'ready' | 'building' | 'failed' | 'pending';

export type BadgeVariant =
  | 'neutral' | 'accent' | 'success' | 'warning' | 'error' | 'info';

export type ButtonVariant =
  | 'primary' | 'secondary' | 'danger' | 'ghost' | 'link';

export type ButtonSize = 'sm' | 'md' | 'block';

export type SpaceToken =
  | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 8 | 10 | 12 | 16 | 20 | 24 | 32;

export type RadiusToken = 'xs' | 's' | 'm' | 'l' | 'full';

export type ShadowToken = 'xs' | 's' | 'm' | 'l' | 'xl';

/** Maps a month theme to its display label + seed color (for pickers). */
export interface ThemeMeta {
  id: AccentTheme;
  label: string;
  /** CSS color or gradient for the swatch dot. */
  swatch: string;
}

export const THEME_META: ThemeMeta[] = [
  { id: 'default',   label: 'Indigo (default)',  swatch: 'rgb(79,70,229)' },
  { id: 'january',   label: 'January · Teal',     swatch: 'rgb(13,148,136)' },
  { id: 'february',  label: 'February · Orange',  swatch: 'rgb(234,88,12)' },
  { id: 'march',     label: 'March · Purple',     swatch: 'rgb(124,58,237)' },
  { id: 'april',     label: 'April · Green',      swatch: 'rgb(22,163,74)' },
  { id: 'may',       label: 'May · Lavender',     swatch: 'rgb(167,139,250)' },
  { id: 'june',      label: 'June · Rainbow',     swatch: 'linear-gradient(90deg,#F43F5E,#FB923C,#FACC15,#22C55E,#06B6D4,#6366F1,#A855F7)' },
  { id: 'july',      label: 'July · Peach',       swatch: 'rgb(253,186,116)' },
  { id: 'august',    label: 'August · Silver',    swatch: 'rgb(156,163,175)' },
  { id: 'september', label: 'September · Yellow', swatch: 'rgb(234,179,8)' },
  { id: 'october',   label: 'October · Pink',     swatch: 'rgb(236,72,153)' },
  { id: 'november',  label: 'November · Blue',     swatch: 'rgb(37,99,235)' },
  { id: 'december',  label: 'December · Red',      swatch: 'rgb(220,38,38)' },
];
