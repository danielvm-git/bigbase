import type { ReactNode, HTMLAttributes } from 'react'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
}

export function Card({ children, className = '', ...rest }: CardProps) {
  return (
    <div className={`card ${className}`} {...rest}>
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
