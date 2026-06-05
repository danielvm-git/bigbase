import React, { forwardRef } from 'react';
import type { ButtonVariant, ButtonSize } from './tokens';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Visual emphasis. @default 'primary' */
  variant?: ButtonVariant;
  /** Sizing modifier. @default 'md' */
  size?: ButtonSize;
  /** Shows a spinner + disables the button. Pass a string to override the label. */
  loading?: boolean | string;
  /** Leading icon (e.g. a Lucide <Plus />). */
  icon?: React.ReactNode;
  /**
   * Accessible name — REQUIRED when the button has no text children
   * (icon-only). Enforced at runtime in dev.
   */
  'aria-label'?: string;
}

/**
 * Button — primary | secondary | danger | ghost | link, in sm | md | block.
 * States (default/hover/focus/active/disabled/loading) are all driven by
 * `components.css` (.btn .btn-{variant} .btn-{size}); this component only
 * composes class names and wires the loading + a11y behavior.
 */
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'primary', size = 'md', loading = false, icon, children, className = '', disabled, ...rest },
  ref,
) {
  const isLoading = Boolean(loading);
  const cls = [
    'btn',
    `btn-${variant}`,
    size === 'sm' ? 'btn-sm' : '',
    size === 'block' ? 'btn-block' : '',
    className,
  ].filter(Boolean).join(' ');

  if (process.env.NODE_ENV !== 'production' && !children && !rest['aria-label']) {
    // eslint-disable-next-line no-console
    console.warn('[Button] icon-only buttons must pass an aria-label.');
  }

  return (
    <button ref={ref} className={cls} disabled={disabled || isLoading} aria-busy={isLoading} {...rest}>
      {isLoading
        ? <><span className="spinner spinner-sm" aria-hidden="true" />{typeof loading === 'string' ? loading : children}</>
        : <>{icon}{children}</>}
    </button>
  );
});
