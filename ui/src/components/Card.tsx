import type { ReactNode, HTMLAttributes } from 'react'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
  /** Render as a clickable card with hover affordance. */
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
