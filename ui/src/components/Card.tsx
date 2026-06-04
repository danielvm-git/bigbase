import type { ReactNode, HTMLAttributes } from 'react'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
  /**
   * Add a hover affordance (cursor + border + shadow). The Card itself
   * stays a non-interactive `<div>` — wrap it in a `<button>` or `<a>`
   * (or pass `onClick` + `role="button"` + a key handler through
   * `...rest`) for clickable use. Decorative only by default.
   */
  interactive?: boolean
}

export function Card({
  children,
  className = '',
  interactive = false,
  ...rest
}: CardProps) {
  const cls = `card ${interactive ? 'card-interactive ' : ''}${className}`.trim()
  return (
    <div className={cls} {...rest}>
      {children}
    </div>
  )
}

interface CardHeaderProps {
  title: string
  children?: ReactNode
}

export function CardHeader({ title, children }: CardHeaderProps) {
  return (
    <div className="card-header">
      <span className="card-title">{title}</span>
      {children}
    </div>
  )
}
