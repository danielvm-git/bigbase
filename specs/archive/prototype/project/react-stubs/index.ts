/**
 * BigBase Design System — React component library (barrel).
 *
 * Usage:
 *   import { ThemeProvider, Button, Badge, Input, Card } from '@bigbase/ds';
 *
 *   <ThemeProvider>
 *     <Button variant="primary" icon={<Plus />}>Create site</Button>
 *   </ThemeProvider>
 *
 * Styling comes from the design-system CSS (colors_and_type.css + components.css).
 * Import it once at your app root:
 *   import '@bigbase/ds/styles.css';
 */

export * from './tokens';
export { ThemeProvider, useTheme } from './ThemeContext';
export type { ThemeContextValue, ThemeProviderProps } from './ThemeContext';
export { Button } from './Button';
export type { ButtonProps } from './Button';
export { Badge, StatusBadge } from './Badge';
export type { BadgeProps, StatusBadgeProps } from './Badge';
export { Input } from './Input';
export type { InputProps } from './Input';
export { Card, EmptyState } from './Card';
export type { CardProps, EmptyStateProps } from './Card';
export { ThemePicker } from './ThemePicker';
