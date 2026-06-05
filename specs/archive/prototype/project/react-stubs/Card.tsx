import React from 'react';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Card title rendered as a .card-title overline-style heading. */
  title?: string;
  /** Lift on hover (translateY + shadow). Only for interactive cards. */
  interactive?: boolean;
}

/** Card — surface container. Static by default; pass `interactive` for hover lift. */
export function Card({ title, interactive = false, children, className = '', ...rest }: CardProps) {
  return (
    <div className={`card ${interactive ? 'card-interactive' : ''} ${className}`} {...rest}>
      {title && <span className="card-title">{title}</span>}
      {children}
    </div>
  );
}

export interface EmptyStateProps {
  /** Icon node (Lucide), shown in the tinted chip. */
  icon: React.ReactNode;
  /** Invitation title — e.g. "Create your first site". */
  title: string;
  /** One sentence stating the payoff. */
  children: React.ReactNode;
  /** Single primary action. */
  action?: React.ReactNode;
}

/**
 * EmptyState — the zero-data invitation: tinted icon chip, invitation title,
 * payoff body, exactly one primary CTA. Use instead of a bare empty table.
 */
export function EmptyState({ icon, title, children, action }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon">{icon}</div>
      <div className="empty-state-title">{title}</div>
      <div className="empty-state-text">{children}</div>
      {action}
    </div>
  );
}
