import React from 'react';
import type { BadgeVariant, StatusKind } from './tokens';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** @default 'neutral' */
  variant?: BadgeVariant;
  /** Render a leading status dot. */
  dot?: boolean;
}

/** Badge — compact semantic label. Background is a ~10–12% tint of the variant. */
export function Badge({ variant = 'neutral', dot = false, children, className = '', ...rest }: BadgeProps) {
  return (
    <span className={`badge badge-${variant} ${className}`} {...rest}>
      {dot && <span className="dot" aria-hidden="true" />}
      {children}
    </span>
  );
}

const STATUS_MAP: Record<StatusKind, { variant: BadgeVariant; label: string; spinner?: boolean }> = {
  ready:    { variant: 'success', label: 'Ready' },
  building: { variant: 'warning', label: 'Building', spinner: true },
  failed:   { variant: 'error',   label: 'Failed' },
  pending:  { variant: 'warning', label: 'Pending' },
};

export interface StatusBadgeProps {
  status: StatusKind;
  /** Override the default word (Ready / Building / Failed / Pending). */
  label?: string;
}

/**
 * StatusBadge — never communicates by color alone: always a word + a
 * dot (or spinner while building) + the semantic color.
 */
export function StatusBadge({ status, label }: StatusBadgeProps) {
  const m = STATUS_MAP[status];
  return (
    <Badge variant={m.variant} dot={!m.spinner}>
      {m.spinner && <span className="spinner spinner-sm" style={{ width: 10, height: 10, borderWidth: 1.5 }} aria-hidden="true" />}
      {label ?? m.label}
    </Badge>
  );
}
